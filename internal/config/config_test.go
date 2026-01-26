package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	// Connection defaults
	if cfg.ChromePort != "9222" {
		t.Errorf("expected ChromePort 9222, got %s", cfg.ChromePort)
	}
	if cfg.AutoLaunch != false {
		t.Errorf("expected AutoLaunch false, got %v", cfg.AutoLaunch)
	}

	// Output defaults
	if cfg.OutputDir != "./logs" {
		t.Errorf("expected OutputDir ./logs, got %s", cfg.OutputDir)
	}
	if cfg.FlushInterval != 100*time.Millisecond {
		t.Errorf("expected FlushInterval 100ms, got %v", cfg.FlushInterval)
	}
	if cfg.BufferSize != 8*1024 {
		t.Errorf("expected BufferSize 8192, got %d", cfg.BufferSize)
	}

	// Privacy defaults
	if cfg.Redact != true {
		t.Errorf("expected Redact true, got %v", cfg.Redact)
	}
	if cfg.CaptureBodies != false {
		t.Errorf("expected CaptureBodies false, got %v", cfg.CaptureBodies)
	}
	if cfg.BodySizeLimitKB != 10 {
		t.Errorf("expected BodySizeLimitKB 10, got %d", cfg.BodySizeLimitKB)
	}

	// Event filtering defaults
	if cfg.EnableNetwork != true {
		t.Errorf("expected EnableNetwork true, got %v", cfg.EnableNetwork)
	}
	if cfg.EnableConsole != true {
		t.Errorf("expected EnableConsole true, got %v", cfg.EnableConsole)
	}
	if cfg.EnableErrors != true {
		t.Errorf("expected EnableErrors true, got %v", cfg.EnableErrors)
	}
	if cfg.EnablePage != true {
		t.Errorf("expected EnablePage true, got %v", cfg.EnablePage)
	}
}

func TestLoadFromFile(t *testing.T) {
	// Create temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
chrome_port: "9223"
auto_launch: true
output_dir: "./test_logs"
flush_interval: 200ms
buffer_size: 16384
redact: false
capture_bodies: true
body_size_limit_kb: 20
enable_network: true
enable_console: false
enable_errors: true
enable_page: false
`

	err := os.WriteFile(configPath, []byte(configContent), 0o644)
	if err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := LoadFromFile(configPath)
	if err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	// Verify loaded values
	if cfg.ChromePort != "9223" {
		t.Errorf("expected ChromePort 9223, got %s", cfg.ChromePort)
	}
	if cfg.AutoLaunch != true {
		t.Errorf("expected AutoLaunch true, got %v", cfg.AutoLaunch)
	}
	if cfg.OutputDir != "./test_logs" {
		t.Errorf("expected OutputDir ./test_logs, got %s", cfg.OutputDir)
	}
	if cfg.FlushInterval != 200*time.Millisecond {
		t.Errorf("expected FlushInterval 200ms, got %v", cfg.FlushInterval)
	}
	if cfg.BufferSize != 16384 {
		t.Errorf("expected BufferSize 16384, got %d", cfg.BufferSize)
	}
	if cfg.Redact != false {
		t.Errorf("expected Redact false, got %v", cfg.Redact)
	}
	if cfg.CaptureBodies != true {
		t.Errorf("expected CaptureBodies true, got %v", cfg.CaptureBodies)
	}
	if cfg.BodySizeLimitKB != 20 {
		t.Errorf("expected BodySizeLimitKB 20, got %d", cfg.BodySizeLimitKB)
	}
	if cfg.EnableConsole != false {
		t.Errorf("expected EnableConsole false, got %v", cfg.EnableConsole)
	}
	if cfg.EnablePage != false {
		t.Errorf("expected EnablePage false, got %v", cfg.EnablePage)
	}
}

func TestLoadFromFileNotFound(t *testing.T) {
	_, err := LoadFromFile("/nonexistent/config.yaml")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestLoadFromFileInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.yaml")

	err := os.WriteFile(configPath, []byte("invalid: yaml: content: ["), 0o644)
	if err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	_, err = LoadFromFile(configPath)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestLoadFromFilePartialConfig(t *testing.T) {
	// Config file with only some values should use defaults for others
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "partial.yaml")

	configContent := `
chrome_port: "9224"
output_dir: "./partial_logs"
`

	err := os.WriteFile(configPath, []byte(configContent), 0o644)
	if err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := LoadFromFile(configPath)
	if err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	// Verify specified values
	if cfg.ChromePort != "9224" {
		t.Errorf("expected ChromePort 9224, got %s", cfg.ChromePort)
	}
	if cfg.OutputDir != "./partial_logs" {
		t.Errorf("expected OutputDir ./partial_logs, got %s", cfg.OutputDir)
	}

	// Verify defaults are preserved
	if cfg.Redact != true {
		t.Errorf("expected Redact default true, got %v", cfg.Redact)
	}
	if cfg.EnableNetwork != true {
		t.Errorf("expected EnableNetwork default true, got %v", cfg.EnableNetwork)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*Config)
		wantErr bool
	}{
		{
			name:    "valid default config",
			modify:  func(c *Config) {},
			wantErr: false,
		},
		{
			name:    "empty chrome port",
			modify:  func(c *Config) { c.ChromePort = "" },
			wantErr: true,
		},
		{
			name:    "empty output dir",
			modify:  func(c *Config) { c.OutputDir = "" },
			wantErr: true,
		},
		{
			name:    "buffer size too small",
			modify:  func(c *Config) { c.BufferSize = 100 },
			wantErr: true,
		},
		{
			name:    "body size limit zero",
			modify:  func(c *Config) { c.BodySizeLimitKB = 0 },
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			tt.modify(cfg)
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
