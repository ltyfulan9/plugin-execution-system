package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"os"
)

// Migrate applies the enterprise Postgres metadata schema. In production this
// should be run by release automation or a migration job; cmd/server may call it
// only when POSTGRES_AUTO_MIGRATE=true for controlled deployments.
func Migrate(ctx context.Context, db *sql.DB, path string) error {
	if path == "" {
		path = "migrations/postgres/001_enterprise_metadata.sql"
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read postgres migration %s: %w", path, err)
	}
	if _, err := db.ExecContext(ctx, string(b)); err != nil {
		return fmt.Errorf("apply postgres migration: %w", err)
	}
	return nil
}
