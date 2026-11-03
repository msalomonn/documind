package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/msalomonn/documind/internal/auth"
	"github.com/msalomonn/documind/internal/documents"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const cookieName = "documind_session"

type Application struct {
	Auth      *auth.Service
	Documents *documents.Service
	Origin    string
	uploads   chan struct{}
}

func NewApplication(a *auth.Service, d *documents.Service, origin string, health http.Handler) http.Handler {
	app := &Application{Auth: a, Documents: d, Origin: origin, uploads: make(chan struct{}, 2)}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		ctx, cancel := context.WithTimeout(r.Context(), 35*time.Second)
		defer cancel()
		r = r.WithContext(ctx)
		if r.Method != "GET" && r.Method != "HEAD" && r.Header.Get("Origin") != origin {
			fail(w, 403, "origin_rejected")
			return
		}
		switch r.URL.Path {
		case "/api/auth/register", "/api/auth/login":
			if r.Method != "POST" {
				fail(w, 405, "method_not_allowed")
				return
			}
			app.credentials(w, r)
			return
		case "/api/auth/logout":
			if r.Method != "POST" {
				fail(w, 405, "method_not_allowed")
				return
			}
			if e := app.Auth.Logout(ctx, token(r)); e != nil {
				fail(w, 503, "service_unavailable")
				return
			}
			app.cookie(w, "", -1)
			write(w, 200, map[string]bool{"signedOut": true})
			return
		}
		if r.URL.Path != "/api/me" && r.URL.Path != "/api/documents" && !strings.HasPrefix(r.URL.Path, "/api/documents/") {
			health.ServeHTTP(w, r)
			return
		}
		user, e := app.Auth.Resolve(ctx, token(r))
		if e != nil {
			if errors.Is(e, auth.ErrSession) {
				fail(w, 401, "authentication_required")
			} else {
				fail(w, 503, "service_unavailable")
			}
			return
		}
		if r.URL.Path == "/api/me" {
			if r.Method != "GET" {
				fail(w, 405, "method_not_allowed")
				return
			}
			write(w, 200, user)
			return
		}
		if r.URL.Path == "/api/documents" {
			if r.Method == "POST" {
				app.upload(w, r, user.ID)
				return
			}
			if r.Method != "GET" {
				fail(w, 405, "method_not_allowed")
				return
			}
			docs, e := app.Documents.List(ctx, user.ID)
			if e != nil {
				fail(w, 503, "service_unavailable")
				return
			}
			write(w, 200, map[string]any{"documents": docs})
			return
		}
		path := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/documents/"), "/")
		if len(path) > 2 || len(path[0]) != 64 {
			fail(w, 404, "not_found")
			return
		}
		doc, e := app.Documents.Get(ctx, user.ID, path[0])
		if e != nil {
			if errors.Is(e, documents.ErrNotFound) {
				fail(w, 404, "not_found")
			} else {
				fail(w, 503, "service_unavailable")
			}
			return
		}
		if len(path) == 1 && r.Method == "GET" {
			write(w, 200, doc)
			return
		}
		if len(path) == 2 {
			switch path[1] {
			case "download":
				if r.Method != "GET" {
					fail(w, 405, "method_not_allowed")
					return
				}
				body, e := app.Documents.Objects.Open(ctx, doc.ObjectKey)
				if e != nil {
					fail(w, 503, "storage_unavailable")
					return
				}
				defer body.Close()
				w.Header().Set("Content-Length", strconv.FormatInt(doc.Size, 10))
				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": doc.Name}))
				w.Header().Set("Content-Security-Policy", "sandbox; default-src 'none'")
				// Stored originals are immutable, bounded and served only after owner authorization.
				_, _ = io.Copy(w, io.LimitReader(body, documents.MaxBytes))
				return
			case "result":
				if r.Method == "GET" {
					fail(w, 409, "result_not_ready")
					return
				}
			case "retry":
				if r.Method == "POST" {
					if doc.Status == "failed" || doc.Status == "succeeded" {
						fail(w, 409, "retry_not_enabled")
					} else {
						fail(w, 409, "job_active")
					}
					return
				}
			}
		}
		fail(w, 404, "not_found")
	})
}
func token(r *http.Request) string {
	c, e := r.Cookie(cookieName)
	if e != nil {
		return ""
	}
	return c.Value
}
func (a *Application) cookie(w http.ResponseWriter, value string, age int) {
	http.SetCookie(w, &http.Cookie{Name: cookieName, Value: value, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: strings.HasPrefix(a.Origin, "https://"), MaxAge: age})
}
func fail(w http.ResponseWriter, status int, code string) {
	write(w, status, map[string]string{"error": code})
}
func (a *Application) credentials(w http.ResponseWriter, r *http.Request) {
	mt, _, e := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if e != nil || mt != "application/json" {
		fail(w, 415, "json_required")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 2048)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if decoder.Decode(&input) != nil || decoder.Decode(new(any)) != io.EOF {
		fail(w, 400, "invalid_request")
		return
	}
	register := r.URL.Path == "/api/auth/register"
	user, t, e := a.Auth.Authenticate(r.Context(), input.Email, input.Password, register)
	if e != nil {
		switch {
		case errors.Is(e, auth.ErrInvalid):
			fail(w, 400, e.Error())
		case errors.Is(e, auth.ErrCredentials):
			fail(w, 401, e.Error())
		case errors.Is(e, auth.ErrExists):
			fail(w, 409, e.Error())
		case errors.Is(e, auth.ErrBusy):
			w.Header().Set("Retry-After", "60")
			fail(w, 429, e.Error())
		default:
			fail(w, 503, "service_unavailable")
		}
		return
	}
	a.cookie(w, t, int(auth.Lifetime.Seconds()))
	status := 200
	if register {
		status = 201
	}
	write(w, status, user)
}
func (a *Application) upload(w http.ResponseWriter, r *http.Request, owner string) {
	select {
	case a.uploads <- struct{}{}:
		defer func() { <-a.uploads }()
	default:
		fail(w, 429, "uploads_busy")
		return
	}
	key := r.Header.Get("Idempotency-Key")
	if !documents.ValidKey(key) {
		fail(w, 400, "idempotency_key_required")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, documents.MaxBytes+64*1024)
	reader, e := r.MultipartReader()
	if e != nil {
		fail(w, 400, "multipart_required")
		return
	}
	part, e := reader.NextPart()
	if e != nil {
		fail(w, 400, "file_required")
		return
	}
	if part.FormName() != "file" || part.FileName() == "" {
		fail(w, 400, "single_file_required")
		return
	}
	b, e := io.ReadAll(io.LimitReader(part, documents.MaxBytes+1))
	if e != nil || len(b) > documents.MaxBytes {
		fail(w, 413, "file_too_large")
		return
	}
	name, mt := part.FileName(), part.Header.Get("Content-Type")
	if _, e = reader.NextPart(); e != io.EOF {
		fail(w, 400, "single_file_required")
		return
	}
	if code := documents.Validate(name, mt, b); code != "" {
		status := 422
		if code == "pdf_not_enabled" || code == "unsupported_file" {
			status = 415
		}
		if code == "file_too_large" {
			status = 413
		}
		fail(w, status, code)
		return
	}
	d, e := a.Documents.Register(r.Context(), owner, key, name, b)
	if e != nil {
		switch {
		case errors.Is(e, documents.ErrConflict):
			fail(w, 409, "idempotency_conflict")
		case errors.Is(e, documents.ErrStorage):
			fail(w, 503, "storage_unavailable")
		default:
			fail(w, 503, "registration_unavailable")
		}
		return
	}
	write(w, 202, d)
}
