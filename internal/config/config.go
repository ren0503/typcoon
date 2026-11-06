package config

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

// Config is the complete runtime configuration for the service.
type Config struct {
	AppName  string         `env:"APP_NAME" validate:"required"`
	AppEnv   string         `env:"APP_ENV" validate:"required,oneof=development test staging production"`
	LogLevel string         `env:"LOG_LEVEL" validate:"required,oneof=debug info warn error"`
	HTTP     HTTPConfig     `validate:"required"`
	Database DatabaseConfig `validate:"required"`
	AWS      AWSConfig      `validate:"required"`
	Auth     AuthConfig     `validate:"required"`
	CORS     CORSConfig
}

// HTTPConfig holds HTTP server settings.
type HTTPConfig struct {
	Host            string        `env:"HTTP_HOST" validate:"required"`
	Port            int           `env:"HTTP_PORT" validate:"required,gte=1,lte=65535"`
	ReadTimeout     time.Duration `env:"HTTP_READ_TIMEOUT" validate:"required,gt=0"`
	WriteTimeout    time.Duration `env:"HTTP_WRITE_TIMEOUT" validate:"required,gt=0"`
	IdleTimeout     time.Duration `env:"HTTP_IDLE_TIMEOUT" validate:"required,gt=0"`
	ShutdownTimeout time.Duration `env:"HTTP_SHUTDOWN_TIMEOUT" validate:"required,gt=0"`
}

// DatabaseConfig holds PostgreSQL connection and pool settings.
type DatabaseConfig struct {
	Host            string        `env:"DB_HOST" validate:"required"`
	Port            int           `env:"DB_PORT" validate:"required,gte=1,lte=65535"`
	Name            string        `env:"DB_NAME" validate:"required"`
	User            string        `env:"DB_USER" validate:"required"`
	Password        string        `env:"DB_PASSWORD" validate:"required"`
	SSLMode         string        `env:"DB_SSLMODE" validate:"required,oneof=disable require verify-ca verify-full"`
	MaxConns        int32         `env:"DB_MAX_CONNS" validate:"required,gt=0"`
	MinConns        int32         `env:"DB_MIN_CONNS" validate:"gte=0"`
	MaxConnLifetime time.Duration `env:"DB_MAX_CONN_LIFETIME" validate:"required,gt=0"`
	MaxConnIdleTime time.Duration `env:"DB_MAX_CONN_IDLE_TIME" validate:"required,gt=0"`
	ConnectTimeout  time.Duration `env:"DB_CONNECT_TIMEOUT" validate:"required,gt=0"`
}

// AWSConfig holds the AWS SDK configuration values used by the integration point.
type AWSConfig struct {
	Region      string `env:"AWS_REGION" validate:"required"`
	EndpointURL string `env:"AWS_ENDPOINT_URL" validate:"omitempty,url"`
}

// AuthConfig selects the principal provider and its development allowlist.
type AuthConfig struct {
	Mode          string   `env:"AUTH_MODE" validate:"required,oneof=dev jwt"`
	DevPrincipals []string `env:"AUTH_DEV_PRINCIPALS"`
	TrustedProxy  bool     `env:"AUTH_TRUST_PROXY"`
	Firebase      FirebaseConfig
	Token         TokenConfig     `validate:"required"`
	Signing       SigningConfig   `validate:"required"`
	RateLimit     RateLimitConfig `validate:"required"`
}

// FirebaseConfig holds Firebase Admin SDK identity and emulator settings.
type FirebaseConfig struct {
	ProjectID       string `env:"AUTH_FIREBASE_PROJECT_ID"`
	CredentialsFile string `env:"AUTH_FIREBASE_CREDENTIALS_FILE"`
	EmulatorHost    string `env:"FIREBASE_AUTH_EMULATOR_HOST"`
}

