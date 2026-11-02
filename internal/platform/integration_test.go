package platform

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"github.com/minio/minio-go/v7"
	"github.com/msalomonn/documind/db"
	"github.com/msalomonn/documind/internal/config"
	"io"
	"net/http"
	"os"
	"testing"
	"time"
)

func TestLocalInfrastructure(t *testing.T) {
	if os.Getenv("DOCUMIND_INTEGRATION") != "1" {
		t.Skip("set DOCUMIND_INTEGRATION=1 for real local-service checks")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	c, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	clients, err := Open(ctx, c)
	if err != nil {
		t.Fatal(err)
	}
	defer clients.DB.Close()
	t.Run("runtime database cannot alter schema", func(t *testing.T) {
		if err := clients.DB.Ping(ctx); err != nil {
			t.Fatal("database unavailable")
		}
		var super, bypass, createRole, createSchema bool
		err := clients.DB.QueryRow(ctx, `SELECT rolsuper,rolbypassrls,rolcreaterole,has_schema_privilege(current_user,'documind','CREATE') FROM pg_roles WHERE rolname=current_user`).Scan(&super, &bypass, &createRole, &createSchema)
		if err != nil || super || bypass || createRole || createSchema {
			t.Fatal("unsafe runtime database permissions")
		}
	})
	t.Run("migrations repeat without duplicates", func(t *testing.T) {
		for i := 0; i < 2; i++ {
			if err := db.Migrate(ctx, os.Getenv("DATABASE_ADMIN_URL")); err != nil {
				t.Fatal("migration failed")
			}
		}
	})
	t.Run("private object round trip and access boundaries", func(t *testing.T) {
		if err := clients.StorageReady(ctx); err != nil {
			t.Fatal(err)
		}
		id := make([]byte, 16)
		if _, err := rand.Read(id); err != nil {
			t.Fatal(err)
		}
		key := "bootstrap-tests/" + hex.EncodeToString(id)
		contents := []byte("Synthetic DocuMind infrastructure fixture. No customer data.")
		if _, err := clients.S3.PutObject(ctx, c.Bucket, key, bytes.NewReader(contents), int64(len(contents)), minio.PutObjectOptions{ContentType: "text/plain"}); err != nil {
			t.Fatal("put failed")
		}
		defer clients.S3.RemoveObject(context.Background(), c.Bucket, key, minio.RemoveObjectOptions{})
		object, err := clients.S3.GetObject(ctx, c.Bucket, key, minio.GetObjectOptions{})
		if err != nil {
			t.Fatal("get failed")
		}
		data, err := io.ReadAll(io.LimitReader(object, 1024))
		object.Close()
		if err != nil || !bytes.Equal(data, contents) {
			t.Fatal("object did not round trip")
		}
		client := &http.Client{Timeout: 3 * time.Second}
		response, err := client.Get("http://" + c.Endpoint + "/" + c.Bucket + "/" + key)
		if err != nil {
			t.Fatal("anonymous request failed unexpectedly")
		}
		response.Body.Close()
		if response.StatusCode != 403 {
			t.Fatalf("anonymous read: %d", response.StatusCode)
		}
		wrong, err := S3(c.Endpoint, "invalid-access-key", "invalid-secret")
		if err != nil {
			t.Fatal(err)
		}
		obj, err := wrong.GetObject(ctx, c.Bucket, key, minio.GetObjectOptions{})
		if err != nil {
			t.Fatal(err)
		}
		_, err = io.ReadAll(io.LimitReader(obj, 1024))
		obj.Close()
		if err == nil {
			t.Fatal("wrong credentials allowed")
		}
		if err := clients.S3.MakeBucket(ctx, "forbidden-"+hex.EncodeToString(id), minio.MakeBucketOptions{}); err == nil {
			t.Fatal("runtime credentials created an unrelated bucket")
		}
		if err := clients.S3.RemoveObject(ctx, c.Bucket, key, minio.RemoveObjectOptions{}); err != nil {
			t.Fatal("delete failed")
		}
		if _, err := clients.S3.StatObject(ctx, c.Bucket, key, minio.StatObjectOptions{}); minio.ToErrorResponse(err).Code != "NoSuchKey" {
			t.Fatal("deleted object still exists or unexpected response")
		}
	})
}
