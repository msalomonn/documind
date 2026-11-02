package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthAndPrivateBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name        string
		db, storage error
		status      int
	}{
		{"ready", nil, nil, 200}, {"database unavailable", errors.New("secret DSN"), nil, 503}, {"storage unavailable", nil, errors.New("secret S3 key"), 503},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := New(func(context.Context) error { return tc.db }, func(context.Context) error { return tc.storage })
			for _, path := range []string{"/api/health/live", "/api/health/ready", "/api/me", "/api/documents", "/api/documents/foreign/retry"} {
				req := httptest.NewRequest("GET", path, nil)
				req.Header.Set("X-User-ID", "forged-owner")
				w := httptest.NewRecorder()
				h.ServeHTTP(w, req)
				want := 401
				if path == "/api/health/live" {
					want = 200
				}
				if path == "/api/health/ready" {
					want = tc.status
				}
				if w.Code != want {
					t.Fatalf("%s: got %d, want %d", path, w.Code, want)
				}
				if strings.Contains(w.Body.String(), "secret") || strings.Contains(w.Body.String(), "forged-owner") {
					t.Fatal("private data leaked")
				}
				if w.Header().Get("Cache-Control") != "no-store" {
					t.Fatal("missing no-store")
				}
			}
		})
	}
}

func TestProbeHasDeadline(t *testing.T) {
	probe := func(ctx context.Context) error {
		if _, ok := ctx.Deadline(); !ok {
			t.Error("probe missing deadline")
		}
		return nil
	}
	h := New(probe, probe)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/health/ready", nil))
	if w.Code != 200 {
		t.Fatal(w.Code)
	}
}