// TokenConfig holds backend-issued access and refresh token settings.
type TokenConfig struct {
	Issuer     string        `env:"AUTH_TOKEN_ISSUER" validate:"required"`
	Audience   string        `env:"AUTH_TOKEN_AUDIENCE" validate:"required"`
	AccessTTL  time.Duration `env:"AUTH_ACCESS_TTL" validate:"required,gt=0"`
	RefreshTTL time.Duration `env:"AUTH_REFRESH_TTL" validate:"required,gt=0"`
	ClockSkew  time.Duration `env:"AUTH_CLOCK_SKEW" validate:"gte=0"`
}

// SigningConfig selects asymmetric access-token signing material.
type SigningConfig struct {
	Mode         string            `env:"AUTH_SIGNING_MODE" validate:"required,oneof=local kms"`
	ActiveKID    string            `env:"AUTH_SIGNING_ACTIVE_KID" validate:"required"`
	KMSKeys      map[string]string `env:"AUTH_SIGNING_KMS_KEYS"`
	LocalKeyFile string            `env:"AUTH_SIGNING_LOCAL_KEY_FILE"`
	LocalKID     string            `env:"AUTH_SIGNING_LOCAL_KID" validate:"required"`
}

// RateLimitConfig controls shared authentication abuse limits.
type RateLimitConfig struct {
	Backend                string `env:"AUTH_RATE_LIMIT_BACKEND" validate:"required,oneof=memory redis"`
	RedisURL               string `env:"AUTH_RATE_LIMIT_REDIS_URL"`
	RedisPrefix            string `env:"AUTH_RATE_LIMIT_REDIS_PREFIX"`
	ExchangeIPPerMin       int    `env:"AUTH_RATE_LIMIT_EXCHANGE_IP_PER_MIN" validate:"required,gt=0"`
	ExchangeIdentityPerMin int    `env:"AUTH_RATE_LIMIT_EXCHANGE_IDENTITY_PER_MIN" validate:"required,gt=0"`
	RefreshIPPerMin        int    `env:"AUTH_RATE_LIMIT_REFRESH_IP_PER_MIN" validate:"required,gt=0"`
	RefreshSessionPerMin   int    `env:"AUTH_RATE_LIMIT_REFRESH_SESSION_PER_MIN" validate:"required,gt=0"`
	LinkAccountPerMin      int    `env:"AUTH_RATE_LIMIT_LINK_ACCOUNT_PER_MIN" validate:"required,gt=0"`
	LogoutAllAccountPerMin int    `env:"AUTH_RATE_LIMIT_LOGOUT_ALL_ACCOUNT_PER_MIN" validate:"required,gt=0"`
}

// CORSConfig holds the cross-origin allowlist.
type CORSConfig struct {
	AllowedOrigins []string `env:"CORS_ALLOWED_ORIGINS" validate:"omitempty,dive,url"`
}

var validate = validator.New(validator.WithRequiredStructEnabled())

func init() {
	validate.RegisterTagNameFunc(func(field reflect.StructField) string {
		name := strings.SplitN(field.Tag.Get("env"), ",", 2)[0]
		if name == "" {
			return field.Name
		}
		return name
	})
}

