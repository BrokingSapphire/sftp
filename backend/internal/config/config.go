// Package config is the single source of truth for application configuration.
//
// Struct-tag conventions:
//
//	mapstructure  drives YAML file decoding AND env var name derivation.
//	              Env var = UPPER(parent_tag) + "_" + UPPER(field_tag).
//	validate      declarative validation (go-playground/validator).
//	default       value applied before any source loads (string form).
package config

import (
	"fmt"
	"strings"
	"time"

	"sapphirebroking.com/sftp_service/pkg/logger"
)

// Config is the root configuration object.
type Config struct {
	App      AppConfig            `mapstructure:"app"`
	Logging  logger.LoggingConfig `mapstructure:"logging"`
	Database DatabaseConfig       `mapstructure:"database"`
	JWT      JWTConfig            `mapstructure:"jwt"`
	Storage  StorageConfig        `mapstructure:"storage"`
	Security SecurityConfig       `mapstructure:"security"`
	CORS     CORSConfig           `mapstructure:"cors"`
	SFTP     SFTPConfig           `mapstructure:"sftp"`
	Redis    RedisConfig          `mapstructure:"redis"`
	SSO      SSOConfig            `mapstructure:"sso"`
}

// SSOConfig groups external identity-provider settings.
type SSOConfig struct {
	Microsoft MicrosoftSSOConfig `mapstructure:"microsoft"`
}

// MicrosoftSSOConfig configures Microsoft Entra ID (Azure AD) OIDC login.
type MicrosoftSSOConfig struct {
	Enabled      bool     `mapstructure:"enabled"       default:"false"`
	TenantID     string   `mapstructure:"tenant_id"     default:"organizations"`
	ClientID     string   `mapstructure:"client_id"`
	ClientSecret string   `mapstructure:"client_secret"`
	RedirectURL  string   `mapstructure:"redirect_url"  default:"http://localhost:8080/api/v1/auth/sso/microsoft/callback"`
	// SuccessURL is the frontend route the callback redirects to with tokens.
	SuccessURL string `mapstructure:"success_url" default:"http://localhost:3000/auth/sso/callback"`
	// AllowedDomains optionally restricts sign-in to specific email domains.
	AllowedDomains []string `mapstructure:"allowed_domains"`
	// DefaultRole is the role slug assigned to newly provisioned SSO users.
	DefaultRole string `mapstructure:"default_role" default:"employee"`
}

// AppConfig holds top-level application settings.
type AppConfig struct {
	Name        string `mapstructure:"name"          validate:"required" default:"sftp_service"`
	Version     string `mapstructure:"version"       validate:"required" default:"0.1.0"`
	Port        int    `mapstructure:"port"          validate:"required,min=1,max=65535" default:"8080"`
	Environment string `mapstructure:"environment"   validate:"required,oneof=local development staging production" default:"local"`
	SelfBaseURL string `mapstructure:"self_base_url" validate:"required,url" default:"http://localhost:8080"`
}

// DatabaseConfig holds the PostgreSQL connection string.
type DatabaseConfig struct {
	URL string `mapstructure:"url" validate:"required"`
}

// JWTConfig holds token settings.
type JWTConfig struct {
	Secret     string        `mapstructure:"secret"      validate:"required,min=32"`
	Issuer     string        `mapstructure:"issuer"      default:"sftp_service"`
	AccessTTL  time.Duration `mapstructure:"access_ttl"  default:"15m"`
	RefreshTTL time.Duration `mapstructure:"refresh_ttl" default:"168h"`
}

// StorageConfig holds file-storage settings.
type StorageConfig struct {
	RootPath      string `mapstructure:"root_path"       validate:"required" default:"./storage/files"`
	TempPath      string `mapstructure:"temp_path"       validate:"required" default:"./storage/tmp"`
	MaxUploadSize int64  `mapstructure:"max_upload_size" default:"0"`         // bytes; 0 = unlimited
	ChunkSize     int64  `mapstructure:"chunk_size"      default:"8388608"`   // 8 MiB
	TrashRetentionDays int `mapstructure:"trash_retention_days" default:"30"`
}

// SecurityConfig holds hardening parameters.
type SecurityConfig struct {
	PasswordMinLength int           `mapstructure:"password_min_length" default:"12"`
	MaxLoginAttempts  int           `mapstructure:"max_login_attempts"  default:"5"`
	LockoutDuration   time.Duration `mapstructure:"lockout_duration"    default:"15m"`
	RateLimitRPS      int           `mapstructure:"rate_limit_rps"      default:"20"`
	RateLimitBurst    int           `mapstructure:"rate_limit_burst"    default:"40"`
	IdleTimeout       time.Duration `mapstructure:"idle_timeout"        default:"30m"`

	// Argon2id parameters.
	ArgonMemoryKiB uint32 `mapstructure:"argon_memory_kib" default:"65536"`
	ArgonTime      uint32 `mapstructure:"argon_time"       default:"3"`
	ArgonThreads   uint8  `mapstructure:"argon_threads"    default:"4"`
	ArgonKeyLen    uint32 `mapstructure:"argon_key_len"    default:"32"`
	ArgonSaltLen   uint32 `mapstructure:"argon_salt_len"   default:"16"`
}

// CORSConfig holds cross-origin settings.
type CORSConfig struct {
	AllowedOrigins   []string `mapstructure:"allowed_origins"   validate:"required,min=1" default:"http://localhost:3000"`
	AllowedMethods   []string `mapstructure:"allowed_methods"   default:"GET,POST,PUT,PATCH,DELETE,OPTIONS"`
	AllowedHeaders   []string `mapstructure:"allowed_headers"   default:"Origin,Content-Type,Authorization,X-Request-ID,X-API-Key,X-Upload-Id,Content-Range"`
	ExposedHeaders   []string `mapstructure:"exposed_headers"   default:"X-Request-ID,Content-Range,Content-Length"`
	AllowCredentials bool     `mapstructure:"allow_credentials" default:"true"`
	MaxAgeSecs       int      `mapstructure:"max_age_secs"      default:"300"`
}

// SFTPConfig holds the embedded SSH/SFTP protocol server settings.
type SFTPConfig struct {
	Enabled     bool   `mapstructure:"enabled"       default:"true"`
	Host        string `mapstructure:"host"          default:"0.0.0.0"`
	Port        int    `mapstructure:"port"          default:"2222"`
	HostKeyPath string `mapstructure:"host_key_path" default:"./storage/ssh_host_ed25519_key"`
}

// RedisConfig holds optional Redis/Valkey settings (jobs, rate limiting).
type RedisConfig struct {
	Enabled  bool   `mapstructure:"enabled"  default:"false"`
	Address  string `mapstructure:"address"  default:"localhost:6379"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"       default:"0"`
}

// IsDevelopment reports whether the app runs in a non-production environment.
func (c *Config) IsDevelopment() bool {
	env := strings.ToLower(c.App.Environment)
	return env == "local" || env == "development" || env == "dev" || env == "staging"
}

// IsProduction reports whether the app runs in production.
func (c *Config) IsProduction() bool {
	return strings.ToLower(c.App.Environment) == "production"
}

// AppID returns "name/version".
func (c *Config) AppID() string {
	if c.App.Name == "" {
		return ""
	}
	return fmt.Sprintf("%s/%s", c.App.Name, c.App.Version)
}
