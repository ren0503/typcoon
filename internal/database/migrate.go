package database

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/ren0503/tycoon/db"
)

const migrationsDir = "migrations"

// RunMigrations executes a Goose command against the embedded migration set.
func RunMigrations(ctx context.Context, dsn, command string) error {
	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("open migration connection: %w", err)
	}
	defer func() { _ = sqlDB.Close() }()

	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("ping migration connection: %w", err)
	}

	goose.SetBaseFS(db.Migrations)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}

	switch command {
	case "up":
		return goose.UpContext(ctx, sqlDB, migrationsDir)
	case "down":
		return goose.DownContext(ctx, sqlDB, migrationsDir)
	case "status":
		return goose.StatusContext(ctx, sqlDB, migrationsDir)
	case "version":
		version, err := goose.GetDBVersionContext(ctx, sqlDB)
		if err != nil {
			return fmt.Errorf("read migration version: %w", err)
		}
		fmt.Printf("goose: version %d\n", version)
		return nil
	default:
		return fmt.Errorf("unknown migration command %q (expected up, down, status, or version)", command)
	}
}
