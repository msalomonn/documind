package integration

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/msalomonn/documind/db"
	"github.com/msalomonn/documind/internal/auth"
	"github.com/msalomonn/documind/internal/config"
	"github.com/msalomonn/documind/internal/documents"
	"github.com/msalomonn/documind/internal/jobs"
	"github.com/msalomonn/documind/internal/pipeline"
	"github.com/msalomonn/documind/internal/platform"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"testing"
	"time"
)

type processorFunc func(context.Context, jobs.Claim) (jobs.Outcome, error)

func (f processorFunc) Process(c context.Context, j jobs.Claim) (jobs.Outcome, error) { return f(c, j) }
func TestWorkerLifecycle(t *testing.T) {
	if os.Getenv("DOCUMIND_INTEGRATION") != "1" {
		t.Skip("requires real PostgreSQL/S3")
	}
	ctx := context.Background()
	cfg, e := config.Load()
	must(t, e)
	must(t, db.Migrate(ctx, os.Getenv("DATABASE_ADMIN_URL")))
	admin, e := pgxpool.New(ctx, os.Getenv("DATABASE_ADMIN_URL"))
	must(t, e)
	defer admin.Close()
	p, e := platform.Open(ctx, cfg)
	must(t, e)
	defer p.DB.Close()
	objects := documents.S3{Client: p.S3, Bucket: p.Bucket}
	docs := documents.Service{DB: p.DB, Objects: objects}
	owner := auth.Random()
	_, e = admin.Exec(ctx, `INSERT INTO documind.users(id,email,password_salt,password_hash) VALUES($1,$2,$3,$4)`, owner, "worker-"+owner+"@example.test", make([]byte, 16), make([]byte, 32))
	must(t, e)
	created := []documents.Document{}
	defer func() {
		for _, d := range created {
			objects.Delete(ctx, d.ObjectKey)
			for _, table := range []string{"results", "jobs", "documents"} {
				col := "document_id"
				if table == "documents" {
					col = "id"
				}
				if _, e := admin.Exec(ctx, "DELETE FROM documind."+table+" WHERE "+col+"=$1", d.ID); e != nil {
					t.Error(e)
				}
			}
		}
		_, e := admin.Exec(ctx, "DELETE FROM documind.users WHERE id=$1", owner)
		if e != nil {
			t.Error(e)
		}
	}()
	fixture := func(t *testing.T) documents.Document {
		t.Helper()
		d, e := docs.Register(ctx, owner, auth.Random(), "worker-smoke.txt", []byte(jobs.SmokePrefix+"Synthetic only.\n"))
		must(t, e)
		created = append(created, d)
		return d
	}
	options := jobs.Config{Lease: 2 * time.Second, Heartbeat: 300 * time.Millisecond, Attempt: 10 * time.Second, Poll: 20 * time.Millisecond}
	queue, e := jobs.New(p.DB, options)
	must(t, e)
	outcome := jobs.Outcome{Result: pipeline.Result{SchemaVersion: 1, Title: "Synthetic", Summary: "No AI calls", KeyFacts: []pipeline.Fact{}}, Provider: "diagnostic", Model: "worker-smoke-v1"}
	expire := func(t *testing.T, d string) {
		_, e := admin.Exec(ctx, "UPDATE documind.jobs SET lease_expires_at=clock_timestamp()-interval '1 second' WHERE document_id=$1", d)
		must(t, e)
	}
	state := func(t *testing.T, id string) (string, int, string) {
		t.Helper()
		var s, code string
		var n int
		must(t, p.DB.QueryRow(ctx, "SELECT state,attempts,COALESCE(error_code,'') FROM documind.jobs WHERE document_id=$1", id).Scan(&s, &n, &code))
		return s, n, code
	}
	t.Run("eight competing consumers claim one job", func(t *testing.T) {
		d := fixture(t)
		var wg sync.WaitGroup
		claims := make(chan jobs.Claim, 8)
		errs := make(chan error, 8)
		for range 8 {
			wg.Add(1)
			go func() { defer wg.Done(); c, e := queue.Claim(ctx, d.ID); claims <- c; errs <- e }()
		}
		wg.Wait()
		close(claims)
		close(errs)
		count := 0
		for e := range errs {
			if e == nil {
				count++
			} else if !errors.Is(e, jobs.ErrEmpty) {
				t.Fatal(e)
			}
		}
		if count != 1 {
			t.Fatalf("claims=%d", count)
		}
		for c := range claims {
			if c.Token != "" {
				must(t, queue.Complete(ctx, c, outcome))
			}
		}
	})
	t.Run("locked and future jobs are skipped", func(t *testing.T) {
		d := fixture(t)
		tx, e := admin.Begin(ctx)
		must(t, e)
		defer tx.Rollback(ctx)
		_, e = tx.Exec(ctx, "SELECT 1 FROM documind.jobs WHERE document_id=$1 FOR UPDATE", d.ID)
		must(t, e)
		_, e = queue.Claim(ctx, d.ID)
		if !errors.Is(e, jobs.ErrEmpty) {
			t.Fatalf("locked claim: %v", e)
		}
		must(t, tx.Rollback(ctx))
		_, e = admin.Exec(ctx, "UPDATE documind.jobs SET state='retry_wait',next_run_at=clock_timestamp()+interval '1 minute' WHERE document_id=$1", d.ID)
		must(t, e)
		_, e = queue.Claim(ctx, d.ID)
		if !errors.Is(e, jobs.ErrEmpty) {
			t.Fatal("future retry claimed")
		}
	})
	t.Run("expired claims are fenced across all mutations", func(t *testing.T) {
		d := fixture(t)
		old, e := queue.Claim(ctx, d.ID)
		must(t, e)
		expire(t, d.ID)
		fresh, e := queue.Claim(ctx, d.ID)
		must(t, e)
		if old.Token == fresh.Token || fresh.Attempts != 2 {
			t.Fatal("claim token/attempt not rotated")
		}
		for _, e := range []error{queue.Renew(ctx, old), queue.Fail(ctx, old, jobs.Failure{Code: "timeout", Transient: true}), queue.Complete(ctx, old, outcome)} {
			if !errors.Is(e, jobs.ErrLost) {
				t.Fatalf("stale mutation: %v", e)
			}
		}
		must(t, queue.Complete(ctx, fresh, outcome))
		if !errors.Is(queue.Complete(ctx, fresh, outcome), jobs.ErrLost) {
			t.Fatal("duplicate completion accepted")
		}
		var n int
		must(t, p.DB.QueryRow(ctx, "SELECT count(*) FROM documind.results WHERE document_id=$1", d.ID).Scan(&n))
		if n != 1 {
			t.Fatal("result uniqueness violated")
		}
	})
	t.Run("heartbeat is capped at hard deadline", func(t *testing.T) {
		d := fixture(t)
		c, e := queue.Claim(ctx, d.ID)
		must(t, e)
		_, e = admin.Exec(ctx, "UPDATE documind.jobs SET lease_expires_at=clock_timestamp()+interval '200 milliseconds',attempt_deadline=clock_timestamp()+interval '400 milliseconds' WHERE document_id=$1", d.ID)
		must(t, e)
		must(t, queue.Renew(ctx, c))
		var capped bool
		must(t, p.DB.QueryRow(ctx, "SELECT lease_expires_at=attempt_deadline FROM documind.jobs WHERE document_id=$1", d.ID).Scan(&capped))
		if !capped {
			t.Fatal("renewal exceeded hard deadline")
		}
	})
	t.Run("completion rechecks time after waiting for a row lock", func(t *testing.T) {
		d := fixture(t)
		short := options
		short.Lease = 200 * time.Millisecond
		short.Heartbeat = 50 * time.Millisecond
		q, e := jobs.New(p.DB, short)
		must(t, e)
		c, e := q.Claim(ctx, d.ID)
		must(t, e)
		tx, e := admin.Begin(ctx)
		must(t, e)
		defer tx.Rollback(ctx)
		_, e = tx.Exec(ctx, "SELECT 1 FROM documind.jobs WHERE document_id=$1 FOR UPDATE", d.ID)
		must(t, e)
		done := make(chan error, 1)
		go func() { done <- q.Complete(ctx, c, outcome) }()
		time.Sleep(300 * time.Millisecond)
		must(t, tx.Commit(ctx))
		if e = <-done; !errors.Is(e, jobs.ErrLost) {
			t.Fatalf("expired waiter completed: %v", e)
		}
	})
	t.Run("transient retries are bounded and permanent failures stop", func(t *testing.T) {
		d := fixture(t)
		for attempt := 1; attempt <= 3; attempt++ {
			c, e := queue.Claim(ctx, d.ID)
			must(t, e)
			must(t, queue.Fail(ctx, c, jobs.Failure{Code: "rate_limited", Transient: true, RetryAfter: time.Hour}))
			s, n, code := state(t, d.ID)
			if n != attempt || code != "rate_limited" {
				t.Fatal("attempt/error mismatch")
			}
			if attempt < 3 {
				if s != "retry_wait" {
					t.Fatal(s)
				}
				var seconds float64
				must(t, p.DB.QueryRow(ctx, "SELECT extract(epoch FROM next_run_at-clock_timestamp()) FROM documind.jobs WHERE document_id=$1", d.ID).Scan(&seconds))
				if seconds < 58 || seconds > 60 {
					t.Fatalf("retry-after unbounded: %v", seconds)
				}
				_, e = queue.Claim(ctx, d.ID)
				if !errors.Is(e, jobs.ErrEmpty) {
					t.Fatal("early retry")
				}
				_, e = admin.Exec(ctx, "UPDATE documind.jobs SET next_run_at=clock_timestamp()-interval '1 second' WHERE document_id=$1", d.ID)
				must(t, e)
			} else if s != "failed" {
				t.Fatal(s)
			}
		}
		_, e = queue.Claim(ctx, d.ID)
		if !errors.Is(e, jobs.ErrEmpty) {
			t.Fatal("fourth attempt")
		}
		d = fixture(t)
		c, e := queue.Claim(ctx, d.ID)
		must(t, e)
		must(t, queue.Fail(ctx, c, jobs.Failure{Code: "untrusted secret text"}))
		s, n, code := state(t, d.ID)
		if s != "failed" || n != 1 || code != "processor_failed" {
			t.Fatal("unsafe permanent failure")
		}
	})
	t.Run("third crashed attempt becomes terminal", func(t *testing.T) {
		d := fixture(t)
		for range 3 {
			_, e := queue.Claim(ctx, d.ID)
			must(t, e)
			expire(t, d.ID)
		}
		_, e := queue.Claim(ctx, d.ID)
		if !errors.Is(e, jobs.ErrEmpty) {
			t.Fatal(e)
		}
		s, n, code := state(t, d.ID)
		if s != "failed" || n != 3 || code != "lease_expired_exhausted" {
			t.Fatalf("stranded job: %s %d %s", s, n, code)
		}
	})
	t.Run("result commit and state transition are atomic", func(t *testing.T) {
		d := fixture(t)
		c, e := queue.Claim(ctx, d.ID)
		must(t, e)
		_, e = admin.Exec(ctx, `CREATE FUNCTION documind.test_result_commit() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.document_id='`+d.ID+`' THEN RAISE EXCEPTION 'synthetic result commit failure'; END IF; RETURN NEW; END $$; CREATE CONSTRAINT TRIGGER test_result_commit AFTER INSERT ON documind.results DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION documind.test_result_commit();`)
		must(t, e)
		defer func() {
			_, e := admin.Exec(ctx, "DROP TRIGGER IF EXISTS test_result_commit ON documind.results; DROP FUNCTION IF EXISTS documind.test_result_commit()")
			must(t, e)
		}()
		if queue.Complete(ctx, c, outcome) == nil {
			t.Fatal("commit fault ignored")
		}
		s, _, _ := state(t, d.ID)
		if s != "processing" {
			t.Fatal("partial succeeded state")
		}
		var n int
		must(t, p.DB.QueryRow(ctx, "SELECT count(*) FROM documind.results WHERE document_id=$1", d.ID).Scan(&n))
		if n != 0 {
			t.Fatal("partial result")
		}
		_, e = admin.Exec(ctx, "DROP TRIGGER test_result_commit ON documind.results; DROP FUNCTION documind.test_result_commit()")
		must(t, e)
		must(t, queue.Complete(ctx, c, outcome))
	})
	t.Run("runner renews a slow processor and rejects malformed results", func(t *testing.T) {
		d := fixture(t)
		short := options
		short.Lease = 500 * time.Millisecond
		short.Heartbeat = 100 * time.Millisecond
		q, e := jobs.New(p.DB, short)
		must(t, e)
		r := jobs.Runner{Queue: q, DocumentID: d.ID, Processor: processorFunc(func(c context.Context, _ jobs.Claim) (jobs.Outcome, error) {
			select {
			case <-c.Done():
				return jobs.Outcome{}, c.Err()
			case <-time.After(1100 * time.Millisecond):
				return outcome, nil
			}
		})}
		must(t, r.Once(ctx))
		s, _, _ := state(t, d.ID)
		if s != "succeeded" {
			t.Fatal("heartbeat did not preserve claim")
		}
		d = fixture(t)
		r.DocumentID = d.ID
		r.Processor = processorFunc(func(context.Context, jobs.Claim) (jobs.Outcome, error) { return jobs.Outcome{}, nil })
		must(t, r.Once(ctx))
		s, _, code := state(t, d.ID)
		if s != "failed" || code != "invalid_result" {
			t.Fatal("invalid result accepted")
		}
	})
	t.Run("processor deadline schedules a retry within the lease", func(t *testing.T) {
		d := fixture(t)
		short := jobs.Config{Lease: 600 * time.Millisecond, Heartbeat: 100 * time.Millisecond, Attempt: time.Second, Poll: 20 * time.Millisecond}
		q, e := jobs.New(p.DB, short)
		must(t, e)
		r := jobs.Runner{Queue: q, DocumentID: d.ID, Processor: processorFunc(func(c context.Context, _ jobs.Claim) (jobs.Outcome, error) {
			<-c.Done()
			return jobs.Outcome{}, c.Err()
		})}
		must(t, r.Once(ctx))
		s, _, code := state(t, d.ID)
		if s != "retry_wait" || code != "timeout" {
			t.Fatalf("timeout: %s %s", s, code)
		}
	})
	t.Run("heartbeat loss cancels processor without stale writes", func(t *testing.T) {
		d := fixture(t)
		cancelled := make(chan struct{})
		r := jobs.Runner{Queue: queue, DocumentID: d.ID, Processor: processorFunc(func(c context.Context, _ jobs.Claim) (jobs.Outcome, error) {
			expire(t, d.ID)
			<-c.Done()
			close(cancelled)
			return outcome, nil
		})}
		e := r.Once(ctx)
		if !errors.Is(e, jobs.ErrLost) {
			t.Fatalf("heartbeat loss: %v", e)
		}
		select {
		case <-cancelled:
		default:
			t.Fatal("processor not cancelled")
		}
		s, _, _ := state(t, d.ID)
		if s != "processing" {
			t.Fatal("stale worker wrote state")
		}
	})
	for _, signal := range []syscall.Signal{syscall.SIGKILL, syscall.SIGTERM} {
		t.Run("real worker restart after "+signal.String(), func(t *testing.T) {
			d := fixture(t)
			binary := filepath.Join(t.TempDir(), "documind-worker")
			build := exec.Command("go", "build", "-o", binary, "./cmd/worker")
			build.Dir = "../.."
			if out, e := build.CombinedOutput(); e != nil {
				t.Fatalf("build worker: %v %s", e, out)
			}
			args := []string{"--smoke-document", d.ID, "--once", "--lease", "2s", "--heartbeat", "300ms", "--attempt-timeout", "10s"}
			first := exec.Command(binary, append(args, "--smoke-delay", "8s")...)
			must(t, first.Start())
			defer func() { _ = first.Process.Kill(); _ = first.Wait() }()
			deadline := time.Now().Add(8 * time.Second)
			for {
				s, _, _ := state(t, d.ID)
				if s == "processing" {
					break
				}
				if time.Now().After(deadline) {
					t.Fatal("worker did not claim")
				}
				time.Sleep(30 * time.Millisecond)
			}
			var old jobs.Claim
			old.DocumentID = d.ID
			old.Generation = 1
			must(t, p.DB.QueryRow(ctx, "SELECT claim_token FROM documind.jobs WHERE document_id=$1", d.ID).Scan(&old.Token))
			must(t, first.Process.Signal(signal))
			wait := make(chan error, 1)
			go func() { wait <- first.Wait() }()
			select {
			case err := <-wait:
				if signal == syscall.SIGTERM {
					must(t, err)
				}
			case <-time.After(3 * time.Second):
				t.Fatal("worker did not stop after signal")
			}
			s, n, _ := state(t, d.ID)
			if s != "processing" || n != 1 {
				t.Fatal("shutdown mutated the in-flight job")
			}
			// Wait for real PostgreSQL lease expiry; do not modify the clock/lease in this process test.
			deadline = time.Now().Add(5 * time.Second)
			for {
				var expired bool
				must(t, p.DB.QueryRow(ctx, "SELECT lease_expires_at<=clock_timestamp() FROM documind.jobs WHERE document_id=$1", d.ID).Scan(&expired))
				if expired {
					break
				}
				if time.Now().After(deadline) {
					t.Fatal("lease did not expire")
				}
				time.Sleep(30 * time.Millisecond)
			}
			restart := exec.Command(binary, args...)
			if out, e := restart.CombinedOutput(); e != nil {
				t.Fatalf("restart: %v %s", e, out)
			}
			s, n, _ = state(t, d.ID)
			if s != "succeeded" || n != 2 {
				t.Fatalf("recovery: %s %d", s, n)
			}
			if !errors.Is(queue.Complete(ctx, old, outcome), jobs.ErrLost) {
				t.Fatal("killed worker token accepted")
			}
		})
	}
}
