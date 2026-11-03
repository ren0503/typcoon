//go:build integration

package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/ren0503/tycoon/db"
)

// Harness is a disposable PostgreSQL database for integration tests.
type Harness struct {
	Pool *pgxpool.Pool
	DSN  string
}

// StartPostgres starts a container, applies all migrations, and returns a pool.
func StartPostgres(t *testing.T) *Harness {
	t.Helper()
	dsn := PostgresDSN(t)
	ApplyMigrations(t, dsn)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("open pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return &Harness{Pool: pool, DSN: dsn}
}

// ApplyMigrations runs every embedded Goose migration against the DSN.
func ApplyMigrations(t *testing.T, dsn string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open migration connection: %v", err)
	}
	defer func() { _ = sqlDB.Close() }()

	sub, err := fs.Sub(db.Migrations, "migrations")
	if err != nil {
		t.Fatalf("migration filesystem: %v", err)
	}
	provider, err := goose.NewProvider(goose.DialectPostgres, sqlDB, sub)
	if err != nil {
		t.Fatalf("goose provider: %v", err)
	}
	if _, err := provider.Up(ctx); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
}

// MigrateUp re-runs migrations and returns the applied results; used to prove
// re-running is harmless.
func MigrateUp(t *testing.T, dsn string) ([]*goose.MigrationResult, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open migration connection: %v", err)
	}
	defer func() { _ = sqlDB.Close() }()

	sub, err := fs.Sub(db.Migrations, "migrations")
	if err != nil {
		t.Fatalf("migration filesystem: %v", err)
	}
	provider, err := goose.NewProvider(goose.DialectPostgres, sqlDB, sub)
	if err != nil {
		t.Fatalf("goose provider: %v", err)
	}
	return provider.Up(ctx)
}

// TruncateAll removes all business rows so tests can share a container while
// staying isolated.
func (h *Harness) TruncateAll(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	rows, err := h.Pool.Query(ctx, `
		SELECT table_name FROM information_schema.tables
		WHERE table_schema = 'public' AND table_type = 'BASE TABLE' AND table_name <> 'goose_db_version'`)
	if err != nil {
		t.Fatalf("list tables: %v", err)
	}
	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			rows.Close()
			t.Fatalf("scan table name: %v", err)
		}
		tables = append(tables, name)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		t.Fatalf("list tables: %v", err)
	}
	if len(tables) == 0 {
		return
	}
	identifiers := make([]string, len(tables))
	for i, name := range tables {
		identifiers[i] = pgx.Identifier{name}.Sanitize()
	}
	stmt := fmt.Sprintf("TRUNCATE %s RESTART IDENTITY CASCADE", joinComma(identifiers))
	if _, err := h.Pool.Exec(ctx, stmt); err != nil {
		t.Fatalf("truncate tables: %v", err)
	}
}

func joinComma(values []string) string {
	out := ""
	for i, value := range values {
		if i > 0 {
			out += ", "
		}
		out += value
	}
	return out
}
