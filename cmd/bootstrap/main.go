package main

import (
	"context"
	"github.com/minio/minio-go/v7"
	"github.com/msalomonn/documind/db"
	"github.com/msalomonn/documind/internal/config"
	"github.com/msalomonn/documind/internal/platform"
	"log"
	"os"
	"strings"
	"time"
)

func main() {
	if !run() {
		os.Exit(1)
	}
}
func run() bool {
	c, err := config.Load()
	if err != nil {
		log.Print("bootstrap configuration unavailable")
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	admin := os.Getenv("DATABASE_ADMIN_URL")
	if admin == "" || db.EnsureRuntimeRole(ctx, admin, os.Getenv("DOCUMIND_DB_PASSWORD")) != nil || db.Migrate(ctx, admin) != nil {
		log.Print("migration failed; inspect local database configuration and migration checksums")
		return false
	}
	s3, err := platform.S3(c.Endpoint, os.Getenv("S3_ADMIN_ACCESS_KEY"), os.Getenv("S3_ADMIN_SECRET_KEY"))
	if err != nil {
		log.Print("storage configuration unavailable")
		return false
	}
	for {
		exists, err := s3.BucketExists(ctx, c.Bucket)
		if err == nil {
			if exists {
				break
			}
			if s3.MakeBucket(ctx, c.Bucket, minio.MakeBucketOptions{Region: "us-east-1"}) == nil {
				break
			}
		}
		select {
		case <-ctx.Done():
			log.Print("storage bootstrap timed out")
			return false
		case <-time.After(time.Second):
		}
	}
	if _, err := s3.PutObject(ctx, c.Bucket, platform.ReadinessKey, strings.NewReader(platform.ReadinessContent), int64(len(platform.ReadinessContent)), minio.PutObjectOptions{ContentType: "text/plain"}); err != nil {
		log.Print("storage marker initialization failed; wait for service health and retry")
		return false
	}
	log.Print("Migrations verified and private bucket ready")
	return true
}
