package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/msalomonn/documind/db"
	"github.com/msalomonn/documind/internal/auth"
	"github.com/msalomonn/documind/internal/config"
	"github.com/msalomonn/documind/internal/documents"
	"github.com/msalomonn/documind/internal/httpapi"
	"github.com/msalomonn/documind/internal/platform"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

const origin = "http://127.0.0.1:4320"

func TestPrivateUpload(t *testing.T) {
	if os.Getenv("DOCUMIND_INTEGRATION") != "1" {
		t.Skip("requires real local PostgreSQL and S3")
	}
	ctx := context.Background()
	cfg, e := config.Load()
	must(t, e)
	must(t, db.Migrate(ctx, os.Getenv("DATABASE_ADMIN_URL")))
	admin, e := pgxpool.New(ctx, os.Getenv("DATABASE_ADMIN_URL"))
	must(t, e)
	defer admin.Close()
	clients, e := platform.Open(ctx, cfg)
	must(t, e)
	defer clients.DB.Close()
	objects := documents.S3{Client: clients.S3, Bucket: clients.Bucket}
	service := &documents.Service{DB: clients.DB, Objects: objects}
	identity := auth.New(clients.DB)
	handler := httpapi.NewApplication(identity, service, origin, httpapi.New(clients.DB.Ping, clients.StorageReady))
	request := func(method, path, body, session, source string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Origin", source)
		if session != "" {
			r.AddCookie(&http.Cookie{Name: "documind_session", Value: session})
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	type account struct {
		user         auth.User
		token, email string
	}
	users := []account{}
	defer func() {
		for _, a := range users {
			rows, e := clients.DB.Query(ctx, "SELECT object_key FROM documind.documents WHERE owner_id=$1", a.user.ID)
			if e == nil {
				for rows.Next() {
					var key string
					rows.Scan(&key)
					objects.Delete(ctx, key)
				}
				rows.Close()
			}
			_, e = admin.Exec(ctx, `DELETE FROM documind.results WHERE document_id IN(SELECT id FROM documind.documents WHERE owner_id=$1);`, a.user.ID)
			if e != nil {
				t.Error(e)
			}
			for _, q := range []string{"DELETE FROM documind.jobs WHERE document_id IN(SELECT id FROM documind.documents WHERE owner_id=$1)", "DELETE FROM documind.documents WHERE owner_id=$1", "DELETE FROM documind.users WHERE id=$1"} {
				_, e = admin.Exec(ctx, q, a.user.ID)
				if e != nil {
					t.Error(e)
				}
			}
		}
	}()
	create := func() account {
		email := "test-" + auth.Random() + "@example.test"
		w := request("POST", "/api/auth/register", fmt.Sprintf(`{"email":%q,"password":"Synthetic-password-123"}`, email), "", origin)
		status(t, w, 201)
		var u auth.User
		must(t, json.Unmarshal(w.Body.Bytes(), &u))
		cookie := w.Result().Cookies()[0]
		if !cookie.HttpOnly || cookie.SameSite != http.SameSiteLaxMode || cookie.MaxAge != 28800 {
			t.Fatal("unsafe session cookie")
		}
		a := account{u, cookie.Value, email}
		users = append(users, a)
		return a
	}
	a, b := create(), create()
	upload := func(name, body, key, token string) *httptest.ResponseRecorder {
		var data bytes.Buffer
		form := multipart.NewWriter(&data)
		file, e := form.CreateFormFile("file", name)
		must(t, e)
		_, e = io.WriteString(file, body)
		must(t, e)
		must(t, form.Close())
		r := httptest.NewRequest("POST", "/api/documents", &data)
		r.Header.Set("Origin", origin)
		r.Header.Set("Content-Type", form.FormDataContentType())
		r.Header.Set("Idempotency-Key", key)
		r.AddCookie(&http.Cookie{Name: "documind_session", Value: token})
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	var doc documents.Document
	key := auth.Random()
	body := "Synthetic invoice\nTotal: 42\n"
	t.Run("sessions and CSRF", func(t *testing.T) {
		status(t, request("GET", "/api/me", "", a.token, ""), 200)
		for _, source := range []string{"", "null", "https://attacker.invalid", origin + ".attacker.invalid"} {
			status(t, request("POST", "/api/auth/logout", "", a.token, source), 403)
		}
		status(t, request("GET", "/api/me", "", auth.Random(), ""), 401)
		status(t, request("POST", "/api/auth/login", fmt.Sprintf(`{"email":%q,"password":"wrong-password-long"}`, a.email), "", origin), 401)
		status(t, request("POST", "/api/auth/register", fmt.Sprintf(`{"email":%q,"password":"Synthetic-password-123"}`, a.email), "", origin), 409)
		var digest string
		must(t, clients.DB.QueryRow(ctx, "SELECT token_hash FROM documind.sessions WHERE user_id=$1", a.user.ID).Scan(&digest))
		if digest == a.token || digest != auth.Digest(a.token) {
			t.Fatal("raw token persisted")
		}
	})
	t.Run("upload accepted only after persistence", func(t *testing.T) {
		w := upload("invoice.txt", body, key, a.token)
		status(t, w, 202)
		must(t, json.Unmarshal(w.Body.Bytes(), &doc))
		if doc.Status != "queued" || doc.Generation != 1 || doc.Attempts != 0 {
			t.Fatal("incorrect initial job")
		}
		if strings.Contains(w.Body.String(), "uploads/") {
			t.Fatal("object key exposed")
		}
		var n int
		must(t, clients.DB.QueryRow(ctx, "SELECT count(*) FROM documind.jobs WHERE document_id=$1", doc.ID).Scan(&n))
		if n != 1 {
			t.Fatal("job not durable")
		}
		must(t, clients.DB.QueryRow(ctx, "SELECT count(*) FROM documind.upload_intents i JOIN documind.documents d ON d.object_key=i.object_key WHERE d.id=$1", doc.ID).Scan(&n))
		if n != 0 {
			t.Fatal("committed upload retains intent")
		}
	})
	t.Run("every read and retry is owner scoped", func(t *testing.T) {
		for _, suffix := range []string{"", "/download", "/result", "/retry"} {
			method := "GET"
			if suffix == "/retry" {
				method = "POST"
			}
			p := "/api/documents/" + doc.ID + suffix
			status(t, request(method, p, "", "", origin), 401)
			status(t, request(method, p, "", b.token, origin), 404)
		}
		status(t, request("POST", "/api/documents", "", "", origin), 401)
		w := request("GET", "/api/documents", "", b.token, "")
		status(t, w, 200)
		if w.Body.String() != "{\"documents\":[]}\n" {
			t.Fatal("foreign document listed")
		}
		w = request("GET", "/api/documents/"+doc.ID+"/download", "", a.token, "")
		status(t, w, 200)
		if w.Body.String() != body || !strings.HasPrefix(w.Header().Get("Content-Disposition"), "attachment;") {
			t.Fatal("download mismatch")
		}
		status(t, request("GET", "/api/documents/"+doc.ID+"/result", "", a.token, ""), 409)
		status(t, request("POST", "/api/documents/"+doc.ID+"/retry", "", a.token, origin), 409)
	})
	t.Run("idempotency is byte based and owner scoped", func(t *testing.T) {
		w := upload("renamed.txt", body, key, a.token)
		status(t, w, 202)
		var replay documents.Document
		must(t, json.Unmarshal(w.Body.Bytes(), &replay))
		if replay.ID != doc.ID || replay.Name != "invoice.txt" {
			t.Fatal("replay changed identity")
		}
		status(t, upload("invoice.txt", "different", key, a.token), 409)
		w = upload("invoice.txt", body, key, b.token)
		status(t, w, 202)
		must(t, json.Unmarshal(w.Body.Bytes(), &replay))
		if replay.ID == doc.ID {
			t.Fatal("owners shared document")
		}
	})
	t.Run("eight concurrent duplicates produce one job", func(t *testing.T) {
		k := auth.Random()
		var wg sync.WaitGroup
		ids := make(chan string, 8)
		errs := make(chan error, 8)
		for range 8 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				d, e := service.Register(ctx, a.user.ID, k, "parallel.txt", []byte(body))
				ids <- d.ID
				errs <- e
			}()
		}
		wg.Wait()
		close(ids)
		close(errs)
		for e := range errs {
			must(t, e)
		}
		id := ""
		for next := range ids {
			if id != "" && id != next {
				t.Fatal("duplicate accepted")
			}
			id = next
		}
		var n int
		must(t, clients.DB.QueryRow(ctx, "SELECT count(*) FROM documind.jobs WHERE document_id=$1", id).Scan(&n))
		if n != 1 {
			t.Fatal("duplicate jobs")
		}
	})
	t.Run("database constraints reject a second active generation and orphan result", func(t *testing.T) {
		_, e := clients.DB.Exec(ctx, "INSERT INTO documind.jobs(document_id,generation,state) VALUES($1,2,'queued')", doc.ID)
		if e == nil {
			t.Fatal("second active generation accepted")
		}
		_, e = clients.DB.Exec(ctx, "INSERT INTO documind.results(document_id,generation,schema_version,result,provider,model) VALUES($1,99,1,'{}','fake','fake')", doc.ID)
		if e == nil {
			t.Fatal("result without job accepted")
		}
	})

	t.Run("input boundaries", func(t *testing.T) {
		for _, tc := range []struct {
			name, body string
			want       int
		}{{"empty.txt", "", 422}, {"fake.txt", "%PDF-1.7", 415}, {"scan.pdf", "%PDF-1.7", 415}, {"binary.txt", "a\x00b", 422}, {"utf.txt", "\xff", 422}, {"big.txt", strings.Repeat("x", documents.MaxBytes+1), 413}, {"long.txt", strings.Repeat("x", 200001), 422}, {"script.html", "<script>hi</script>", 415}} {
			status(t, upload(tc.name, tc.body, auth.Random(), a.token), tc.want)
		}
		status(t, upload("valid.txt", "hello", "short", a.token), 400)
	})
	t.Run("queue survives application recreation", func(t *testing.T) {
		handler = httpapi.NewApplication(auth.New(clients.DB), &documents.Service{DB: clients.DB, Objects: objects}, origin, httpapi.New(clients.DB.Ping, clients.StorageReady))
		w := request("GET", "/api/documents/"+doc.ID, "", a.token, "")
		status(t, w, 200)
		if !strings.Contains(w.Body.String(), `"status":"queued"`) {
			t.Fatal("queue lost")
		}
	})
	t.Run("storage and deferred commit failures leave no job", func(t *testing.T) {
		for _, mode := range []string{"storage", "commit"} {
			t.Run(mode, func(t *testing.T) {
				tracking := &trackingObjects{Objects: objects, fail: mode == "storage"}
				svc := documents.Service{DB: clients.DB, Objects: tracking}
				k := auth.Random()
				name := "fail-" + auth.Random() + ".txt"
				if mode == "commit" {
					// A real deferred PostgreSQL constraint trigger fails COMMIT, after both inserts succeeded.
					_, e = admin.Exec(ctx, `CREATE FUNCTION documind.test_reject_commit() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.name='`+name+`' THEN RAISE EXCEPTION 'synthetic commit failure'; END IF; RETURN NEW; END $$; CREATE CONSTRAINT TRIGGER test_reject_commit AFTER INSERT ON documind.documents DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION documind.test_reject_commit();`)
					must(t, e)
					defer func() {
						_, e := admin.Exec(ctx, "DROP TRIGGER test_reject_commit ON documind.documents; DROP FUNCTION documind.test_reject_commit()")
						must(t, e)
					}()
				}
				_, e := svc.Register(ctx, a.user.ID, k, name, []byte(body))
				if e == nil {
					t.Fatal("failure accepted")
				}
				var n int
				must(t, clients.DB.QueryRow(ctx, "SELECT count(*) FROM documind.documents WHERE owner_id=$1 AND idempotency_key=$2", a.user.ID, k).Scan(&n))
				if n != 0 {
					t.Fatal("partial registration")
				}
				key := tracking.key
				if key == "" {
					t.Fatal("object write not reached")
				}
				defer objects.Delete(ctx, key)
				must(t, clients.DB.QueryRow(ctx, "SELECT count(*) FROM documind.upload_intents WHERE object_key=$1", key).Scan(&n))
				if n != 1 {
					t.Fatal("no durable cleanup intent")
				}
				_, e = svc.Cleanup(ctx)
				must(t, e)
				r, e := objects.Open(ctx, key)
				must(t, e)
				r.Close() // fresh object retained
				_, e = admin.Exec(ctx, "UPDATE documind.upload_intents SET created_at=now()-interval '2 hours' WHERE object_key=$1", key)
				must(t, e)
				_, e = svc.Cleanup(ctx)
				must(t, e)
				r, e = objects.Open(ctx, key)
				if e == nil {
					r.Close()
					t.Fatal("orphan not deleted")
				}
				must(t, clients.DB.QueryRow(ctx, "SELECT count(*) FROM documind.upload_intents WHERE object_key=$1", key).Scan(&n))
				if n != 0 {
					t.Fatal("cleanup intent not removed")
				}
				kept, e := service.Get(ctx, a.user.ID, doc.ID)
				must(t, e)
				r, e = objects.Open(ctx, kept.ObjectKey)
				must(t, e)
				r.Close()
			})
		}
	})
	t.Run("logout revokes and expired session fails", func(t *testing.T) {
		status(t, request("POST", "/api/auth/logout", "", a.token, origin), 200)
		status(t, request("GET", "/api/me", "", a.token, ""), 401)
		w := request("POST", "/api/auth/login", fmt.Sprintf(`{"email":%q,"password":"Synthetic-password-123"}`, a.email), "", origin)
		status(t, w, 200)
		newToken := w.Result().Cookies()[0].Value
		_, e = clients.DB.Exec(ctx, "UPDATE documind.sessions SET expires_at=$1 WHERE token_hash=$2", time.Now().Add(-time.Second), auth.Digest(newToken))
		must(t, e)
		status(t, request("GET", "/api/me", "", newToken, ""), 401)
	})
}

type trackingObjects struct {
	documents.Objects
	fail bool
	key  string
}

func (s *trackingObjects) Put(ctx context.Context, key string, b []byte) error {
	s.key = key
	e := s.Objects.Put(ctx, key, b)
	if e != nil {
		return e
	}
	if s.fail {
		return errors.New("synthetic storage confirmation failure")
	}
	return nil
}
func must(t *testing.T, e error) {
	t.Helper()
	if e != nil {
		t.Fatal(e)
	}
}
func status(t *testing.T, w *httptest.ResponseRecorder, want int) {
	t.Helper()
	if w.Code != want {
		t.Fatalf("status %d want %d: %s", w.Code, want, w.Body.String())
	}
}
