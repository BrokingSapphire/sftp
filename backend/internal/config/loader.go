package config

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"

	"sapphirebroking.com/sftp_service/internal/utils"
)

const (
	configFileEnv = "CONFIG_FILE"
	defaultConfig = "config.yaml"
)

// Load resolves configuration through the pipeline:
//
//	defaults → config file → .env → env var overrides → validate
//
// Everything is self-contained (no cloud secret managers), suitable for an
// on-premise, open-source deployment.
func Load(_ context.Context) (*Config, error) {
	cfg := &Config{}

	applyDefaults(cfg)
	slog.Info("config: defaults applied")

	if err := loadFromFile(cfg); err != nil {
		return nil, fmt.Errorf("config file: %w", err)
	}

	if err := loadDotEnv(); err != nil {
		return nil, fmt.Errorf(".env: %w", err)
	}

	applyEnvOverrides(cfg)

	// The canonical white-label brand.config.json (shared with the frontend)
	// seeds org domains and the mail From header when they are not set via
	// config/env — so one file drives both services.
	applyBrandConfig(cfg)

	// Lock SSO down to the organisation: if Microsoft SSO is enabled but no
	// explicit allow-list was given, restrict sign-in to the org's own email
	// domains so outsiders (guest/personal accounts) are rejected by default.
	if cfg.SSO.Microsoft.Enabled && len(cfg.SSO.Microsoft.AllowedDomains) == 0 && len(cfg.OrgDomains) > 0 {
		cfg.SSO.Microsoft.AllowedDomains = cfg.OrgDomains
		slog.Info("config: SSO restricted to org domains", "domains", cfg.OrgDomains)
	}

	if err := utils.Validate(cfg); err != nil {
		return nil, fmt.Errorf("validation: %w", err)
	}

	return cfg, nil
}

// applyBrandConfig reads BRAND_CONFIG_PATH (default /app/brand.config.json) and
// fills OrgDomains / Mail.From from it unless already configured. Missing or
// invalid files are ignored (env/config still win).
func applyBrandConfig(cfg *Config) {
	path := os.Getenv("BRAND_CONFIG_PATH")
	if path == "" {
		path = "/app/brand.config.json"
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var brand struct {
		Org struct {
			Domains []string `json:"domains"`
		} `json:"org"`
		Mail struct {
			From string `json:"from"`
		} `json:"mail"`
		SMTP struct {
			Enabled  bool   `json:"enabled"`
			Host     string `json:"host"`
			Port     int    `json:"port"`
			Username string `json:"username"`
			Password string `json:"password"`
			StartTLS bool   `json:"startTls"`
		} `json:"smtp"`
		SSO struct {
			Microsoft struct {
				Enabled        bool     `json:"enabled"`
				TenantID       string   `json:"tenantId"`
				ClientID       string   `json:"clientId"`
				ClientSecret   string   `json:"clientSecret"`
				RedirectURL    string   `json:"redirectUrl"`
				SuccessURL     string   `json:"successUrl"`
				AllowedDomains []string `json:"allowedDomains"`
				DefaultRole    string   `json:"defaultRole"`
			} `json:"microsoft"`
		} `json:"sso"`
		AI struct {
			Enabled    bool   `json:"enabled"`
			OllamaURL  string `json:"ollamaUrl"`
			EmbedModel string `json:"embedModel"`
			ChatModel  string `json:"chatModel"`
		} `json:"ai"`
		Editor struct {
			Enabled         bool   `json:"enabled"`
			DocServerURL    string `json:"docServerUrl"`
			JWTSecret       string `json:"jwtSecret"`
			InternalBaseURL string `json:"internalBaseUrl"`
		} `json:"editor"`
	}
	if json.Unmarshal(raw, &brand) != nil {
		return
	}

	// AI + Office editor — brand config drives them unless already enabled via env.
	if !cfg.AI.Enabled && brand.AI.Enabled {
		cfg.AI.Enabled = true
		if brand.AI.OllamaURL != "" {
			cfg.AI.OllamaURL = brand.AI.OllamaURL
		}
		if brand.AI.EmbedModel != "" {
			cfg.AI.EmbedModel = brand.AI.EmbedModel
		}
		if brand.AI.ChatModel != "" {
			cfg.AI.ChatModel = brand.AI.ChatModel
		}
	}
	if !cfg.Editor.Enabled && brand.Editor.Enabled {
		cfg.Editor.Enabled = true
		cfg.Editor.DocServerURL = brand.Editor.DocServerURL
		cfg.Editor.JWTSecret = brand.Editor.JWTSecret
		if brand.Editor.InternalBaseURL != "" {
			cfg.Editor.InternalBaseURL = brand.Editor.InternalBaseURL
		}
	}

	if len(cfg.OrgDomains) == 0 && len(brand.Org.Domains) > 0 {
		cfg.OrgDomains = brand.Org.Domains
	}
	if cfg.Mail.From == "" && brand.Mail.From != "" {
		cfg.Mail.From = brand.Mail.From
	}

	// SMTP credentials — apply from the brand config unless a host was already
	// configured via config/env.
	if cfg.Mail.Host == "" && brand.SMTP.Host != "" {
		cfg.Mail.Enabled = brand.SMTP.Enabled
		cfg.Mail.Host = brand.SMTP.Host
		if brand.SMTP.Port != 0 {
			cfg.Mail.Port = brand.SMTP.Port
		}
		cfg.Mail.Username = brand.SMTP.Username
		cfg.Mail.Password = brand.SMTP.Password
		cfg.Mail.StartTLS = brand.SMTP.StartTLS
	}

	// Microsoft SSO — apply unless a client id was already configured.
	if cfg.SSO.Microsoft.ClientID == "" && brand.SSO.Microsoft.ClientID != "" {
		m := &cfg.SSO.Microsoft
		m.Enabled = brand.SSO.Microsoft.Enabled
		m.ClientID = brand.SSO.Microsoft.ClientID
		m.ClientSecret = brand.SSO.Microsoft.ClientSecret
		if brand.SSO.Microsoft.TenantID != "" {
			m.TenantID = brand.SSO.Microsoft.TenantID
		}
		if brand.SSO.Microsoft.RedirectURL != "" {
			m.RedirectURL = brand.SSO.Microsoft.RedirectURL
		}
		if brand.SSO.Microsoft.SuccessURL != "" {
			m.SuccessURL = brand.SSO.Microsoft.SuccessURL
		}
		if len(brand.SSO.Microsoft.AllowedDomains) > 0 {
			m.AllowedDomains = brand.SSO.Microsoft.AllowedDomains
		}
		if brand.SSO.Microsoft.DefaultRole != "" {
			m.DefaultRole = brand.SSO.Microsoft.DefaultRole
		}
	}

	slog.Info("config: brand overrides applied", "path", path,
		"org_domains", cfg.OrgDomains, "mail_enabled", cfg.Mail.Enabled, "sso_enabled", cfg.SSO.Microsoft.Enabled)
}

func applyDefaults(cfg *Config) {
	applyDefaultsRecursive(reflect.ValueOf(cfg).Elem())
}

func applyDefaultsRecursive(v reflect.Value) {
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		fv := v.Field(i)
		ft := t.Field(i)

		if fv.Kind() == reflect.Struct && fv.CanSet() {
			applyDefaultsRecursive(fv)
			continue
		}
		if !fv.IsZero() {
			continue
		}
		def := ft.Tag.Get("default")
		if def == "" {
			continue
		}
		if err := setFieldFromString(fv, def); err != nil {
			panic(fmt.Sprintf("config: bad default for %s: %v", ft.Name, err))
		}
	}
}

