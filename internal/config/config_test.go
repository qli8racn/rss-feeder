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
