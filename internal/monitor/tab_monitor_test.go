package monitor

import (
	"testing"
	"time"

	"github.com/chromedp/cdproto/network"

	"github.com/ajsharma/browser_tail/internal/config"
)

func TestCleanExpiredRequests(t *testing.T) {
	cfg := config.DefaultConfig()
	tm := &TabMonitor{
		config:         cfg,
		requestTracker: make(map[network.RequestID]*responseInfo),
	}

	now := time.Now()

	// Add an old entry (2 minutes ago) and a fresh entry
	tm.requestTracker[network.RequestID("old-req")] = &responseInfo{
		URL:       "https://example.com/old",
		CreatedAt: now.Add(-2 * time.Minute),
	}
	tm.requestTracker[network.RequestID("fresh-req")] = &responseInfo{
		URL:       "https://example.com/fresh",
		CreatedAt: now.Add(-10 * time.Second),
	}

	if len(tm.requestTracker) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(tm.requestTracker))
	}

	// Clean entries older than 60 seconds
	tm.cleanExpiredRequests(60 * time.Second)

	if len(tm.requestTracker) != 1 {
		t.Fatalf("expected 1 entry after cleanup, got %d", len(tm.requestTracker))
	}

	if _, exists := tm.requestTracker[network.RequestID("fresh-req")]; !exists {
		t.Error("expected fresh-req to survive cleanup")
	}
	if _, exists := tm.requestTracker[network.RequestID("old-req")]; exists {
		t.Error("expected old-req to be cleaned up")
	}
}

func TestCleanExpiredRequestsEmpty(t *testing.T) {
	cfg := config.DefaultConfig()
	tm := &TabMonitor{
		config:         cfg,
		requestTracker: make(map[network.RequestID]*responseInfo),
	}

	// Should not panic on empty map
	tm.cleanExpiredRequests(60 * time.Second)

	if len(tm.requestTracker) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(tm.requestTracker))
	}
}
