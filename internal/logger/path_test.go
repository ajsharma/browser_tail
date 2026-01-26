package logger

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestSanitizeSiteName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: UnknownSite,
		},
		{
			name:     "simple hostname",
			input:    "example.com",
			expected: "example.com",
		},
		{
			name:     "localhost with port",
			input:    "localhost:3000",
			expected: "localhost_3000",
		},
		{
			name:     "127.0.0.1 with port",
			input:    "127.0.0.1:8080",
			expected: "127.0.0.1_8080",
		},
		{
			name:     "0.0.0.0 with port",
			input:    "0.0.0.0:9000",
			expected: "0.0.0.0_9000",
		},
		{
			name:     "hostname with port (non-localhost)",
			input:    "example.com:443",
			expected: "example.com",
		},
		{
			name:     "special characters in hostname",
			input:    "test/site\\name",
			expected: "test_site_name",
		},
		{
			name:     "hostname with colon treated as port",
			input:    "example:8080",
			expected: "example",
		},
		{
			name:     "spaces",
			input:    "site with spaces",
			expected: "site_with_spaces",
		},
		{
			name:     "quotes and pipes",
			input:    "\"test\"|<site>",
			expected: "_test___site_",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeSiteName(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeSiteName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizeSiteNameTruncation(t *testing.T) {
	// Create a string longer than 255 characters
	longName := strings.Repeat("a", 300)
	result := SanitizeSiteName(longName)

	if len(result) != 255 {
		t.Errorf("SanitizeSiteName should truncate to 255 chars, got %d", len(result))
	}
}

func TestExtractSite(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: UnknownSite,
		},
		{
			name:     "simple http URL",
			input:    "http://example.com",
			expected: "example.com",
		},
		{
			name:     "https URL",
			input:    "https://example.com/path/to/page",
			expected: "example.com",
		},
		{
			name:     "URL with subdomain",
			input:    "https://www.example.com",
			expected: "www.example.com",
		},
		{
			name:     "localhost URL with port",
			input:    "http://localhost:3000/api",
			expected: "localhost_3000",
		},
		{
			name:     "127.0.0.1 URL with port",
			input:    "http://127.0.0.1:8080/",
			expected: "127.0.0.1_8080",
		},
		{
			name:     "non-localhost URL with port",
			input:    "https://api.example.com:8443/v1",
			expected: "api.example.com",
		},
		{
			name:     "about:blank",
			input:    "about:blank",
			expected: "about_blank",
		},
		{
			name:     "chrome:// URL",
			input:    "chrome://newtab",
			expected: "newtab",
		},
		{
			name:     "invalid URL",
			input:    "not a valid url ::::",
			expected: UnknownSite,
		},
		{
			name:     "file URL",
			input:    "file:///path/to/file.html",
			expected: "file_",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractSite(tt.input)
			if result != tt.expected {
				t.Errorf("ExtractSite(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGetLogPath(t *testing.T) {
	tests := []struct {
		name     string
		baseDir  string
		site     string
		tabID    string
		expected string
	}{
		{
			name:     "basic path",
			baseDir:  "/logs",
			site:     "example.com",
			tabID:    "tab-1",
			expected: filepath.Join("/logs", "example.com", "tab-1", "session.log"),
		},
		{
			name:     "relative path",
			baseDir:  "./logs",
			site:     "localhost_3000",
			tabID:    "tab-42",
			expected: filepath.Join("./logs", "localhost_3000", "tab-42", "session.log"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetLogPath(tt.baseDir, tt.site, tt.tabID)
			if result != tt.expected {
				t.Errorf("GetLogPath(%q, %q, %q) = %q, want %q",
					tt.baseDir, tt.site, tt.tabID, result, tt.expected)
			}
		})
	}
}

func TestTabRegistry(t *testing.T) {
	registry := NewTabRegistry()

	// Test that session ID is not empty
	sessionID := registry.GetSessionID()
	if sessionID == "" {
		t.Error("GetSessionID() returned empty string")
	}

	// Test GetOrCreateTabID creates new IDs
	tabID1 := registry.GetOrCreateTabID("target-123")
	if tabID1 == "" {
		t.Error("GetOrCreateTabID returned empty string")
	}

	// Test that same target gets same tab ID
	tabID1Again := registry.GetOrCreateTabID("target-123")
	if tabID1Again != tabID1 {
		t.Errorf("GetOrCreateTabID returned different ID for same target: %q vs %q", tabID1Again, tabID1)
	}

	// Test that different target gets different tab ID
	tabID2 := registry.GetOrCreateTabID("target-456")
	if tabID2 == tabID1 {
		t.Error("GetOrCreateTabID returned same ID for different targets")
	}

	// Test GetTabID
	if got := registry.GetTabID("target-123"); got != tabID1 {
		t.Errorf("GetTabID(target-123) = %q, want %q", got, tabID1)
	}

	// Test GetTabID for non-existent target
	if got := registry.GetTabID("non-existent"); got != "" {
		t.Errorf("GetTabID(non-existent) = %q, want empty string", got)
	}

	// Test RemoveTarget
	registry.RemoveTarget("target-123")
	if got := registry.GetTabID("target-123"); got != "" {
		t.Errorf("After RemoveTarget, GetTabID = %q, want empty string", got)
	}
}

func TestGenerateTabID(t *testing.T) {
	// Generate a few tab IDs and verify they are unique and sequential
	ids := make(map[string]bool)
	for i := 0; i < 10; i++ {
		id := GenerateTabID()
		if ids[id] {
			t.Errorf("GenerateTabID returned duplicate ID: %s", id)
		}
		ids[id] = true

		if !strings.HasPrefix(id, "tab-") {
			t.Errorf("GenerateTabID returned ID without 'tab-' prefix: %s", id)
		}
	}
}

func TestGetSessionIDIdempotent(t *testing.T) {
	// GetSessionID should return the same value on multiple calls
	id1 := GetSessionID()
	id2 := GetSessionID()
	if id1 != id2 {
		t.Errorf("GetSessionID returned different values: %q vs %q", id1, id2)
	}

	if id1 == "" {
		t.Error("GetSessionID returned empty string")
	}
}
