package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/msalomonn/documind/internal/auth"
	"time"
)

type Claim struct {
	DocumentID                          string
	Generation, Attempts                int
	Token, ObjectKey, Name, ContentType string
	Deadline                            time.Time
}
type Queue struct {
	DB     *pgxpool.Pool
	Config Config
}

func New(db *pgxpool.Pool, c Config) (*Queue, error) {
	if !c.Valid() {
		return nil, errors.New("invalid worker timing configuration")
	}
	return &Queue{db, c}, nil
}
func bounded(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, 5*time.Second)
}
func rollback(tx pgx.Tx) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_ = tx.Rollback(ctx)
}

// Claim commits before processor I/O; documentID optionally scopes a diagnostic/test worker.
func (q *Queue) Claim(ctx context.Context, documentID string) (Claim, error) {
	ctx, cancel := bounded(ctx)
	defer cancel()
	tx, e := q.DB.Begin(ctx)
	if e != nil {
		return Claim{}, e
	}
	defer rollback(tx)
	var c Claim
	e = tx.QueryRow(ctx, `SELECT j.document_id,j.generation,j.attempts,d.object_key,d.name,d.content_type FROM documind.jobs j JOIN documind.documents d ON d.id=j.document_id WHERE ($1='' OR j.document_id=$1) AND ((j.state IN ('queued','retry_wait') AND j.next_run_at<=clock_timestamp()) OR (j.state='processing' AND j.lease_expires_at<=clock_timestamp())) ORDER BY j.next_run_at,j.created_at,j.document_id LIMIT 1 FOR UPDATE OF j SKIP LOCKED`, documentID).Scan(&c.DocumentID, &c.Generation, &c.Attempts, &c.ObjectKey, &c.Name, &c.ContentType)
	if errors.Is(e, pgx.ErrNoRows) {
		return c, ErrEmpty
	}
	if e != nil {
		return c, e
	}
	if c.Attempts >= 3 {
		_, e = tx.Exec(ctx, `UPDATE documind.jobs SET state='failed',error_code='lease_expired_exhausted',claim_token=NULL,lease_expires_at=NULL,attempt_deadline=NULL WHERE document_id=$1 AND generation=$2`, c.DocumentID, c.Generation)
		if e != nil {
			return Claim{}, e
		}
		if e = tx.Commit(ctx); e != nil {
			return Claim{}, e
		}
		return Claim{}, ErrEmpty
	}
	c.Token = auth.Random()
	e = tx.QueryRow(ctx, `UPDATE documind.jobs SET state='processing',attempts=attempts+1,claim_token=$3,lease_expires_at=clock_timestamp()+$4*interval '1 millisecond',attempt_deadline=clock_timestamp()+$5*interval '1 millisecond',error_code=NULL WHERE document_id=$1 AND generation=$2 RETURNING attempts,attempt_deadline`, c.DocumentID, c.Generation, c.Token, q.Config.Lease.Milliseconds(), q.Config.Attempt.Milliseconds()).Scan(&c.Attempts, &c.Deadline)
	if e != nil {
		return Claim{}, e
	}
	if e = tx.Commit(ctx); e != nil {
		return Claim{}, e
	}
	return c, nil
}

// Lock first, then check wall time in a separate statement, including after a lock wait.
func (q *Queue) lock(ctx context.Context, c Claim) (pgx.Tx, error) {
	tx, e := q.DB.Begin(ctx)
	if e != nil {
		return nil, e
	}
	var exists int
	e = tx.QueryRow(ctx, `SELECT 1 FROM documind.jobs WHERE document_id=$1 AND generation=$2 FOR UPDATE`, c.DocumentID, c.Generation).Scan(&exists)
	if e != nil {
		rollback(tx)
		if errors.Is(e, pgx.ErrNoRows) {
			return nil, ErrLost
		}
		return nil, e
	}
	return tx, nil
}

const owned = `document_id=$1 AND generation=$2 AND state='processing' AND claim_token=$3 AND lease_expires_at>clock_timestamp() AND attempt_deadline>clock_timestamp()`

func (q *Queue) Renew(ctx context.Context, c Claim) error {
	ctx, cancel := bounded(ctx)
	defer cancel()
	tx, e := q.lock(ctx, c)
	if e != nil {
		return e
	}
	defer rollback(tx)
	tag, e := tx.Exec(ctx, `UPDATE documind.jobs SET lease_expires_at=LEAST(attempt_deadline,clock_timestamp()+$4*interval '1 millisecond') WHERE `+owned, c.DocumentID, c.Generation, c.Token, q.Config.Lease.Milliseconds())
	if e != nil {
		return e
	}
	if tag.RowsAffected() != 1 {
		return ErrLost
	}
	return tx.Commit(ctx)
}
func (q *Queue) Complete(ctx context.Context, c Claim, o Outcome) error {
	if !validOutcome(o) {
		return ErrInvalidResult
	}
	ctx, cancel := bounded(ctx)
	defer cancel()
	tx, e := q.lock(ctx, c)
	if e != nil {
		return e
	}
	defer rollback(tx)
	tag, e := tx.Exec(ctx, `UPDATE documind.jobs SET state='succeeded',claim_token=NULL,lease_expires_at=NULL,attempt_deadline=NULL,error_code=NULL WHERE `+owned, c.DocumentID, c.Generation, c.Token)
	if e != nil {
		return e
	}
	if tag.RowsAffected() != 1 {
		return ErrLost
	}
	data, e := json.Marshal(o.Result)
	if e != nil {
		return e
	}
	_, e = tx.Exec(ctx, `INSERT INTO documind.results(document_id,generation,schema_version,result,provider,model) VALUES($1,$2,$3,$4,$5,$6)`, c.DocumentID, c.Generation, o.Result.SchemaVersion, data, o.Provider, o.Model)
	if e != nil {
		return e
	}
	return tx.Commit(ctx)
}
func (q *Queue) Fail(ctx context.Context, c Claim, f Failure) error {
	ctx, cancel := bounded(ctx)
	defer cancel()
	tx, e := q.lock(ctx, c)
	if e != nil {
		return e
	}
	defer rollback(tx)
	var attempts int
	e = tx.QueryRow(ctx, `SELECT attempts FROM documind.jobs WHERE `+owned, c.DocumentID, c.Generation, c.Token).Scan(&attempts)
	if errors.Is(e, pgx.ErrNoRows) {
		return ErrLost
	}
	if e != nil {
		return e
	}
	state := "failed"
	delay := time.Duration(0)
	if f.Transient && attempts < 3 {
		state = "retry_wait"
		delay = RetryDelay(attempts, f.RetryAfter)
	}
	tag, e := tx.Exec(ctx, `UPDATE documind.jobs SET state=$4,error_code=$5,next_run_at=clock_timestamp()+$6*interval '1 millisecond',claim_token=NULL,lease_expires_at=NULL,attempt_deadline=NULL WHERE `+owned, c.DocumentID, c.Generation, c.Token, state, safeCode(f.Code), delay.Milliseconds())
	if e != nil {
		return e
	}
	if tag.RowsAffected() != 1 {
		return ErrLost
	}
	return tx.Commit(ctx)
}
