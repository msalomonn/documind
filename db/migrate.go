package db

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"errors"
	"github.com/jackc/pgx/v5"
)

//go:embed migrations/*.sql
var migrations embed.FS

func Migrate(ctx context.Context, url string) error {
	c, err := pgx.Connect(ctx, url)
	if err != nil {
		return err
	}
	defer c.Close(ctx)
	tx, err := c.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(43204321)"); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `CREATE TABLE IF NOT EXISTS public.documind_migrations(name text PRIMARY KEY,checksum text NOT NULL);REVOKE ALL ON public.documind_migrations FROM PUBLIC;`); err != nil {
		return err
	}
	files, err := migrations.ReadDir("migrations")
	if err != nil {
		return err
	}
	for _, f := range files {
		data, err := migrations.ReadFile("migrations/" + f.Name())
		if err != nil {
			return err
		}
		sum := sha256.Sum256(data)
		checksum := hex.EncodeToString(sum[:])
		var stored string
		err = tx.QueryRow(ctx, "SELECT checksum FROM public.documind_migrations WHERE name=$1", f.Name()).Scan(&stored)
		if err == nil {
			if stored != checksum {
				return errors.New("migration checksum mismatch")
			}
			continue
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		if _, err = tx.Exec(ctx, string(data)); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, "INSERT INTO public.documind_migrations VALUES($1,$2)", f.Name(), checksum); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
