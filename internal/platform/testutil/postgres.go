//go:build integration

package testutil

import (
	"context"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/ren0503/tycoon/internal/platform/config"
)

// PostgresDSN starts a disposable PostgreSQL container and returns a DSN for it.
// The container is terminated when the test completes.
func PostgresDSN(t *testing.T) string {
	t.Helper()
	testcontainers.SkipIfProviderIsNotHealthy(t)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	t.Cleanup(cancel)

	container, err := postgres.Run(
		ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("tycoon_test"),
		postgres.WithUsername("tycoon"),
		postgres.WithPassword("tycoon"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(90*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}
	t.Cleanup(func() {
		if err := testcontainers.TerminateContainer(container); err != nil {
			t.Logf("terminate postgres container: %v", err)
		}
	})

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("build postgres connection string: %v", err)
	}
	return dsn
}

// PostgresConfig starts a disposable PostgreSQL container and returns database
// configuration pointing at it.
func PostgresConfig(t *testing.T) config.DatabaseConfig {
	t.Helper()

	parsed, err := url.Parse(PostgresDSN(t))
	if err != nil {
		t.Fatalf("parse postgres connection string: %v", err)
	}
	port, err := strconv.Atoi(parsed.Port())
	if err != nil {
		t.Fatalf("parse postgres port: %v", err)
	}
	password, _ := parsed.User.Password()

	return config.DatabaseConfig{
		Host:            parsed.Hostname(),
		Port:            port,
		Name:            strings.TrimPrefix(parsed.Path, "/"),
		User:            parsed.User.Username(),
		Password:        password,
		SSLMode:         "disable",
		MaxConns:        5,
		MinConns:        1,
		MaxConnLifetime: time.Hour,
		MaxConnIdleTime: time.Minute,
		ConnectTimeout:  10 * time.Second,
	}
}
