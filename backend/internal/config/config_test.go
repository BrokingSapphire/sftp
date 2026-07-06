package config

import (
	"reflect"
	"testing"
)

func TestApplyDefaults(t *testing.T) {
	cfg := &Config{}
	applyDefaults(cfg)

	if cfg.App.Port != 8080 {
		t.Errorf("App.Port = %d, want 8080", cfg.App.Port)
	}
	if cfg.App.Environment != "local" {
		t.Errorf("App.Environment = %q", cfg.App.Environment)
	}
	if cfg.Storage.ChunkSize != 8388608 {
		t.Errorf("Storage.ChunkSize = %d", cfg.Storage.ChunkSize)
	}
	if cfg.Security.PasswordMinLength != 12 {
		t.Errorf("PasswordMinLength = %d", cfg.Security.PasswordMinLength)
	}
	if cfg.Security.ArgonMemoryKiB != 65536 {
		t.Errorf("ArgonMemoryKiB = %d", cfg.Security.ArgonMemoryKiB)
	}
	if cfg.SFTP.Port != 2222 {
		t.Errorf("SFTP.Port = %d", cfg.SFTP.Port)
	}
	if cfg.JWT.AccessTTL.String() != "15m0s" {
		t.Errorf("JWT.AccessTTL = %s", cfg.JWT.AccessTTL)
	}
}

func TestEnvironmentHelpers(t *testing.T) {
	if !(&Config{App: AppConfig{Environment: "local"}}).IsDevelopment() {
		t.Error("local should be development")
	}
	if !(&Config{App: AppConfig{Environment: "production"}}).IsProduction() {
		t.Error("production should be production")
	}
	if (&Config{App: AppConfig{Environment: "production"}}).IsDevelopment() {
		t.Error("production is not development")
	}
}

func TestSetFieldFromString(t *testing.T) {
	cfg := &Config{}
	if err := setFieldFromString(reflect.ValueOf(&cfg.App.Port).Elem(), "9090"); err != nil {
		t.Fatal(err)
	}
	if cfg.App.Port != 9090 {
		t.Fatalf("port override failed: %d", cfg.App.Port)
	}
	if err := setFieldFromString(reflect.ValueOf(&cfg.JWT.AccessTTL).Elem(), "1h"); err != nil {
		t.Fatal(err)
	}
	if cfg.JWT.AccessTTL.String() != "1h0m0s" {
		t.Fatalf("duration override failed: %s", cfg.JWT.AccessTTL)
	}
	if err := setFieldFromString(reflect.ValueOf(&cfg.CORS.AllowedOrigins).Elem(), "a.com, b.com"); err != nil {
		t.Fatal(err)
	}
	if len(cfg.CORS.AllowedOrigins) != 2 || cfg.CORS.AllowedOrigins[1] != "b.com" {
		t.Fatalf("slice override failed: %v", cfg.CORS.AllowedOrigins)
	}
}
