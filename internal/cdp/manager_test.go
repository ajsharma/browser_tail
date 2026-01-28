package cdp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ajsharma/browser_tail/internal/config"
	"github.com/ajsharma/browser_tail/internal/logger"
)

func TestNewManager(t *testing.T) {
	cfg := &config.Config{
		ChromePort: "9222",
		OutputDir:  "./logs",
	}
	fm := logger.NewFileManager("./logs")

	m := NewManager(cfg, fm)

	if m == nil {
		t.Fatal("NewManager returned nil")
	}
	if m.config != cfg {
		t.Error("config not set correctly")
	}
	if m.fileManager != fm {
		t.Error("fileManager not set correctly")
	}
	if m.tabMonitors == nil {
		t.Error("tabMonitors map not initialized")
	}
	if m.tabRegistry == nil {
		t.Error("tabRegistry not initialized")
	}
}

func TestManagerGetActiveTabCount(t *testing.T) {
	cfg := &config.Config{
		ChromePort: "9222",
		OutputDir:  "./logs",
	}
	fm := logger.NewFileManager("./logs")
	m := NewManager(cfg, fm)

	// Initially should be 0
	if count := m.GetActiveTabCount(); count != 0 {
		t.Errorf("expected 0 active tabs, got %d", count)
	}
}

func TestManagerIsConnected(t *testing.T) {
	cfg := &config.Config{
		ChromePort: "9222",
		OutputDir:  "./logs",
	}
	fm := logger.NewFileManager("./logs")
	m := NewManager(cfg, fm)

	// Initially should be false
	if m.IsConnected() {
		t.Error("expected IsConnected to be false initially")
	}
}

func TestManagerConcurrentAccess(t *testing.T) {
	cfg := &config.Config{
		ChromePort: "9222",
		OutputDir:  "./logs",
	}
	fm := logger.NewFileManager("./logs")
	m := NewManager(cfg, fm)

	// Test concurrent access to GetActiveTabCount and IsConnected
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			_ = m.GetActiveTabCount()
		}()
		go func() {
			defer wg.Done()
			_ = m.IsConnected()
		}()
	}
	wg.Wait()
}

func TestCleanupOrphanedAnchorTabsFiltering(t *testing.T) {
	// Test the filtering logic that determines which tabs to clean up
	tests := []struct {
		name             string
		tabs             []*Tab
		internalTargetID string
		expectedCleanups []string // target IDs that should be closed
	}{
		{
			name: "no orphaned tabs",
			tabs: []*Tab{
				{TargetID: "INTERNAL123456789", URL: "about:blank"},
				{TargetID: "USER45678901234", URL: "https://example.com"},
			},
			internalTargetID: "INTERNAL123456789",
			expectedCleanups: []string{},
		},
		{
			name: "one orphaned about:blank",
			tabs: []*Tab{
				{TargetID: "INTERNAL123456789", URL: "about:blank"},
				{TargetID: "ORPHAN78901234567", URL: "about:blank"},
				{TargetID: "USER45678901234", URL: "https://example.com"},
			},
			internalTargetID: "INTERNAL123456789",
			expectedCleanups: []string{"ORPHAN78901234567"},
		},
		{
			name: "multiple orphaned about:blank",
			tabs: []*Tab{
				{TargetID: "ORPHAN11234567890", URL: "about:blank"},
				{TargetID: "ORPHAN21234567890", URL: "about:blank"},
				{TargetID: "INTERNAL123456789", URL: "about:blank"},
				{TargetID: "ORPHAN31234567890", URL: "about:blank"},
			},
			internalTargetID: "INTERNAL123456789",
			expectedCleanups: []string{"ORPHAN11234567890", "ORPHAN21234567890", "ORPHAN31234567890"},
		},
		{
			name: "no about:blank tabs except internal",
			tabs: []*Tab{
				{TargetID: "INTERNAL123456789", URL: "about:blank"},
				{TargetID: "USER11234567890", URL: "https://example.com"},
				{TargetID: "USER21234567890", URL: "https://google.com"},
			},
			internalTargetID: "INTERNAL123456789",
			expectedCleanups: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Track which tabs were attempted to be closed
			closedTabs := make([]string, 0)
			var mu sync.Mutex

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.HasPrefix(r.URL.Path, "/json/close/") {
					targetID := strings.TrimPrefix(r.URL.Path, "/json/close/")
					mu.Lock()
					closedTabs = append(closedTabs, targetID)
					mu.Unlock()
					w.WriteHeader(http.StatusOK)
					return
				}
				w.WriteHeader(http.StatusNotFound)
			}))
			defer server.Close()

			port := strings.TrimPrefix(server.URL, "http://127.0.0.1:")

			cfg := &config.Config{
				ChromePort: port,
				OutputDir:  "./logs",
			}
			fm := logger.NewFileManager("./logs")
			m := NewManager(cfg, fm)
			m.internalTargetID = tt.internalTargetID

			m.cleanupOrphanedAnchorTabs(tt.tabs)

			// Verify the correct tabs were closed
			if len(closedTabs) != len(tt.expectedCleanups) {
				t.Errorf("expected %d cleanups, got %d", len(tt.expectedCleanups), len(closedTabs))
			}

			for _, expected := range tt.expectedCleanups {
				found := false
				for _, closed := range closedTabs {
					if closed == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected tab %s to be closed, but it wasn't", expected)
				}
			}
		})
	}
}

func TestManagerStopWithoutStart(t *testing.T) {
	cfg := &config.Config{
		ChromePort: "9222",
		OutputDir:  "./logs",
	}
	fm := logger.NewFileManager("./logs")
	m := NewManager(cfg, fm)

	// Stop should not panic even when never started
	m.Stop()
}

func TestClearTabMonitors(t *testing.T) {
	cfg := &config.Config{
		ChromePort: "9222",
		OutputDir:  "./logs",
	}
	fm := logger.NewFileManager("./logs")
	m := NewManager(cfg, fm)

	// clearTabMonitors should work on empty map
	m.clearTabMonitors()

	if count := m.GetActiveTabCount(); count != 0 {
		t.Errorf("expected 0 tabs after clear, got %d", count)
	}
}

func TestManagerContextCancellation(t *testing.T) {
	cfg := &config.Config{
		ChromePort: "59999", // Port nothing is listening on
		OutputDir:  "./logs",
		AutoLaunch: false,
	}
	fm := logger.NewFileManager("./logs")
	m := NewManager(cfg, fm)

	ctx, cancel := context.WithCancel(context.Background())

	// Start in goroutine
	done := make(chan error, 1)
	go func() {
		done <- m.Start(ctx)
	}()

	// Give it a moment to start trying to connect
	time.Sleep(100 * time.Millisecond)

	// Cancel context
	cancel()

	// Should return without hanging
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("Start did not return after context cancellation")
	}
}

func TestReconnectIntervalConstants(t *testing.T) {
	if reconnectInterval <= 0 {
		t.Error("reconnectInterval should be positive")
	}
	if maxReconnectWait <= reconnectInterval {
		t.Error("maxReconnectWait should be greater than reconnectInterval")
	}
}
