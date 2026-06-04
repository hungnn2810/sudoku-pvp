package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Port int `mapstructure:"port"`
}

// PostgresConfig holds PostgreSQL connection settings.
type PostgresConfig struct {
	DSN      string `mapstructure:"dsn"`
	MaxConns int32  `mapstructure:"max_conns"`
	MinConns int32  `mapstructure:"min_conns"`
}

// RedisConfig holds Redis client settings.
type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
	PoolSize int    `mapstructure:"pool_size"`
}

// RabbitMQConfig holds AMQP connection settings.
type RabbitMQConfig struct {
	URL string `mapstructure:"url"`
}

// TelemetryConfig holds OpenTelemetry settings.
type TelemetryConfig struct {
	OTLPEndpoint string `mapstructure:"otlp_endpoint"`
}

// AuthConfig holds JWT signing and token TTL settings.
// D-04: HS256 signing; secret from env var SUDOKU_AUTH_JWT_SECRET.
// D-02: AccessTokenTTL default 15 minutes; RefreshTokenTTL default 30 days.
// GoogleClientID is required for Google ID token audience validation (CR-02).
// Set via env var SUDOKU_AUTH_GOOGLE_CLIENT_ID.
type AuthConfig struct {
	JWTSecret       string        `mapstructure:"jwt_secret"`
	AccessTokenTTL  time.Duration `mapstructure:"access_token_ttl"`
	RefreshTokenTTL time.Duration `mapstructure:"refresh_token_ttl"`
	GoogleClientID  string        `mapstructure:"google_client_id"`
}

// Config is the root typed configuration struct.
// All sub-configs are populated by Load() from env vars and/or a YAML file.
type Config struct {
	Server    ServerConfig    `mapstructure:"server"`
	Postgres  PostgresConfig  `mapstructure:"postgres"`
	Redis     RedisConfig     `mapstructure:"redis"`
	RabbitMQ  RabbitMQConfig  `mapstructure:"rabbitmq"`
	Telemetry TelemetryConfig `mapstructure:"telemetry"`
	Auth      AuthConfig      `mapstructure:"auth"`
	LogLevel  string          `mapstructure:"log_level"`
}

// Load reads configuration from the YAML file (deployments/docker/config.yaml or ./config.yaml)
// and overlays env vars with the SUDOKU_ prefix. Env vars take precedence over file values
// (12-factor compliant). Required fields are validated after unmarshal.
func Load() (*Config, error) {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath("./deployments/docker")
	v.AddConfigPath(".")

	// Defaults — applied before env vars and file values.
	v.SetDefault("server.port", 8080)

	// Env vars override file values — 12-factor.
	// SUDOKU_POSTGRES_DSN maps to postgres.dsn, etc.
	v.SetEnvPrefix("SUDOKU")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// ReadInConfig is best-effort; production deployments may use env vars only.
	_ = v.ReadInConfig()

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("config unmarshal: %w", err)
	}

	// Validate required fields.
	if cfg.Postgres.DSN == "" {
		return nil, fmt.Errorf("postgres DSN is required")
	}
	if cfg.Redis.Addr == "" {
		return nil, fmt.Errorf("redis addr is required")
	}
	if cfg.RabbitMQ.URL == "" {
		return nil, fmt.Errorf("rabbitmq URL is required")
	}
	if cfg.Server.Port <= 0 || cfg.Server.Port > 65535 {
		return nil, fmt.Errorf("server.port must be between 1 and 65535, got %d", cfg.Server.Port)
	}
	if cfg.Auth.JWTSecret == "" {
		return nil, fmt.Errorf("auth.jwt_secret is required")
	}
	if cfg.Auth.GoogleClientID == "" {
		return nil, fmt.Errorf("auth.google_client_id is required (set SUDOKU_AUTH_GOOGLE_CLIENT_ID)")
	}
	if cfg.Auth.AccessTokenTTL <= 0 {
		cfg.Auth.AccessTokenTTL = 15 * time.Minute
	}
	if cfg.Auth.RefreshTokenTTL <= 0 {
		cfg.Auth.RefreshTokenTTL = 30 * 24 * time.Hour
	}

	return &cfg, nil
}
