package documents

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
	"github.com/msalomonn/documind/internal/auth"
	"io"
	"mime"
	"path/filepath"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const MaxBytes = 10 * 1024 * 1024

var ErrConflict = errors.New("idempotency_conflict")
var ErrNotFound = errors.New("not_found")
var ErrStorage = errors.New("storage_unavailable")

type Document struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Size           int64     `json:"size"`
	ContentType    string    `json:"contentType"`
	Status         string    `json:"status"`
	ErrorCode      string    `json:"errorCode,omitempty"`
	ResultProvider string    `json:"resultProvider,omitempty"`
	Generation     int       `json:"generation"`
	Attempts       int       `json:"attempts"`
	CreatedAt      time.Time `json:"createdAt"`
	ObjectKey      string    `json:"-"`
	Hash           string    `json:"-"`
}
type Objects interface {
	Put(context.Context, string, []byte) error
	Open(context.Context, string) (io.ReadCloser, error)
	Delete(context.Context, string) error
}
type S3 struct {
	Client *minio.Client
	Bucket string
}

func (s S3) Put(ctx context.Context, key string, b []byte) error {
	_, e := s.Client.PutObject(ctx, s.Bucket, key, bytes.NewReader(b), int64(len(b)), minio.PutObjectOptions{ContentType: "text/plain; charset=utf-8"})
	if e != nil {
		return e
	}
	info, e := s.Client.StatObject(ctx, s.Bucket, key, minio.StatObjectOptions{})
	if e != nil {
		return e
	}
	if info.Size != int64(len(b)) {
		return ErrStorage
	}
	return nil
}
func (s S3) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	o, e := s.Client.GetObject(ctx, s.Bucket, key, minio.GetObjectOptions{})
	if e != nil {
		return nil, e
	}
	if _, e = o.Stat(); e != nil {
		o.Close()
		return nil, e
	}
	return o, nil
}
func (s S3) Delete(ctx context.Context, key string) error {
	return s.Client.RemoveObject(ctx, s.Bucket, key, minio.RemoveObjectOptions{})
}

type Service struct {
	DB      *pgxpool.Pool
	Objects Objects
}

func Validate(name, contentType string, b []byte) string {
	if len(b) > MaxBytes {
		return "file_too_large"
	}
	ext := strings.ToLower(filepath.Ext(name))
	if ext == ".pdf" || bytes.HasPrefix(bytes.TrimSpace(b), []byte("%PDF-")) {
		return "pdf_not_enabled"
	}
	mt, _, e := mime.ParseMediaType(contentType)
	if e != nil || ext != ".txt" || (mt != "text/plain" && mt != "application/octet-stream") || len(name) > 180 {
		return "unsupported_file"
	}
	if len(bytes.TrimSpace(b)) == 0 || !utf8.Valid(b) {
		return "invalid_text"
	}
	if utf8.RuneCount(b) > 200000 {
		return "text_too_long"
	}
	for _, r := range string(b) {
		if unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t' {
			return "invalid_text"
		}
	}
	for _, r := range name {
		if unicode.IsControl(r) {
			return "unsupported_file"
		}
	}
	return ""
}
func ValidKey(key string) bool {
	if len(key) < 16 || len(key) > 128 {
		return false
	}
	for _, c := range key {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_') {
			return false
		}
	}
	return true
}

const columns = `d.id,d.name,d.size,d.content_type,j.state,j.generation,j.attempts,d.created_at,d.object_key,d.content_hash,COALESCE(j.error_code,''),COALESCE((SELECT provider FROM documind.results r WHERE r.document_id=d.id AND r.generation=j.generation),'')`
const join = ` FROM documind.documents d JOIN LATERAL (SELECT * FROM documind.jobs WHERE document_id=d.id ORDER BY generation DESC LIMIT 1) j ON true `

