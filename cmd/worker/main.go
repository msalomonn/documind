package main

import (
	"context"
	"encoding/hex"
	"errors"
	"flag"
	"github.com/msalomonn/documind/internal/config"
	"github.com/msalomonn/documind/internal/documents"
	"github.com/msalomonn/documind/internal/jobs"
	"github.com/msalomonn/documind/internal/platform"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	if e := run(); e != nil {
		log.Print("worker stopped without confirming completion; check services and documented options")
		os.Exit(1)
	}
}
func run() error {
	check := flag.Bool("check", false, "check dependencies without claiming jobs")
	id := flag.String("smoke-document", "", "explicit synthetic document ID; no general extractor is configured")
	once := flag.Bool("once", false, "attempt one eligible claim and exit")
	delay := flag.Duration("smoke-delay", 0, "bounded diagnostic processing delay")
	failures := flag.Int("smoke-failures", 0, "diagnostic transient failures before success (0-3)")
	permanent := flag.Bool("smoke-permanent", false, "inject a permanent diagnostic failure")
	options := jobs.Defaults()
	flag.DurationVar(&options.Lease, "lease", options.Lease, "claim lease duration")
	flag.DurationVar(&options.Heartbeat, "heartbeat", options.Heartbeat, "lease renewal interval")
	flag.DurationVar(&options.Attempt, "attempt-timeout", options.Attempt, "maximum fenced attempt duration including finalization")
	flag.Parse()
	if !*check {
		decoded, e := hex.DecodeString(*id)
		if e != nil || len(decoded) != 32 {
			log.Print("No extractor configured. Use --smoke-document ID only with a DOCUMIND WORKER SMOKE fixture; see README.")
			return errors.New("smoke_document_required")
		}
		if !options.Valid() || *delay < 0 || *delay > 2*time.Minute || *failures < 0 || *failures > 3 {
			return errors.New("invalid worker options")
		}
	}
	c, e := config.Load()
	if e != nil {
		return e
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	clients, e := platform.Open(ctx, c)
	if e != nil {
		return e
	}
	defer clients.DB.Close()
	probe, cancel := context.WithTimeout(ctx, 3*time.Second)
	e = clients.DB.Ping(probe)
	if e == nil {
		e = clients.StorageReady(probe)
	}
	cancel()
	if e != nil {
		return e
	}
	if *check {
		log.Print("Worker dependencies ready; check mode does not claim jobs")
		return nil
	}
	q, e := jobs.New(clients.DB, options)
	if e != nil {
		return e
	}
	runner := jobs.Runner{Queue: q, Processor: jobs.Smoke{Objects: documents.S3{Client: clients.S3, Bucket: clients.Bucket}, Delay: *delay, TransientFailures: *failures, Permanent: *permanent}, DocumentID: *id}
	log.Print("Diagnostic worker started for the explicitly selected fixture; no AI calls")
	if *once {
		e = runner.Once(ctx)
	} else {
		e = runner.Run(ctx)
	}
	if errors.Is(e, context.Canceled) || errors.Is(e, jobs.ErrEmpty) {
		return nil
	}
	return e
}
