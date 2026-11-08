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

func setProductionAuthEnv(t *testing.T) {
	t.Helper()
	setValidEnv(t)
	t.Setenv("APP_ENV", "production")
	t.Setenv("AUTH_MODE", "jwt")
	t.Setenv("AUTH_FIREBASE_PROJECT_ID", "prod-project")
	t.Setenv("AUTH_FIREBASE_CREDENTIALS_FILE", "/run/secrets/firebase.json")
	t.Setenv("FIREBASE_AUTH_EMULATOR_HOST", "")
	t.Setenv("AUTH_TOKEN_ISSUER", "https://api.example.com")
	t.Setenv("AUTH_TOKEN_AUDIENCE", "tycoon-api")
	t.Setenv("AUTH_ACCESS_TTL", "15m")
	t.Setenv("AUTH_REFRESH_TTL", "720h")
	t.Setenv("AUTH_CLOCK_SKEW", "60s")
	t.Setenv("AUTH_SIGNING_MODE", "kms")
	t.Setenv("AUTH_SIGNING_ACTIVE_KID", "kms-1")
	t.Setenv("AUTH_SIGNING_KMS_KEYS", "kms-1=arn:aws:kms:us-east-1:123456789012:key/abc")
	t.Setenv("AUTH_SIGNING_LOCAL_KEY_FILE", "")
	t.Setenv("AUTH_RATE_LIMIT_BACKEND", "redis")
	t.Setenv("AUTH_RATE_LIMIT_REDIS_URL", "redis://localhost:6379/0")
	t.Setenv("AUTH_RATE_LIMIT_REDIS_PREFIX", "tycoon:auth:ratelimit:")
	t.Setenv("AUTH_RATE_LIMIT_EXCHANGE_IP_PER_MIN", "10")
	t.Setenv("AUTH_RATE_LIMIT_EXCHANGE_IDENTITY_PER_MIN", "30")
	t.Setenv("AUTH_RATE_LIMIT_REFRESH_IP_PER_MIN", "60")
	t.Setenv("AUTH_RATE_LIMIT_REFRESH_SESSION_PER_MIN", "30")
	t.Setenv("AUTH_RATE_LIMIT_LINK_ACCOUNT_PER_MIN", "10")
	t.Setenv("AUTH_RATE_LIMIT_LOGOUT_ALL_ACCOUNT_PER_MIN", "5")
}

