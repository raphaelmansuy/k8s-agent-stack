// Package config handles application configuration loading and validation.
package config

import (
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all application configuration.
type Config struct {
	Environment string       `mapstructure:"environment"`
	Server      ServerConfig `mapstructure:"server"`
	Database    DBConfig     `mapstructure:"database"`
	Redis       RedisConfig  `mapstructure:"redis"`
	Telemetry   OTelConfig   `mapstructure:"telemetry"`
	Auth        AuthConfig   `mapstructure:"auth"`
	MLflow      MLflowConfig `mapstructure:"mlflow"`
}

// MLflowConfig holds MLflow connection settings.
type MLflowConfig struct {
	URL string `mapstructure:"url"`
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Port         int           `mapstructure:"port"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
}

// DBConfig holds database connection settings.
type DBConfig struct {
	URL             string        `mapstructure:"url"`
	MaxConns        int32         `mapstructure:"max_conns"`
	MinConns        int32         `mapstructure:"min_conns"`
	MaxConnLifetime time.Duration `mapstructure:"max_conn_lifetime"`
	MaxConnIdleTime time.Duration `mapstructure:"max_conn_idle_time"`
}

// RedisConfig holds Redis connection settings.
type RedisConfig struct {
	URL          string        `mapstructure:"url"`
	PoolSize     int           `mapstructure:"pool_size"`
	MinIdleConns int           `mapstructure:"min_idle_conns"`
	MaxRetries   int           `mapstructure:"max_retries"`
	DialTimeout  time.Duration `mapstructure:"dial_timeout"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

// OTelConfig holds OpenTelemetry configuration.
type OTelConfig struct {
	Enabled     bool   `mapstructure:"enabled"`
	ServiceName string `mapstructure:"service_name"`
	Endpoint    string `mapstructure:"endpoint"`
	Insecure    bool   `mapstructure:"insecure"`
}

// AuthConfig holds authentication settings.
type AuthConfig struct {
	JWTSecret     string        `mapstructure:"jwt_secret"`
	JWTExpiration time.Duration `mapstructure:"jwt_expiration"`
	APIKeyPrefix  string        `mapstructure:"api_key_prefix"`
}

// Load reads configuration from environment variables and config files.
func Load() (*Config, error) {
	v := viper.New()

	// Set defaults
	setDefaults(v)

	// Read from environment variables
	v.SetEnvPrefix("AGENTSTACK")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Try to read config file if it exists
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")
	v.AddConfigPath("/etc/agentstack")

	// Ignore file not found error
	_ = v.ReadInConfig()

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	// Environment
	v.SetDefault("environment", "development")

	// Server
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.read_timeout", 15*time.Second)
	v.SetDefault("server.write_timeout", 15*time.Second)
	v.SetDefault("server.idle_timeout", 60*time.Second)

	// Database
	v.SetDefault("database.url", "postgres://agentstack:agentstack@localhost:5432/agentstack?sslmode=disable")
	v.SetDefault("database.max_conns", 25)
	v.SetDefault("database.min_conns", 5)
	v.SetDefault("database.max_conn_lifetime", time.Hour)
	v.SetDefault("database.max_conn_idle_time", 30*time.Minute)

	// Redis
	v.SetDefault("redis.url", "redis://localhost:6379")
	v.SetDefault("redis.pool_size", 10)
	v.SetDefault("redis.min_idle_conns", 2)
	v.SetDefault("redis.max_retries", 3)
	v.SetDefault("redis.dial_timeout", 5*time.Second)
	v.SetDefault("redis.read_timeout", 3*time.Second)
	v.SetDefault("redis.write_timeout", 3*time.Second)

	// Telemetry
	v.SetDefault("telemetry.enabled", false)
	v.SetDefault("telemetry.service_name", "agentstack-api")
	v.SetDefault("telemetry.endpoint", "localhost:4318")
	v.SetDefault("telemetry.insecure", true)

	// Auth
	v.SetDefault("auth.jwt_secret", "change-me-in-production")
	v.SetDefault("auth.jwt_expiration", 24*time.Hour)
	v.SetDefault("auth.api_key_prefix", "ask_")

	// MLflow
	v.SetDefault("mlflow.url", "http://mlflow:5000")
}
