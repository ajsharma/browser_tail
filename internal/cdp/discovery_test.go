package cdp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDiscoverBrowserInfo(t *testing.T) {
	tests := []struct {
		name           string
		responseCode   int
		responseBody   string
		expectError    bool
		expectedBrowser string
	}{
		{
			name:         "successful response",
			responseCode: http.StatusOK,
			responseBody: `{
				"Browser": "Chrome/120.0.0.0",
				"Protocol-Version": "1.3",
				"User-Agent": "Mozilla/5.0",
				"V8-Version": "12.0.0",
				"WebKit-Version": "537.36",
				"webSocketDebuggerUrl": "ws://localhost:9222/devtools/browser/abc123"
			}`,
			expectError:     false,
			expectedBrowser: "Chrome/120.0.0.0",
		},
		{
			name:         "non-200 response",
			responseCode: http.StatusInternalServerError,
			responseBody: "error",
			expectError:  true,
		},
		{
			name:         "invalid JSON",
			responseCode: http.StatusOK,
			responseBody: "not json",
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/json/version" {
					t.Errorf("expected path /json/version, got %s", r.URL.Path)
				}
				w.WriteHeader(tt.responseCode)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			// Extract port from server URL
			port := strings.TrimPrefix(server.URL, "http://127.0.0.1:")

			info, err := DiscoverBrowserInfo(port)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if info.Browser != tt.expectedBrowser {
				t.Errorf("expected browser %q, got %q", tt.expectedBrowser, info.Browser)
			}
		})
	}
}

func TestDiscoverTabs(t *testing.T) {
	tests := []struct {
		name         string
		responseCode int
		responseBody string
		expectError  bool
		expectedTabs int
	}{
		{
			name:         "multiple tabs including non-page types",
			responseCode: http.StatusOK,
			responseBody: `[
				{"id": "tab1", "type": "page", "title": "Tab 1", "url": "https://example.com"},
				{"id": "tab2", "type": "page", "title": "Tab 2", "url": "https://google.com"},
				{"id": "bg1", "type": "background_page", "title": "Extension", "url": "chrome-extension://abc"},
				{"id": "sw1", "type": "service_worker", "title": "Worker", "url": "chrome-extension://def"}
			]`,
			expectError:  false,
			expectedTabs: 2, // Only page types
		},
		{
			name:         "empty tabs",
			responseCode: http.StatusOK,
			responseBody: `[]`,
			expectError:  false,
			expectedTabs: 0,
		},
		{
			name:         "no page tabs",
			responseCode: http.StatusOK,
			responseBody: `[
				{"id": "bg1", "type": "background_page", "title": "Extension", "url": "chrome-extension://abc"}
			]`,
			expectError:  false,
			expectedTabs: 0,
		},
		{
			name:         "non-200 response",
			responseCode: http.StatusServiceUnavailable,
			responseBody: "",
			expectError:  true,
		},
		{
			name:         "invalid JSON",
			responseCode: http.StatusOK,
			responseBody: "invalid",
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/json" {
					t.Errorf("expected path /json, got %s", r.URL.Path)
				}
				w.WriteHeader(tt.responseCode)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			port := strings.TrimPrefix(server.URL, "http://127.0.0.1:")

			tabs, err := DiscoverTabs(port)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(tabs) != tt.expectedTabs {
				t.Errorf("expected %d tabs, got %d", tt.expectedTabs, len(tabs))
			}
		})
	}
}

func TestDiscoverTabsContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[
			{"id": "ABC123", "type": "page", "title": "Example", "url": "https://example.com/path"}
		]`))
	}))
	defer server.Close()

	port := strings.TrimPrefix(server.URL, "http://127.0.0.1:")

	tabs, err := DiscoverTabs(port)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tabs) != 1 {
		t.Fatalf("expected 1 tab, got %d", len(tabs))
	}

	tab := tabs[0]
	if tab.TargetID != "ABC123" {
		t.Errorf("expected TargetID 'ABC123', got %q", tab.TargetID)
	}
	if tab.Type != "page" {
		t.Errorf("expected Type 'page', got %q", tab.Type)
	}
	if tab.Title != "Example" {
		t.Errorf("expected Title 'Example', got %q", tab.Title)
	}
	if tab.URL != "https://example.com/path" {
		t.Errorf("expected URL 'https://example.com/path', got %q", tab.URL)
	}
}

func TestCloseTabViaHTTP(t *testing.T) {
	tests := []struct {
		name         string
		targetID     string
		responseCode int
		responseBody string
		expectError  bool
	}{
		{
			name:         "successful close",
			targetID:     "ABC123",
			responseCode: http.StatusOK,
			responseBody: "Target is closing",
			expectError:  false,
		},
		{
			name:         "target not found",
			targetID:     "NOTFOUND",
			responseCode: http.StatusNotFound,
			responseBody: "No such target",
			expectError:  true,
		},
		{
			name:         "server error",
			targetID:     "ABC123",
			responseCode: http.StatusInternalServerError,
			responseBody: "Internal error",
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := "/json/close/" + tt.targetID
				if r.URL.Path != expectedPath {
					t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path)
				}
				if r.Method != http.MethodPut {
					t.Errorf("expected PUT method, got %s", r.Method)
				}
				w.WriteHeader(tt.responseCode)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			port := strings.TrimPrefix(server.URL, "http://127.0.0.1:")

			err := closeTabViaHTTP(port, tt.targetID)

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

func TestWaitForChrome(t *testing.T) {
	t.Run("chrome ready immediately", func(t *testing.T) {
		versionCalls := 0
		jsonCalls := 0

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/json/version":
				versionCalls++
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(BrowserInfo{Browser: "Chrome"})
			case "/json":
				jsonCalls++
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`[{"id": "tab1", "type": "page", "url": "https://example.com"}]`))
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()

		port := strings.TrimPrefix(server.URL, "http://127.0.0.1:")

		err := WaitForChrome(port, 5*time.Second)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if versionCalls < 1 {
			t.Error("expected at least one call to /json/version")
		}
		if jsonCalls < 1 {
			t.Error("expected at least one call to /json")
		}
	})

	t.Run("timeout when no page targets", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/json/version":
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(BrowserInfo{Browser: "Chrome"})
			case "/json":
				// Return empty array - no page targets
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`[]`))
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()

		port := strings.TrimPrefix(server.URL, "http://127.0.0.1:")

		err := WaitForChrome(port, 500*time.Millisecond)
		if err == nil {
			t.Error("expected timeout error, got nil")
		}
		if !strings.Contains(err.Error(), "no page targets") {
			t.Errorf("expected 'no page targets' error, got: %v", err)
		}
	})

	t.Run("timeout when chrome not available", func(t *testing.T) {
		// Use a port that nothing is listening on
		err := WaitForChrome("59999", 500*time.Millisecond)
		if err == nil {
			t.Error("expected timeout error, got nil")
		}
		if !strings.Contains(err.Error(), "not available") {
			t.Errorf("expected 'not available' error, got: %v", err)
		}
	})

	t.Run("waits for page target after version ready", func(t *testing.T) {
		callCount := 0

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/json/version":
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(BrowserInfo{Browser: "Chrome"})
			case "/json":
				callCount++
				w.WriteHeader(http.StatusOK)
				// Return empty on first 2 calls, then return a tab
				if callCount <= 2 {
					w.Write([]byte(`[]`))
				} else {
					w.Write([]byte(`[{"id": "tab1", "type": "page", "url": "https://example.com"}]`))
				}
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()

		port := strings.TrimPrefix(server.URL, "http://127.0.0.1:")

		err := WaitForChrome(port, 5*time.Second)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if callCount < 3 {
			t.Errorf("expected at least 3 calls to /json, got %d", callCount)
		}
	})
}

func TestTargetTypePage(t *testing.T) {
	if TargetTypePage != "page" {
		t.Errorf("expected TargetTypePage to be 'page', got %q", TargetTypePage)
	}
}

func TestTabStruct(t *testing.T) {
	tab := Tab{
		TargetID: "ABC123",
		Type:     "page",
		Title:    "Test Tab",
		URL:      "https://example.com",
	}

	if tab.TargetID != "ABC123" {
		t.Errorf("expected TargetID 'ABC123', got %q", tab.TargetID)
	}
	if tab.Type != "page" {
		t.Errorf("expected Type 'page', got %q", tab.Type)
	}
	if tab.Title != "Test Tab" {
		t.Errorf("expected Title 'Test Tab', got %q", tab.Title)
	}
	if tab.URL != "https://example.com" {
		t.Errorf("expected URL 'https://example.com', got %q", tab.URL)
	}
}

func TestBrowserInfoStruct(t *testing.T) {
	info := BrowserInfo{
		Browser:              "Chrome/120.0.0.0",
		ProtocolVersion:      "1.3",
		UserAgent:            "Mozilla/5.0",
		V8Version:            "12.0.0",
		WebKitVersion:        "537.36",
		WebSocketDebuggerURL: "ws://localhost:9222/devtools/browser/abc",
	}

	if info.Browser != "Chrome/120.0.0.0" {
		t.Errorf("expected Browser 'Chrome/120.0.0.0', got %q", info.Browser)
	}
	if info.WebSocketDebuggerURL != "ws://localhost:9222/devtools/browser/abc" {
		t.Errorf("expected WebSocketDebuggerURL, got %q", info.WebSocketDebuggerURL)
	}
}
