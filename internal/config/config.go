package config

import (
	"fmt"
	"strings"

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

// Config is the root typed configuration struct.
// All sub-configs are populated by Load() from env vars and/or a YAML file.
type Config struct {
	Server    ServerConfig    `mapstructure:"server"`
	Postgres  PostgresConfig  `mapstructure:"postgres"`
	Redis     RedisConfig     `mapstructure:"redis"`
	RabbitMQ  RabbitMQConfig  `mapstructure:"rabbitmq"`
	Telemetry TelemetryConfig `mapstructure:"telemetry"`
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

	return &cfg, nil
}
