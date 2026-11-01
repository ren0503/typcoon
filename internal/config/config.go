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
		CORS: CORSConfig{
			AllowedOrigins: splitOrigins(v.GetString("cors_allowed_origins")),
		},
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Validate checks required values, formats, ranges, and cross-field constraints.
func (c *Config) Validate() error {
	if err := validate.Struct(c); err != nil {
		return formatValidationError(err)
	}
	if c.Database.MinConns > c.Database.MaxConns {
		return errors.New("invalid configuration: DB_MIN_CONNS must be less than or equal to DB_MAX_CONNS")
	}
	return nil
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
