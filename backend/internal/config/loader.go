package config

import (
	"context"
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

	if err := utils.Validate(cfg); err != nil {
		return nil, fmt.Errorf("validation: %w", err)
	}

	return cfg, nil
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
