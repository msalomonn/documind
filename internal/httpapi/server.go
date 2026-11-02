package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

type Probe func(context.Context) error

func New(database, storage Probe) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health/live", func(w http.ResponseWriter, r *http.Request) { write(w, 200, map[string]string{"status": "alive"}) })
	mux.HandleFunc("GET /api/health/ready", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		status := 200
		parts := map[string]string{}
		for _, p := range []struct {
			name  string
			check Probe
		}{{"database", database}, {"storage", storage}} {
			parts[p.name] = "ready"
			if err := p.check(ctx); err != nil {
				parts[p.name] = "unavailable"
				status = 503
			}
		}
		state := "ready"
		if status != 200 {
			state = "unavailable"
		}
		write(w, status, map[string]any{"status": state, "components": parts, "documentProcessing": false})
	})
	// No identity bypass: authentication is implemented before document routes in DM-01.
	deny := func(w http.ResponseWriter, r *http.Request) {
		write(w, 401, map[string]string{"error": "authentication_required"})
	}
	mux.HandleFunc("/api/me", deny)
	mux.HandleFunc("/api/documents", deny)
	mux.HandleFunc("/api/documents/", deny)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { write(w, 404, map[string]string{"error": "not_found"}) })
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		mux.ServeHTTP(w, r)
	})
}
func write(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
