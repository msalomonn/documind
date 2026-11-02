package config

import (
	"errors"
	"os"
)

type Config struct{ DatabaseURL, Endpoint, Bucket, AccessKey, SecretKey, Address string }

func Load() (Config, error) {
	c := Config{os.Getenv("DATABASE_URL"), os.Getenv("S3_ENDPOINT"), os.Getenv("S3_BUCKET"), os.Getenv("S3_ACCESS_KEY"), os.Getenv("S3_SECRET_KEY"), os.Getenv("API_ADDRESS")}
	if c.Address == "" {
		c.Address = "127.0.0.1:4321"
	}
	if c.DatabaseURL == "" || c.Endpoint == "" || c.Bucket == "" || c.AccessKey == "" || c.SecretKey == "" {
		return Config{}, errors.New("missing local configuration; run npm run setup")
	}
	return c, nil
}
