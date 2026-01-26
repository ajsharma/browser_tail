// Package config provides configuration management for browser_tail.
package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Version is the current version of browser_tail.
// This is set at build time via ldflags.
var Version = "dev"

// Config holds all configuration options for browser_tail.
type Config struct {
	// Connection
	ChromePort string `yaml:"chrome_port"`
	AutoLaunch bool   `yaml:"auto_launch"`

	// Output
	OutputDir     string        `yaml:"output_dir"`
	FlushInterval time.Duration `yaml:"flush_interval"`
	BufferSize    int           `yaml:"buffer_size"`

	// Privacy & Body Capture
	Redact           bool     `yaml:"redact"`
	CaptureBodies    bool     `yaml:"capture_bodies"`
	BodySizeLimitKB  int      `yaml:"body_size_limit_kb"`
	BodyContentTypes []string `yaml:"body_content_types"`

	// Event Filtering
	EnableNetwork bool `yaml:"enable_network"`
	EnableConsole bool `yaml:"enable_console"`
	EnableErrors  bool `yaml:"enable_errors"`
	EnablePage    bool `yaml:"enable_page"`
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		// Connection
		ChromePort: "9222",
		AutoLaunch: false,

		// Output
		OutputDir:     "./logs",
		FlushInterval: 100 * time.Millisecond,
		BufferSize:    8 * 1024, // 8 KB

		// Privacy & Body Capture
		Redact:           true,
		CaptureBodies:    false,
		BodySizeLimitKB:  10,
		BodyContentTypes: []string{"text/*", "application/json"},

		// Event Filtering
		EnableNetwork: true,
		EnableConsole: true,
		EnableErrors:  true,
		EnablePage:    true,
	}
}

// LoadFromFile loads configuration from a YAML file.
// Values from the file override the defaults.
func LoadFromFile(path string) (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return cfg, nil
}

// Validate checks the configuration for errors.
func (c *Config) Validate() error {
	if c.ChromePort == "" {
		return fmt.Errorf("chrome_port is required")
	}
	if c.OutputDir == "" {
		return fmt.Errorf("output_dir is required")
	}
	if c.BufferSize < 1024 {
		return fmt.Errorf("buffer_size must be at least 1024 bytes")
	}
	if c.BodySizeLimitKB < 1 {
		return fmt.Errorf("body_size_limit_kb must be at least 1")
	}
	return nil
}
