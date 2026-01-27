package cdp

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestTestPageHTML(t *testing.T) {
	// Verify TestPageHTML is not empty
	if TestPageHTML == "" {
		t.Error("TestPageHTML should not be empty")
	}

	// Verify it contains expected elements
	expectedStrings := []string{
		"<!DOCTYPE html>",
		"<title>browser_tail",
		"browser_tail is monitoring",
		"console.log",
		"[browser_tail]",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(TestPageHTML, expected) {
			t.Errorf("TestPageHTML should contain %q", expected)
		}
	}
}

func TestTestPageHTMLBase64Encoding(t *testing.T) {
	// Verify the HTML can be base64 encoded (used for data: URL)
	encoded := base64.StdEncoding.EncodeToString([]byte(TestPageHTML))
	if encoded == "" {
		t.Error("Base64 encoding of TestPageHTML should not be empty")
	}

	// Verify it can be decoded back
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Errorf("Failed to decode base64: %v", err)
	}

	if string(decoded) != TestPageHTML {
		t.Error("Decoded HTML should match original")
	}
}

func TestIsInternalURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		// Internal URLs - should return true
		{"about:blank", "about:blank", true},
		{"data URL", "data:text/html;base64,abc123", true},
		{"data URL plain", "data:text/html,<html></html>", true},
		{"browser_tail in path", "file:///tmp/browser_tail_status.html", true},
		{"browser_tail in URL", "http://localhost/browser_tail/test", true},

		// External URLs - should return false
		{"google", "https://www.google.com", false},
		{"localhost", "http://localhost:8080", false},
		{"chrome newtab", "chrome://newtab/", false},
		{"file URL", "file:///Users/test/document.html", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isInternalURL(tt.url)
			if result != tt.expected {
				t.Errorf("isInternalURL(%q) = %v, want %v", tt.url, result, tt.expected)
			}
		})
	}
}
