package main

import (
	"context"
	"errors"
	"github.com/msalomonn/documind/internal/auth"
	"github.com/msalomonn/documind/internal/config"
	"github.com/msalomonn/documind/internal/documents"
	"github.com/msalomonn/documind/internal/httpapi"
	"github.com/msalomonn/documind/internal/platform"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	if err := run(); err != nil {
		log.Print("api startup or shutdown failed; check local configuration and services")
		os.Exit(1)
	}
}
func run() error {
	c, err := config.Load()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	clients, err := platform.Open(ctx, c)
	if err != nil {
		return err
	}
	defer clients.DB.Close()
	origin := os.Getenv("APP_ORIGIN")
	if origin == "" {
		origin = "http://127.0.0.1:4320"
	}
	parsed, e := url.Parse(origin)
	if e != nil || parsed.Host == "" || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.User != nil || (parsed.Scheme != "https" && (parsed.Scheme != "http" || (parsed.Hostname() != "127.0.0.1" && parsed.Hostname() != "localhost"))) {
		return errors.New("invalid APP_ORIGIN")
	}
	handler := httpapi.NewApplication(auth.New(clients.DB), &documents.Service{DB: clients.DB, Objects: documents.S3{Client: clients.S3, Bucket: clients.Bucket}}, origin, httpapi.New(clients.DB.Ping, clients.StorageReady))
	server := &http.Server{Addr: c.Address, Handler: handler, ReadHeaderTimeout: 3 * time.Second, ReadTimeout: 35 * time.Second, WriteTimeout: 40 * time.Second, IdleTimeout: 30 * time.Second}
	done := make(chan error, 1)
	go func() { done <- server.ListenAndServe() }()
	log.Print("DocuMind API started; document processing is not implemented")
	select {
	case err = <-done:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return server.Shutdown(shutdown)
	}
}
