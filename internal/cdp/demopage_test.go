package cdp

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDemoPageHTML(t *testing.T) {
	if DemoPageHTML == "" {
		t.Error("DemoPageHTML should not be empty")
	}

	expectedStrings := []string{
		"<!DOCTYPE html>",
		"<title>browser_tail Demo</title>",
		"console.log",
		"console.warn",
		"console.error",
		"console.info",
		"throw new Error",
		"fetch(",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(DemoPageHTML, expected) {
			t.Errorf("DemoPageHTML should contain %q", expected)
		}
	}
}

func TestOpenNewTab(t *testing.T) {
	tests := []struct {
		name         string
		responseCode int
		expectError  bool
	}{
		{
			name:         "successful open",
			responseCode: http.StatusOK,
			expectError:  false,
		},
		{
			name:         "server error",
			responseCode: http.StatusInternalServerError,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if !strings.HasPrefix(r.URL.Path, "/json/new") {
					t.Errorf("expected path /json/new, got %s", r.URL.Path)
				}
				w.WriteHeader(tt.responseCode)
				w.Write([]byte(`{"id": "new-tab-id"}`))
			}))
			defer server.Close()

			port := strings.TrimPrefix(server.URL, "http://127.0.0.1:")

			err := OpenNewTab(port, "https://example.com")

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
