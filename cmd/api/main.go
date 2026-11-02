package main

import (
	"context"
	"errors"
	"github.com/msalomonn/documind/internal/config"
	"github.com/msalomonn/documind/internal/httpapi"
	"github.com/msalomonn/documind/internal/platform"
	"log"
	"net/http"
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
	server := &http.Server{Addr: c.Address, Handler: httpapi.New(clients.DB.Ping, clients.StorageReady), ReadHeaderTimeout: 3 * time.Second, ReadTimeout: 5 * time.Second, WriteTimeout: 5 * time.Second, IdleTimeout: 30 * time.Second}
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