func loadFromFile(cfg *Config) error {
	filePath := os.Getenv(configFileEnv)
	explicit := filePath != ""
	if !explicit {
		filePath = defaultConfig
	}

	v := viper.New()
	v.SetConfigFile(filePath)
	v.SetConfigType("yaml")

	if err := v.ReadInConfig(); err != nil {
		if !explicit && isFileNotFound(err) {
			slog.Warn("config: no config file found, using defaults and env vars only", "path", filePath)
			return nil
		}
		return fmt.Errorf("reading %q: %w", filePath, err)
	}
	slog.Info("config: loaded config file", "path", filePath)

	dec, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:           cfg,
		TagName:          "mapstructure",
		WeaklyTypedInput: true,
		ZeroFields:       false,
		DecodeHook:       mapstructure.ComposeDecodeHookFunc(mapstructure.StringToTimeDurationHookFunc()),
	})
	if err != nil {
		return fmt.Errorf("decoder init: %w", err)
	}
	return dec.Decode(v.AllSettings())
}

func isFileNotFound(err error) bool {
	if errors.Is(err, os.ErrNotExist) {
		return true
	}
	var nf viper.ConfigFileNotFoundError
	return errors.As(err, &nf)
}

func loadDotEnv() error {
	file := ".env"
	if f := os.Getenv("ENV_FILE"); f != "" {
		file = f
	}
	if err := godotenv.Load(file); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			slog.Warn("config: no env file found, skipping", "path", file)
			return nil
		}
		return err
	}
	slog.Info("config: loaded env file", "path", file)
	return nil
}

func applyEnvOverrides(cfg *Config) {
	applyEnvRecursive(reflect.ValueOf(cfg).Elem(), "")
}

func applyEnvRecursive(v reflect.Value, prefix string) {
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		fv := v.Field(i)
		ft := t.Field(i)

		tag := ft.Tag.Get("mapstructure")
		if tag == "" || tag == "-" {
			continue
		}
		name, opts, _ := strings.Cut(tag, ",")

		if fv.Kind() == reflect.Struct && fv.CanSet() {
			key := prefix
			if !strings.Contains(opts, "squash") {
				key = joinKey(prefix, name)
			}
			applyEnvRecursive(fv, key)
			continue
		}

		key := joinKey(prefix, name)
		if val := os.Getenv(key); val != "" {
			if err := setFieldFromString(fv, val); err != nil {
				slog.Warn("config: failed to parse env var, skipping", "key", key, "error", err)
			}
		}
	}
}

func joinKey(prefix, name string) string {
	key := strings.ToUpper(name)
	if prefix != "" {
		return prefix + "_" + key
	}
	return key
}

func setFieldFromString(fv reflect.Value, s string) error {
	if !fv.CanSet() {
		return fmt.Errorf("field is not settable")
	}

	if fv.Type() == reflect.TypeOf(time.Duration(0)) {
		d, err := time.ParseDuration(s)
		if err != nil {
			return err
		}
		fv.Set(reflect.ValueOf(d))
		return nil
	}

	switch fv.Kind() {
	case reflect.String:
		fv.SetString(s)
	case reflect.Bool:
		b, err := strconv.ParseBool(s)
		if err != nil {
			return err
		}
		fv.SetBool(b)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return err
		}
		fv.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return err
		}
		fv.SetUint(n)
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return err
		}
		fv.SetFloat(f)
	case reflect.Slice:
		if fv.Type().Elem().Kind() == reflect.String {
			parts := strings.Split(s, ",")
			for i := range parts {
				parts[i] = strings.TrimSpace(parts[i])
			}
			fv.Set(reflect.ValueOf(parts))
		} else {
			return fmt.Errorf("unsupported slice element type: %s", fv.Type().Elem().Kind())
		}
	default:
		return fmt.Errorf("unsupported field type: %s", fv.Kind())
	}
	return nil
}
