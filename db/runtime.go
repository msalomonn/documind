package db

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
)

// EnsureRuntimeRole initializes the local role without resetting an existing password.
func EnsureRuntimeRole(ctx context.Context, url, password string) error {
	if len(password) < 24 {
		return errors.New("runtime password missing or too short")
	}
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
	var exists bool
	if err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='documind_app')").Scan(&exists); err != nil {
		return err
	}
	if !exists {
		var statement string
		if err = tx.QueryRow(ctx, "SELECT format('CREATE ROLE documind_app LOGIN PASSWORD %L', $1::text)", password).Scan(&statement); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, statement); err != nil {
			return errors.New("runtime role creation failed")
		}
	}
	if _, err = tx.Exec(ctx, "REVOKE CREATE ON SCHEMA public FROM PUBLIC; GRANT CONNECT ON DATABASE documind TO documind_app"); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
