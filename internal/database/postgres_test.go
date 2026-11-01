package database_test

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/ren0503/tycoon/internal/config"
	"github.com/ren0503/tycoon/internal/database"
)

func TestNewFailsWhenDatabaseIsUnreachable(t *testing.T) {
	cfg := config.DatabaseConfig{
		Host:            "127.0.0.1",
		Port:            1,
		Name:            "missing",
		User:            "missing",
		Password:        "missing",
		SSLMode:         "disable",
		MaxConns:        1,
		MinConns:        0,
		MaxConnLifetime: time.Minute,
		MaxConnIdleTime: time.Minute,
		ConnectTimeout:  500 * time.Millisecond,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := database.New(ctx, cfg); err == nil {
		t.Fatal("New() expected an error")
	}
}

func TestDSN(t *testing.T) {
	cfg := config.DatabaseConfig{
		Host:     "db.internal",
		Port:     5432,
		Name:     "tycoon",
		User:     "app",
		Password: "p@ss word",
		SSLMode:  "require",
	}

	dsn := database.DSN(cfg)
	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}
	if parsed.Host != "db.internal:5432" {
		t.Errorf("host = %q", parsed.Host)
	}
	if parsed.Path != "/tycoon" {
		t.Errorf("path = %q", parsed.Path)
	}
	if got := parsed.Query().Get("sslmode"); got != "require" {
		t.Errorf("sslmode = %q", got)
	}
	password, _ := parsed.User.Password()
	if password != cfg.Password {
		t.Errorf("password = %q", password)
	}
}
