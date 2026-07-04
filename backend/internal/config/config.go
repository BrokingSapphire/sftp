// Package config loads and validates application configuration from
// environment variables (and an optional .env file) using Viper.
package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config is the root application configuration.
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	JWT      JWTConfig
	Storage  StorageConfig
	Redis    RedisConfig
	Log      LogConfig
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Host            string
	Port            int
	Env             string // development | production
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
	AllowedOrigins  []string
}

// DatabaseConfig holds PostgreSQL connection settings.
type DatabaseConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	Name            string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// JWTConfig holds token signing settings.
type JWTConfig struct {
	Secret          string
	AccessTTL       time.Duration
	RefreshTTL      time.Duration
	Issuer          string
}

// StorageConfig holds file-storage settings.
type StorageConfig struct {
	RootPath      string
	TempPath      string
	MaxUploadSize int64 // bytes; 0 = unlimited
	ChunkSize     int64 // bytes
}

// RedisConfig holds optional Redis settings.
type RedisConfig struct {
	Enabled  bool
	Host     string
	Port     int
	Password string
	DB       int
}

// LogConfig holds logging settings.
type LogConfig struct {
	Level  string // debug | info | warn | error
	Format string // json | console
}

// DSN returns a PostgreSQL connection string.
func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.Name, d.SSLMode,
	)
}

// Load reads configuration from the environment and an optional .env file.
func Load() (*Config, error) {
	v := viper.New()

	v.SetConfigName(".env")
	v.SetConfigType("env")
	v.AddConfigPath(".")
	v.AddConfigPath("..")
	// Missing .env file is fine — env vars still apply.
	_ = v.ReadInConfig()

	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	setDefaults(v)

	cfg := &Config{
		Server: ServerConfig{
			Host:            v.GetString("SERVER_HOST"),
			Port:            v.GetInt("SERVER_PORT"),
			Env:             v.GetString("APP_ENV"),
			ReadTimeout:     v.GetDuration("SERVER_READ_TIMEOUT"),
			WriteTimeout:    v.GetDuration("SERVER_WRITE_TIMEOUT"),
			ShutdownTimeout: v.GetDuration("SERVER_SHUTDOWN_TIMEOUT"),
			AllowedOrigins:  splitAndTrim(v.GetString("ALLOWED_ORIGINS")),
		},
		Database: DatabaseConfig{
			Host:            v.GetString("POSTGRES_HOST"),
			Port:            v.GetInt("POSTGRES_PORT"),
			User:            v.GetString("POSTGRES_USER"),
			Password:        v.GetString("POSTGRES_PASSWORD"),
			Name:            v.GetString("POSTGRES_DB"),
			SSLMode:         v.GetString("POSTGRES_SSLMODE"),
			MaxOpenConns:    v.GetInt("POSTGRES_MAX_OPEN_CONNS"),
			MaxIdleConns:    v.GetInt("POSTGRES_MAX_IDLE_CONNS"),
			ConnMaxLifetime: v.GetDuration("POSTGRES_CONN_MAX_LIFETIME"),
		},
		JWT: JWTConfig{
			Secret:     v.GetString("JWT_SECRET"),
			AccessTTL:  v.GetDuration("JWT_ACCESS_TTL"),
			RefreshTTL: v.GetDuration("JWT_REFRESH_TTL"),
			Issuer:     v.GetString("JWT_ISSUER"),
		},
		Storage: StorageConfig{
			RootPath:      v.GetString("UPLOAD_PATH"),
			TempPath:      v.GetString("TEMP_PATH"),
			MaxUploadSize: v.GetInt64("MAX_UPLOAD_SIZE"),
			ChunkSize:     v.GetInt64("CHUNK_SIZE"),
		},
		Redis: RedisConfig{
			Enabled:  v.GetBool("REDIS_ENABLED"),
			Host:     v.GetString("REDIS_HOST"),
			Port:     v.GetInt("REDIS_PORT"),
			Password: v.GetString("REDIS_PASSWORD"),
			DB:       v.GetInt("REDIS_DB"),
		},
		Log: LogConfig{
			Level:  v.GetString("LOG_LEVEL"),
			Format: v.GetString("LOG_FORMAT"),
		},
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("SERVER_HOST", "0.0.0.0")
	v.SetDefault("SERVER_PORT", 8080)
	v.SetDefault("APP_ENV", "development")
	v.SetDefault("SERVER_READ_TIMEOUT", "30s")
	v.SetDefault("SERVER_WRITE_TIMEOUT", "0s") // 0 = no timeout (large streaming downloads)
	v.SetDefault("SERVER_SHUTDOWN_TIMEOUT", "30s")
	v.SetDefault("ALLOWED_ORIGINS", "http://localhost:3000")

	v.SetDefault("POSTGRES_HOST", "localhost")
	v.SetDefault("POSTGRES_PORT", 5432)
	v.SetDefault("POSTGRES_USER", "sftp")
	v.SetDefault("POSTGRES_DB", "sftp")
	v.SetDefault("POSTGRES_SSLMODE", "disable")
	v.SetDefault("POSTGRES_MAX_OPEN_CONNS", 50)
	v.SetDefault("POSTGRES_MAX_IDLE_CONNS", 10)
	v.SetDefault("POSTGRES_CONN_MAX_LIFETIME", "1h")

	v.SetDefault("JWT_ACCESS_TTL", "15m")
	v.SetDefault("JWT_REFRESH_TTL", "168h") // 7 days
	v.SetDefault("JWT_ISSUER", "sftp")

	v.SetDefault("UPLOAD_PATH", "./storage/files")
	v.SetDefault("TEMP_PATH", "./storage/tmp")
	v.SetDefault("MAX_UPLOAD_SIZE", 0)         // unlimited
	v.SetDefault("CHUNK_SIZE", 8*1024*1024)    // 8 MiB

	v.SetDefault("REDIS_ENABLED", false)
	v.SetDefault("REDIS_HOST", "localhost")
	v.SetDefault("REDIS_PORT", 6379)
	v.SetDefault("REDIS_DB", 0)

	v.SetDefault("LOG_LEVEL", "info")
	v.SetDefault("LOG_FORMAT", "json")
}

func (c *Config) validate() error {
	if c.JWT.Secret == "" || len(c.JWT.Secret) < 32 {
		return fmt.Errorf("JWT_SECRET must be set and at least 32 characters")
	}
	if c.Database.Password == "" {
		return fmt.Errorf("POSTGRES_PASSWORD must be set")
	}
	if c.Storage.RootPath == "" {
		return fmt.Errorf("UPLOAD_PATH must be set")
	}
	return nil
}

// IsProduction reports whether the app runs in production mode.
func (c *Config) IsProduction() bool {
	return strings.EqualFold(c.Server.Env, "production")
}

func splitAndTrim(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
