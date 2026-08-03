package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_NoConfigFile(t *testing.T) {
	t.Chdir(t.TempDir())

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.AnthropicAPIKey != "" {
		t.Errorf("AnthropicAPIKey: got %q, want empty", cfg.AnthropicAPIKey)
	}
}

func TestLoad_ReadsConfigFile(t *testing.T) {
	dir := t.TempDir()
	configDir := filepath.Join(dir, "internal", "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("failed to create internal/config: %v", err)
	}
	content := []byte("anthropic_api_key: \"sk-ant-test123\"\n")
	if err := os.WriteFile(filepath.Join(configDir, "config.yml"), content, 0o600); err != nil {
		t.Fatalf("failed to write config.yml: %v", err)
	}
	t.Chdir(dir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.AnthropicAPIKey != "sk-ant-test123" {
		t.Errorf("AnthropicAPIKey: got %q, want %q", cfg.AnthropicAPIKey, "sk-ant-test123")
	}
}

func TestLoad_DBDriverDefaultsToEmpty(t *testing.T) {
	t.Chdir(t.TempDir())

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.DB.Driver != "" {
		t.Errorf("DB.Driver: got %q, want empty (未設定はsqlite扱い)", cfg.DB.Driver)
	}
	if cfg.DB.IsSupabase() {
		t.Errorf("IsSupabase: got true, want false (未設定時)")
	}
}

func TestLoad_ReadsDBConfig(t *testing.T) {
	dir := t.TempDir()
	configDir := filepath.Join(dir, "internal", "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("failed to create internal/config: %v", err)
	}
	content := []byte("db:\n  driver: supabase\n  supabase:\n    dsn: \"postgres://user:pass@host:5432/postgres?sslmode=require\"\n")
	if err := os.WriteFile(filepath.Join(configDir, "config.yml"), content, 0o600); err != nil {
		t.Fatalf("failed to write config.yml: %v", err)
	}
	t.Chdir(dir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.DB.Driver != DriverSupabase {
		t.Errorf("DB.Driver: got %q, want %q", cfg.DB.Driver, DriverSupabase)
	}
	if !cfg.DB.IsSupabase() {
		t.Errorf("IsSupabase: got false, want true")
	}
	wantDSN := "postgres://user:pass@host:5432/postgres?sslmode=require"
	if cfg.DB.Supabase.DSN != wantDSN {
		t.Errorf("DB.Supabase.DSN: got %q, want %q", cfg.DB.Supabase.DSN, wantDSN)
	}
}

func TestLoad_InvalidDBDriver(t *testing.T) {
	dir := t.TempDir()
	configDir := filepath.Join(dir, "internal", "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("failed to create internal/config: %v", err)
	}
	content := []byte("db:\n  driver: supavase\n") // タイプミスを想定
	if err := os.WriteFile(filepath.Join(configDir, "config.yml"), content, 0o600); err != nil {
		t.Fatalf("failed to write config.yml: %v", err)
	}
	t.Chdir(dir)

	if _, err := Load(); err == nil {
		t.Fatal("expected error for invalid db.driver, got nil")
	}
}

func TestLoad_ReadsLogConfig(t *testing.T) {
	dir := t.TempDir()
	configDir := filepath.Join(dir, "internal", "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("failed to create internal/config: %v", err)
	}
	content := []byte("log:\n  output: stdout\n  format: json\n")
	if err := os.WriteFile(filepath.Join(configDir, "config.yml"), content, 0o600); err != nil {
		t.Fatalf("failed to write config.yml: %v", err)
	}
	t.Chdir(dir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Log.Output != "stdout" {
		t.Errorf("Log.Output: got %q, want %q", cfg.Log.Output, "stdout")
	}
	if cfg.Log.Format != "json" {
		t.Errorf("Log.Format: got %q, want %q", cfg.Log.Format, "json")
	}
}
