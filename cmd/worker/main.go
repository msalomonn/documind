package main

import (
	"context"
	"flag"
	"github.com/msalomonn/documind/internal/config"
	"github.com/msalomonn/documind/internal/platform"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	check := flag.Bool("check", false, "check dependencies and exit without processing jobs")
	flag.Parse()
	if !run(*check) {
		os.Exit(1)
	}
}
func run(check bool) bool {
	c, err := config.Load()
	if err != nil {
		log.Print("worker configuration unavailable")
		return false
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	clients, err := platform.Open(ctx, c)
	if err != nil {
		log.Print("worker clients unavailable")
		return false
	}
	defer clients.DB.Close()
	probe, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if clients.DB.Ping(probe) != nil || clients.StorageReady(probe) != nil {
		log.Print("worker dependencies unavailable")
		return false
	}
	log.Print("Worker bootstrap ready; job processing is not implemented")
	if !check {
		<-ctx.Done()
	}
	return true
}
