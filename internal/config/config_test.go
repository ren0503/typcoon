package config

import (
	"strings"
	"testing"
	"time"
)

func setValidEnv(t *testing.T) {
	t.Helper()
	t.Setenv("APP_NAME", "test-backend")
	t.Setenv("APP_ENV", "test")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("HTTP_HOST", "127.0.0.1")
	t.Setenv("HTTP_PORT", "9090")
	t.Setenv("HTTP_READ_TIMEOUT", "2s")
	t.Setenv("HTTP_WRITE_TIMEOUT", "3s")
	t.Setenv("HTTP_IDLE_TIMEOUT", "4s")
	t.Setenv("HTTP_SHUTDOWN_TIMEOUT", "5s")
	t.Setenv("DB_HOST", "localhost")
	t.Setenv("DB_PORT", "5433")
	t.Setenv("DB_NAME", "tycoon_test")
	t.Setenv("DB_USER", "tester")
	t.Setenv("DB_PASSWORD", "super-secret-value")
	t.Setenv("DB_SSLMODE", "disable")
	t.Setenv("DB_MAX_CONNS", "8")
	t.Setenv("DB_MIN_CONNS", "1")
	t.Setenv("DB_MAX_CONN_LIFETIME", "2h")
	t.Setenv("DB_MAX_CONN_IDLE_TIME", "15m")
	t.Setenv("DB_CONNECT_TIMEOUT", "3s")
	t.Setenv("AWS_REGION", "eu-west-1")
	t.Setenv("AWS_ENDPOINT_URL", "")
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://console.example.com,https://admin.example.com")
}

func TestLoadValidConfiguration(t *testing.T) {
	setValidEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.AppName != "test-backend" {
		t.Errorf("AppName = %q", cfg.AppName)
	}
	if cfg.HTTP.Port != 9090 {
		t.Errorf("HTTP.Port = %d", cfg.HTTP.Port)
	}
	if cfg.HTTP.ReadTimeout != 2*time.Second {
		t.Errorf("HTTP.ReadTimeout = %v", cfg.HTTP.ReadTimeout)
	}
	if cfg.Database.MaxConns != 8 || cfg.Database.MinConns != 1 {
		t.Errorf("pool sizes = %d/%d", cfg.Database.MaxConns, cfg.Database.MinConns)
	}
	if cfg.Database.Password != "super-secret-value" {
		t.Errorf("password was not loaded")
	}
	if cfg.AWS.Region != "eu-west-1" {
		t.Errorf("AWS.Region = %q", cfg.AWS.Region)
	}
	if len(cfg.CORS.AllowedOrigins) != 2 {
		t.Errorf("AllowedOrigins = %v", cfg.CORS.AllowedOrigins)
	}
}

func TestLoadDefaults(t *testing.T) {
	setValidEnv(t)
	t.Setenv("APP_NAME", "")
	t.Setenv("APP_ENV", "")
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("HTTP_HOST", "")
	t.Setenv("HTTP_PORT", "")
	t.Setenv("HTTP_READ_TIMEOUT", "")
	t.Setenv("HTTP_WRITE_TIMEOUT", "")
	t.Setenv("HTTP_IDLE_TIMEOUT", "")
	t.Setenv("HTTP_SHUTDOWN_TIMEOUT", "")
	t.Setenv("DB_PORT", "")
	t.Setenv("DB_SSLMODE", "")
	t.Setenv("DB_MAX_CONNS", "")
	t.Setenv("DB_MIN_CONNS", "")
	t.Setenv("DB_MAX_CONN_LIFETIME", "")
	t.Setenv("DB_MAX_CONN_IDLE_TIME", "")
	t.Setenv("DB_CONNECT_TIMEOUT", "")
	t.Setenv("AWS_REGION", "")
	t.Setenv("CORS_ALLOWED_ORIGINS", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.AppName != "tycoon-backend" || cfg.AppEnv != "development" || cfg.LogLevel != "info" {
		t.Errorf("application defaults = %q/%q/%q", cfg.AppName, cfg.AppEnv, cfg.LogLevel)
	}
	if cfg.HTTP.Port != 8080 || cfg.HTTP.ReadTimeout != 5*time.Second {
		t.Errorf("http defaults = %d/%v", cfg.HTTP.Port, cfg.HTTP.ReadTimeout)
	}
	if cfg.Database.Port != 5432 || cfg.Database.SSLMode != "disable" || cfg.Database.MaxConns != 10 {
		t.Errorf("database defaults = %d/%q/%d", cfg.Database.Port, cfg.Database.SSLMode, cfg.Database.MaxConns)
	}
	if cfg.AWS.Region != "us-east-1" {
		t.Errorf("AWS.Region = %q", cfg.AWS.Region)
	}
	if len(cfg.CORS.AllowedOrigins) != 0 {
		t.Errorf("AllowedOrigins = %v", cfg.CORS.AllowedOrigins)
	}
}

func TestLoadRejectsMissingRequired(t *testing.T) {
	setValidEnv(t)
	t.Setenv("DB_HOST", "")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() expected an error")
	}
	if !strings.Contains(err.Error(), "DB_HOST") {
		t.Errorf("error %q does not name DB_HOST", err)
	}
	if strings.Contains(err.Error(), "super-secret-value") {
		t.Errorf("error leaked the database password: %q", err)
	}
}

func TestLoadRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
		want  string
	}{
		{name: "app env", key: "APP_ENV", value: "local", want: "APP_ENV"},
		{name: "log level", key: "LOG_LEVEL", value: "verbose", want: "LOG_LEVEL"},
		{name: "http port", key: "HTTP_PORT", value: "70000", want: "HTTP_PORT"},
		{name: "db ssl mode", key: "DB_SSLMODE", value: "preferred", want: "DB_SSLMODE"},
		{name: "duration", key: "HTTP_READ_TIMEOUT", value: "soon", want: "HTTP_READ_TIMEOUT"},
		{name: "aws endpoint", key: "AWS_ENDPOINT_URL", value: "not a url", want: "AWS_ENDPOINT_URL"},
		{name: "cors origin", key: "CORS_ALLOWED_ORIGINS", value: "not-a-url", want: "CORS_ALLOWED_ORIGINS"},
		{name: "pool bounds", key: "DB_MIN_CONNS", value: "99", want: "DB_MIN_CONNS"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setValidEnv(t)
			t.Setenv(tt.key, tt.value)

			_, err := Load()
			if err == nil {
				t.Fatal("Load() expected an error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error %q does not name %s", err, tt.want)
			}
		})
	}
}