func scan(row pgx.Row) (Document, error) {
	var d Document
	e := row.Scan(&d.ID, &d.Name, &d.Size, &d.ContentType, &d.Status, &d.Generation, &d.Attempts, &d.CreatedAt, &d.ObjectKey, &d.Hash, &d.ErrorCode, &d.ResultProvider)
	return d, e
}
func (s *Service) Register(ctx context.Context, owner, key, name string, b []byte) (Document, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	sum := sha256.Sum256(b)
	hash := hex.EncodeToString(sum[:])
	objectKey := "uploads/" + auth.Random()
	// Persist before acquiring a transaction connection: concurrent uploads cannot exhaust the pool waiting for an intent connection.
	if _, e := s.DB.Exec(ctx, "INSERT INTO documind.upload_intents(object_key) VALUES($1)", objectKey); e != nil {
		return Document{}, e
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return Document{}, e
	}
	defer tx.Rollback(context.Background())
	// A DB uniqueness constraint also protects against callers that omit the advisory lock.
	_, e = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1,0))", owner+":"+key)
	if e != nil {
		return Document{}, e
	}
	d, e := scan(tx.QueryRow(ctx, "SELECT "+columns+join+"WHERE d.owner_id=$1 AND d.idempotency_key=$2", owner, key))
	if e == nil {
		if d.Hash != hash {
			return Document{}, ErrConflict
		}
		if _, e = tx.Exec(ctx, "DELETE FROM documind.upload_intents WHERE object_key=$1", objectKey); e != nil {
			return Document{}, e
		}
		if e = tx.Commit(ctx); e != nil {
			return Document{}, e
		}
		return d, nil
	}
	if !errors.Is(e, pgx.ErrNoRows) {
		return Document{}, e
	}
	if e = s.Objects.Put(ctx, objectKey, b); e != nil {
		return Document{}, ErrStorage
	}
	id := auth.Random()
	_, e = tx.Exec(ctx, `INSERT INTO documind.documents(id,owner_id,idempotency_key,object_key,content_hash,name,content_type,size) VALUES($1,$2,$3,$4,$5,$6,'text/plain; charset=utf-8',$7)`, id, owner, key, objectKey, hash, name, len(b))
	if e != nil {
		return Document{}, e
	}
	_, e = tx.Exec(ctx, "INSERT INTO documind.jobs(document_id,generation,state) VALUES($1,1,'queued')", id)
	if e != nil {
		return Document{}, e
	}
	_, e = tx.Exec(ctx, "DELETE FROM documind.upload_intents WHERE object_key=$1", objectKey)
	if e != nil {
		return Document{}, e
	}
	d, e = scan(tx.QueryRow(ctx, "SELECT "+columns+join+"WHERE d.id=$1", id))
	if e != nil {
		return Document{}, e
	}
	if e = tx.Commit(ctx); e != nil {
		return Document{}, e
	}
	return d, nil
}
func (s *Service) List(ctx context.Context, owner string) ([]Document, error) {
	rows, e := s.DB.Query(ctx, "SELECT "+columns+join+"WHERE d.owner_id=$1 ORDER BY d.created_at DESC,d.id LIMIT 100", owner)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	docs := []Document{}
	for rows.Next() {
		d, e := scan(rows)
		if e != nil {
			return nil, e
		}
		docs = append(docs, d)
	}
	return docs, rows.Err()
}
func (s *Service) Get(ctx context.Context, owner, id string) (Document, error) {
	d, e := scan(s.DB.QueryRow(ctx, "SELECT "+columns+join+"WHERE d.owner_id=$1 AND d.id=$2", owner, id))
	if errors.Is(e, pgx.ErrNoRows) {
		e = ErrNotFound
	}
	return d, e
}

// Cleanup processes a bounded batch. The one-hour grace exceeds the upload's hard deadline.
func (s *Service) Cleanup(ctx context.Context) (int, error) {
	count := 0
	for range 100 {
		tx, e := s.DB.Begin(ctx)
		if e != nil {
			return count, e
		}
		var key string
		e = tx.QueryRow(ctx, `SELECT i.object_key FROM documind.upload_intents i WHERE i.created_at<now()-interval '1 hour' AND NOT EXISTS(SELECT 1 FROM documind.documents d WHERE d.object_key=i.object_key) ORDER BY i.created_at LIMIT 1 FOR UPDATE OF i SKIP LOCKED`).Scan(&key)
		if errors.Is(e, pgx.ErrNoRows) {
			tx.Rollback(ctx)
			return count, nil
		}
		if e == nil {
			e = s.Objects.Delete(ctx, key)
		}
		if e == nil {
			_, e = tx.Exec(ctx, "DELETE FROM documind.upload_intents WHERE object_key=$1", key)
		}
		if e != nil {
			tx.Rollback(ctx)
			return count, e
		}
		if e = tx.Commit(ctx); e != nil {
			return count, e
		}
		count++
	}
	return count, nil
}
