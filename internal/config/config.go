// Package config provides configuration management for browser_tail.
package config

import (
	"time"
)

// Version is the current version of browser_tail.
// This is set at build time via ldflags.
var Version = "dev"

// Config holds all configuration options for browser_tail.
type Config struct {
	// Connection
	ChromePort string
	AutoLaunch bool

	// Output
	OutputDir     string
	FlushInterval time.Duration
	BufferSize    int

	// Privacy & Body Capture
	Redact           bool
	CaptureBodies    bool
	BodySizeLimitKB  int
	BodyContentTypes []string

	// Event Filtering
	EnableNetwork bool
	EnableConsole bool
	EnableErrors  bool
	EnablePage    bool
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