// Load reads configuration from process environment variables, applies defaults, and validates.
func Load() (*Config, error) {
	v := viper.New()

	bind := func(key, env string) {
		_ = v.BindEnv(key, env)
	}
	bind("app_name", "APP_NAME")
	bind("app_env", "APP_ENV")
	bind("log_level", "LOG_LEVEL")
	bind("http_host", "HTTP_HOST")
	bind("http_port", "HTTP_PORT")
	bind("http_read_timeout", "HTTP_READ_TIMEOUT")
	bind("http_write_timeout", "HTTP_WRITE_TIMEOUT")
	bind("http_idle_timeout", "HTTP_IDLE_TIMEOUT")
	bind("http_shutdown_timeout", "HTTP_SHUTDOWN_TIMEOUT")
	bind("db_host", "DB_HOST")
	bind("db_port", "DB_PORT")
	bind("db_name", "DB_NAME")
	bind("db_user", "DB_USER")
	bind("db_password", "DB_PASSWORD")
	bind("db_sslmode", "DB_SSLMODE")
	bind("db_max_conns", "DB_MAX_CONNS")
	bind("db_min_conns", "DB_MIN_CONNS")
	bind("db_max_conn_lifetime", "DB_MAX_CONN_LIFETIME")
	bind("db_max_conn_idle_time", "DB_MAX_CONN_IDLE_TIME")
	bind("db_connect_timeout", "DB_CONNECT_TIMEOUT")
	bind("aws_region", "AWS_REGION")
	bind("aws_endpoint_url", "AWS_ENDPOINT_URL")
	bind("cors_allowed_origins", "CORS_ALLOWED_ORIGINS")
	bind("auth_mode", "AUTH_MODE")
	bind("auth_dev_principals", "AUTH_DEV_PRINCIPALS")
	bind("auth_trust_proxy", "AUTH_TRUST_PROXY")
	bind("auth_firebase_project_id", "AUTH_FIREBASE_PROJECT_ID")
	bind("auth_firebase_credentials_file", "AUTH_FIREBASE_CREDENTIALS_FILE")
	bind("firebase_auth_emulator_host", "FIREBASE_AUTH_EMULATOR_HOST")
	bind("auth_token_issuer", "AUTH_TOKEN_ISSUER")
	bind("auth_token_audience", "AUTH_TOKEN_AUDIENCE")
	bind("auth_access_ttl", "AUTH_ACCESS_TTL")
	bind("auth_refresh_ttl", "AUTH_REFRESH_TTL")
	bind("auth_clock_skew", "AUTH_CLOCK_SKEW")
	bind("auth_signing_mode", "AUTH_SIGNING_MODE")
	bind("auth_signing_active_kid", "AUTH_SIGNING_ACTIVE_KID")
	bind("auth_signing_kms_keys", "AUTH_SIGNING_KMS_KEYS")
	bind("auth_signing_local_key_file", "AUTH_SIGNING_LOCAL_KEY_FILE")
	bind("auth_signing_local_kid", "AUTH_SIGNING_LOCAL_KID")
	bind("auth_rate_limit_backend", "AUTH_RATE_LIMIT_BACKEND")
	bind("auth_rate_limit_redis_url", "AUTH_RATE_LIMIT_REDIS_URL")
	bind("auth_rate_limit_redis_prefix", "AUTH_RATE_LIMIT_REDIS_PREFIX")
	bind("auth_rate_limit_exchange_ip_per_min", "AUTH_RATE_LIMIT_EXCHANGE_IP_PER_MIN")
	bind("auth_rate_limit_exchange_identity_per_min", "AUTH_RATE_LIMIT_EXCHANGE_IDENTITY_PER_MIN")
	bind("auth_rate_limit_refresh_ip_per_min", "AUTH_RATE_LIMIT_REFRESH_IP_PER_MIN")
	bind("auth_rate_limit_refresh_session_per_min", "AUTH_RATE_LIMIT_REFRESH_SESSION_PER_MIN")
	bind("auth_rate_limit_link_account_per_min", "AUTH_RATE_LIMIT_LINK_ACCOUNT_PER_MIN")
	bind("auth_rate_limit_logout_all_account_per_min", "AUTH_RATE_LIMIT_LOGOUT_ALL_ACCOUNT_PER_MIN")

	v.SetDefault("app_name", "tycoon-backend")
	v.SetDefault("app_env", "development")
	v.SetDefault("log_level", "info")
	v.SetDefault("http_host", "0.0.0.0")
	v.SetDefault("http_port", 8080)
	v.SetDefault("http_read_timeout", "5s")
	v.SetDefault("http_write_timeout", "10s")
	v.SetDefault("http_idle_timeout", "60s")
	v.SetDefault("http_shutdown_timeout", "10s")
	v.SetDefault("db_port", 5432)
	v.SetDefault("db_sslmode", "disable")
	v.SetDefault("db_max_conns", 10)
	v.SetDefault("db_min_conns", 0)
	v.SetDefault("db_max_conn_lifetime", "1h")
	v.SetDefault("db_max_conn_idle_time", "30m")
	v.SetDefault("db_connect_timeout", "5s")
	v.SetDefault("aws_region", "us-east-1")
	v.SetDefault("aws_endpoint_url", "")
	v.SetDefault("cors_allowed_origins", "")
	v.SetDefault("auth_mode", "jwt")
	v.SetDefault("auth_dev_principals", "")
	v.SetDefault("auth_trust_proxy", false)
	v.SetDefault("auth_firebase_project_id", "")
	v.SetDefault("auth_firebase_credentials_file", "")
	v.SetDefault("firebase_auth_emulator_host", "")
	v.SetDefault("auth_token_issuer", "http://localhost:8080")
	v.SetDefault("auth_token_audience", "tycoon-api")
	v.SetDefault("auth_access_ttl", "15m")
	v.SetDefault("auth_refresh_ttl", "720h")
	v.SetDefault("auth_clock_skew", "60s")
	v.SetDefault("auth_signing_mode", "local")
	v.SetDefault("auth_signing_active_kid", "local-dev-1")
	v.SetDefault("auth_signing_kms_keys", "")
	v.SetDefault("auth_signing_local_key_file", "")
	v.SetDefault("auth_signing_local_kid", "local-dev-1")
	v.SetDefault("auth_rate_limit_backend", "memory")
	v.SetDefault("auth_rate_limit_redis_url", "")
	v.SetDefault("auth_rate_limit_redis_prefix", "tycoon:auth:ratelimit:")
	v.SetDefault("auth_rate_limit_exchange_ip_per_min", 10)
	v.SetDefault("auth_rate_limit_exchange_identity_per_min", 30)
	v.SetDefault("auth_rate_limit_refresh_ip_per_min", 60)
	v.SetDefault("auth_rate_limit_refresh_session_per_min", 30)
	v.SetDefault("auth_rate_limit_link_account_per_min", 10)
	v.SetDefault("auth_rate_limit_logout_all_account_per_min", 5)

	cfg := &Config{
		AppName:  v.GetString("app_name"),
		AppEnv:   v.GetString("app_env"),
		LogLevel: v.GetString("log_level"),
		HTTP: HTTPConfig{
			Host:            v.GetString("http_host"),
			Port:            v.GetInt("http_port"),
			ReadTimeout:     v.GetDuration("http_read_timeout"),
			WriteTimeout:    v.GetDuration("http_write_timeout"),
			IdleTimeout:     v.GetDuration("http_idle_timeout"),
			ShutdownTimeout: v.GetDuration("http_shutdown_timeout"),
		},
		Database: DatabaseConfig{
			Host:            v.GetString("db_host"),
			Port:            v.GetInt("db_port"),
			Name:            v.GetString("db_name"),
			User:            v.GetString("db_user"),
			Password:        v.GetString("db_password"),
			SSLMode:         v.GetString("db_sslmode"),
			MaxConns:        v.GetInt32("db_max_conns"),
			MinConns:        v.GetInt32("db_min_conns"),
			MaxConnLifetime: v.GetDuration("db_max_conn_lifetime"),
			MaxConnIdleTime: v.GetDuration("db_max_conn_idle_time"),
			ConnectTimeout:  v.GetDuration("db_connect_timeout"),
		},
		AWS: AWSConfig{
			Region:      v.GetString("aws_region"),
			EndpointURL: v.GetString("aws_endpoint_url"),
		},
		Auth: AuthConfig{
			Mode:          v.GetString("auth_mode"),
			DevPrincipals: splitList(v.GetString("auth_dev_principals")),
			TrustedProxy:  v.GetBool("auth_trust_proxy"),
			Firebase: FirebaseConfig{
				ProjectID:       v.GetString("auth_firebase_project_id"),
				CredentialsFile: v.GetString("auth_firebase_credentials_file"),
				EmulatorHost:    v.GetString("firebase_auth_emulator_host"),
			},
			Token: TokenConfig{
				Issuer:     v.GetString("auth_token_issuer"),
				Audience:   v.GetString("auth_token_audience"),
				AccessTTL:  v.GetDuration("auth_access_ttl"),
				RefreshTTL: v.GetDuration("auth_refresh_ttl"),
				ClockSkew:  v.GetDuration("auth_clock_skew"),
			},
			Signing: SigningConfig{
				Mode:         v.GetString("auth_signing_mode"),
				ActiveKID:    v.GetString("auth_signing_active_kid"),
				KMSKeys:      splitKeyMap(v.GetString("auth_signing_kms_keys")),
				LocalKeyFile: v.GetString("auth_signing_local_key_file"),
				LocalKID:     v.GetString("auth_signing_local_kid"),
			},
			RateLimit: RateLimitConfig{
				Backend:                v.GetString("auth_rate_limit_backend"),
				RedisURL:               v.GetString("auth_rate_limit_redis_url"),
				RedisPrefix:            v.GetString("auth_rate_limit_redis_prefix"),
				ExchangeIPPerMin:       v.GetInt("auth_rate_limit_exchange_ip_per_min"),
				ExchangeIdentityPerMin: v.GetInt("auth_rate_limit_exchange_identity_per_min"),
				RefreshIPPerMin:        v.GetInt("auth_rate_limit_refresh_ip_per_min"),
				RefreshSessionPerMin:   v.GetInt("auth_rate_limit_refresh_session_per_min"),
				LinkAccountPerMin:      v.GetInt("auth_rate_limit_link_account_per_min"),
				LogoutAllAccountPerMin: v.GetInt("auth_rate_limit_logout_all_account_per_min"),
			},
		},
		CORS: CORSConfig{
			AllowedOrigins: splitOrigins(v.GetString("cors_allowed_origins")),
		},
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

const maxAccessTokenTTL = 15 * time.Minute

type rateLimitMinimum struct {
	name  string
	value int
	min   int
}

// Validate checks required values, formats, ranges, and cross-field constraints.
func (c *Config) Validate() error {
	if err := validate.Struct(c); err != nil {
		return formatValidationError(err)
	}
	if c.Database.MinConns > c.Database.MaxConns {
		return errors.New("invalid configuration: DB_MIN_CONNS must be less than or equal to DB_MAX_CONNS")
	}
	if c.Auth.Mode == "dev" {
		if c.AppEnv == "production" {
			return errors.New("invalid configuration: AUTH_MODE=dev must not be used in production")
		}
		if len(c.Auth.DevPrincipals) == 0 {
			return errors.New("invalid configuration: AUTH_MODE=dev requires a non-empty AUTH_DEV_PRINCIPALS allowlist")
		}
	}
	if err := c.validateAuth(); err != nil {
		return err
	}
	return nil
}

func (c *Config) validateAuth() error {
	if c.Auth.Token.AccessTTL > maxAccessTokenTTL {
		return errors.New("invalid configuration: AUTH_ACCESS_TTL must not exceed 15m")
	}
	if c.AppEnv != "production" && c.AppEnv != "staging" {
		return nil
	}
	if strings.TrimSpace(c.Auth.Token.Issuer) == "" {
		return errors.New("invalid configuration: AUTH_TOKEN_ISSUER is required in production/staging")
	}
	if strings.TrimSpace(c.Auth.Token.Audience) == "" {
		return errors.New("invalid configuration: AUTH_TOKEN_AUDIENCE is required in production/staging")
	}
	if strings.TrimSpace(c.Auth.Firebase.ProjectID) == "" {
		return errors.New("invalid configuration: AUTH_FIREBASE_PROJECT_ID is required in production/staging")
	}
	if strings.TrimSpace(c.Auth.Firebase.CredentialsFile) == "" {
		return errors.New("invalid configuration: AUTH_FIREBASE_CREDENTIALS_FILE is required in production/staging")
	}
	if strings.TrimSpace(c.Auth.Firebase.EmulatorHost) != "" {
		return errors.New("invalid configuration: FIREBASE_AUTH_EMULATOR_HOST must not be set in production/staging")
	}
	if c.Auth.Signing.Mode != "kms" {
		return errors.New("invalid configuration: AUTH_SIGNING_MODE must be kms in production/staging")
	}
	if strings.TrimSpace(c.Auth.Signing.LocalKeyFile) != "" {
		return errors.New("invalid configuration: AUTH_SIGNING_LOCAL_KEY_FILE must not be set in production/staging")
	}
	if len(c.Auth.Signing.KMSKeys) == 0 {
		return errors.New("invalid configuration: AUTH_SIGNING_KMS_KEYS is required in production/staging")
	}
	if _, ok := c.Auth.Signing.KMSKeys[c.Auth.Signing.ActiveKID]; !ok {
		return errors.New("invalid configuration: AUTH_SIGNING_ACTIVE_KID must reference a configured AUTH_SIGNING_KMS_KEYS entry")
	}
	if c.Auth.RateLimit.Backend != "redis" {
		return errors.New("invalid configuration: AUTH_RATE_LIMIT_BACKEND must be redis in production/staging")
	}
	if strings.TrimSpace(c.Auth.RateLimit.RedisURL) == "" {
		return errors.New("invalid configuration: AUTH_RATE_LIMIT_REDIS_URL is required in production/staging")
	}
	if strings.TrimSpace(c.Auth.RateLimit.RedisPrefix) == "" {
		return errors.New("invalid configuration: AUTH_RATE_LIMIT_REDIS_PREFIX is required in production/staging")
	}
	minimums := []rateLimitMinimum{
		{"AUTH_RATE_LIMIT_EXCHANGE_IP_PER_MIN", c.Auth.RateLimit.ExchangeIPPerMin, 10},
		{"AUTH_RATE_LIMIT_EXCHANGE_IDENTITY_PER_MIN", c.Auth.RateLimit.ExchangeIdentityPerMin, 30},
		{"AUTH_RATE_LIMIT_REFRESH_IP_PER_MIN", c.Auth.RateLimit.RefreshIPPerMin, 60},
		{"AUTH_RATE_LIMIT_REFRESH_SESSION_PER_MIN", c.Auth.RateLimit.RefreshSessionPerMin, 30},
		{"AUTH_RATE_LIMIT_LINK_ACCOUNT_PER_MIN", c.Auth.RateLimit.LinkAccountPerMin, 10},
		{"AUTH_RATE_LIMIT_LOGOUT_ALL_ACCOUNT_PER_MIN", c.Auth.RateLimit.LogoutAllAccountPerMin, 5},
	}
	for _, minimum := range minimums {
		if minimum.value < minimum.min {
			return fmt.Errorf("invalid configuration: %s must be at least %d in production/staging", minimum.name, minimum.min)
		}
	}
	return nil
}

func splitList(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func splitKeyMap(value string) map[string]string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	keys := make(map[string]string)
	for _, entry := range strings.Split(value, ",") {
		kid, arn, ok := strings.Cut(entry, "=")
		kid = strings.TrimSpace(kid)
		arn = strings.TrimSpace(arn)
		if ok && kid != "" && arn != "" {
			keys[kid] = arn
		}
	}
	if len(keys) == 0 {
		return nil
	}
	return keys
}

func splitOrigins(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	origins := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			origins = append(origins, part)
		}
	}
	if len(origins) == 0 {
		return nil
	}
	return origins
}

func formatValidationError(err error) error {
	var validationErrs validator.ValidationErrors
	if errors.As(err, &validationErrs) {
		parts := make([]string, 0, len(validationErrs))
		for _, fieldErr := range validationErrs {
			parts = append(parts, fmt.Sprintf("%s failed %s validation", fieldErr.Field(), fieldErr.Tag()))
		}
		return fmt.Errorf("invalid configuration: %s", strings.Join(parts, "; "))
	}
	return fmt.Errorf("invalid configuration: %w", err)
}
