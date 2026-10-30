package platform

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/msalomonn/documind/internal/config"
	"io"
)

type Clients struct {
	DB     *pgxpool.Pool
	S3     *minio.Client
	Bucket string
}

func S3(endpoint, key, secret string) (*minio.Client, error) {
	return minio.New(endpoint, &minio.Options{Creds: credentials.NewStaticV4(key, secret, ""), Secure: false, Region: "us-east-1", BucketLookup: minio.BucketLookupPath})
}
func Open(ctx context.Context, c config.Config) (*Clients, error) {
	pool, err := pgxpool.New(ctx, c.DatabaseURL)
	if err != nil {
		return nil, errors.New("invalid database configuration")
	}
	client, err := S3(c.Endpoint, c.AccessKey, c.SecretKey)
	if err != nil {
		pool.Close()
		return nil, errors.New("invalid storage configuration")
	}
	return &Clients{pool, client, c.Bucket}, nil
}

const ReadinessKey = ".bootstrap/ready"
const ReadinessContent = "documind-ready-v1"

func (c *Clients) StorageReady(ctx context.Context) error {
	object, err := c.S3.GetObject(ctx, c.Bucket, ReadinessKey, minio.GetObjectOptions{})
	if err != nil {
		return errors.New("storage unavailable")
	}
	defer object.Close()
	data, err := io.ReadAll(io.LimitReader(object, 64))
	if err != nil || string(data) != ReadinessContent {
		return errors.New("storage unavailable or not initialized")
	}
	return nil
}
