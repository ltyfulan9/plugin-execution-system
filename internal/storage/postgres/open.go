package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type OpenOptions struct {
	DriverName      string
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// OpenDB centralizes the production Postgres connection contract. The actual
// SQL driver is intentionally not hidden behind local fallbacks: production
// builds must register a real Postgres database/sql driver such as pgx stdlib
// or lib/pq at the application boundary.
func OpenDB(ctx context.Context, opts OpenOptions) (*sql.DB, error) {
	if opts.DriverName == "" {
		opts.DriverName = "postgres"
	}
	if opts.DSN == "" {
		return nil, fmt.Errorf("postgres DSN is required")
	}
	db, err := sql.Open(opts.DriverName, opts.DSN)
	if err != nil {
		return nil, err
	}
	if opts.MaxOpenConns > 0 {
		db.SetMaxOpenConns(opts.MaxOpenConns)
	}
	if opts.MaxIdleConns > 0 {
		db.SetMaxIdleConns(opts.MaxIdleConns)
	}
	if opts.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(opts.ConnMaxLifetime)
	}
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("postgres ping failed: %w", err)
	}
	return db, nil
}
