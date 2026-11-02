package main

import (
	"context"
	"github.com/msalomonn/documind/internal/config"
	"github.com/msalomonn/documind/internal/documents"
	"github.com/msalomonn/documind/internal/platform"
	"log"
	"os"
	"time"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	c, e := config.Load()
	if e != nil {
		log.Print("cleanup configuration unavailable")
		os.Exit(1)
	}
	p, e := platform.Open(ctx, c)
	if e != nil {
		log.Print("cleanup services unavailable")
		os.Exit(1)
	}
	defer p.DB.Close()
	s := documents.Service{DB: p.DB, Objects: documents.S3{Client: p.S3, Bucket: p.Bucket}}
	n, e := s.Cleanup(ctx)
	if e != nil {
		log.Print("cleanup incomplete; safe to retry")
		os.Exit(1)
	}
	log.Printf("cleaned %d expired upload intents", n)
}