func TestLoadAuthDefaults(t *testing.T) {
	setValidEnv(t)
	for _, key := range []string{
		"AUTH_TOKEN_ISSUER", "AUTH_TOKEN_AUDIENCE", "AUTH_ACCESS_TTL", "AUTH_REFRESH_TTL",
		"AUTH_CLOCK_SKEW", "AUTH_SIGNING_MODE", "AUTH_SIGNING_ACTIVE_KID",
		"AUTH_SIGNING_LOCAL_KID", "AUTH_RATE_LIMIT_BACKEND",
		"AUTH_RATE_LIMIT_EXCHANGE_IP_PER_MIN", "AUTH_RATE_LIMIT_EXCHANGE_IDENTITY_PER_MIN",
		"AUTH_RATE_LIMIT_REFRESH_IP_PER_MIN", "AUTH_RATE_LIMIT_REFRESH_SESSION_PER_MIN",
		"AUTH_RATE_LIMIT_LINK_ACCOUNT_PER_MIN", "AUTH_RATE_LIMIT_LOGOUT_ALL_ACCOUNT_PER_MIN",
	} {
		t.Setenv(key, "")
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Auth.Token.AccessTTL != 15*time.Minute {
		t.Errorf("AccessTTL = %v", cfg.Auth.Token.AccessTTL)
	}
	if cfg.Auth.Token.RefreshTTL != 720*time.Hour {
		t.Errorf("RefreshTTL = %v", cfg.Auth.Token.RefreshTTL)
	}
	if cfg.Auth.Token.ClockSkew != 60*time.Second {
		t.Errorf("ClockSkew = %v", cfg.Auth.Token.ClockSkew)
	}
	if cfg.Auth.Signing.Mode != "local" || cfg.Auth.Signing.ActiveKID != "local-dev-1" {
		t.Errorf("signing defaults = %q/%q", cfg.Auth.Signing.Mode, cfg.Auth.Signing.ActiveKID)
	}
	if cfg.Auth.RateLimit.ExchangeIPPerMin != 10 ||
		cfg.Auth.RateLimit.ExchangeIdentityPerMin != 30 ||
		cfg.Auth.RateLimit.RefreshIPPerMin != 60 ||
		cfg.Auth.RateLimit.RefreshSessionPerMin != 30 ||
		cfg.Auth.RateLimit.LinkAccountPerMin != 10 ||
		cfg.Auth.RateLimit.LogoutAllAccountPerMin != 5 {
		t.Errorf("rate-limit defaults = %+v", cfg.Auth.RateLimit)
	}
}

func TestLoadAcceptsValidProductionConfiguration(t *testing.T) {
	setProductionAuthEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.AppEnv != "production" || cfg.Auth.Signing.Mode != "kms" {
		t.Fatalf("production config = env %q signing %q", cfg.AppEnv, cfg.Auth.Signing.Mode)
	}
}

func TestLoadRejectsInsecureOrIncompleteProductionAuth(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
		want  string
	}{
		{name: "missing firebase project", key: "AUTH_FIREBASE_PROJECT_ID", value: "", want: "AUTH_FIREBASE_PROJECT_ID"},
		{name: "missing firebase credentials", key: "AUTH_FIREBASE_CREDENTIALS_FILE", value: "", want: "AUTH_FIREBASE_CREDENTIALS_FILE"},
		{name: "firebase emulator", key: "FIREBASE_AUTH_EMULATOR_HOST", value: "localhost:9099", want: "FIREBASE_AUTH_EMULATOR_HOST"},
		{name: "local signing", key: "AUTH_SIGNING_MODE", value: "local", want: "AUTH_SIGNING_MODE"},
		{name: "local signing key file", key: "AUTH_SIGNING_LOCAL_KEY_FILE", value: "/tmp/dev-key.pem", want: "AUTH_SIGNING_LOCAL_KEY_FILE"},
		{name: "missing kms keys", key: "AUTH_SIGNING_KMS_KEYS", value: "", want: "AUTH_SIGNING_KMS_KEYS"},
		{name: "active kid not configured", key: "AUTH_SIGNING_ACTIVE_KID", value: "other", want: "AUTH_SIGNING_ACTIVE_KID"},
		{name: "memory limiter", key: "AUTH_RATE_LIMIT_BACKEND", value: "memory", want: "AUTH_RATE_LIMIT_BACKEND"},
		{name: "missing redis url", key: "AUTH_RATE_LIMIT_REDIS_URL", value: "", want: "AUTH_RATE_LIMIT_REDIS_URL"},
		{name: "missing issuer", key: "AUTH_TOKEN_ISSUER", value: " ", want: "AUTH_TOKEN_ISSUER"},
		{name: "missing audience", key: "AUTH_TOKEN_AUDIENCE", value: " ", want: "AUTH_TOKEN_AUDIENCE"},
		{name: "access ttl too long", key: "AUTH_ACCESS_TTL", value: "16m", want: "AUTH_ACCESS_TTL"},
		{name: "exchange ip below minimum", key: "AUTH_RATE_LIMIT_EXCHANGE_IP_PER_MIN", value: "9", want: "AUTH_RATE_LIMIT_EXCHANGE_IP_PER_MIN"},
		{name: "exchange identity below minimum", key: "AUTH_RATE_LIMIT_EXCHANGE_IDENTITY_PER_MIN", value: "29", want: "AUTH_RATE_LIMIT_EXCHANGE_IDENTITY_PER_MIN"},
		{name: "refresh ip below minimum", key: "AUTH_RATE_LIMIT_REFRESH_IP_PER_MIN", value: "59", want: "AUTH_RATE_LIMIT_REFRESH_IP_PER_MIN"},
		{name: "refresh session below minimum", key: "AUTH_RATE_LIMIT_REFRESH_SESSION_PER_MIN", value: "29", want: "AUTH_RATE_LIMIT_REFRESH_SESSION_PER_MIN"},
		{name: "link below minimum", key: "AUTH_RATE_LIMIT_LINK_ACCOUNT_PER_MIN", value: "9", want: "AUTH_RATE_LIMIT_LINK_ACCOUNT_PER_MIN"},
		{name: "logout all below minimum", key: "AUTH_RATE_LIMIT_LOGOUT_ALL_ACCOUNT_PER_MIN", value: "4", want: "AUTH_RATE_LIMIT_LOGOUT_ALL_ACCOUNT_PER_MIN"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setProductionAuthEnv(t)
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
