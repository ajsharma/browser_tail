package events

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestNewLogEvent(t *testing.T) {
	before := time.Now().UTC()
	event := NewLogEvent("example.com", "tab-1", "test.event", map[string]interface{}{
		"key": "value",
	})
	after := time.Now().UTC()

	if event.Site != "example.com" {
		t.Errorf("expected Site 'example.com', got %s", event.Site)
	}
	if event.TabID != "tab-1" {
		t.Errorf("expected TabID 'tab-1', got %s", event.TabID)
	}
	if event.EventType != "test.event" {
		t.Errorf("expected EventType 'test.event', got %s", event.EventType)
	}
	if event.Data["key"] != "value" {
		t.Errorf("expected Data['key'] 'value', got %v", event.Data["key"])
	}

	// Verify timestamp is valid and within range
	ts, err := time.Parse(time.RFC3339Nano, event.Timestamp)
	if err != nil {
		t.Errorf("failed to parse timestamp: %v", err)
	}
	if ts.Before(before) || ts.After(after) {
		t.Errorf("timestamp %v not in expected range [%v, %v]", ts, before, after)
	}
}

func TestLogEventJSON(t *testing.T) {
	event := NewLogEvent("example.com", "tab-1", "page.navigate", map[string]interface{}{
		"url":      "https://example.com",
		"referrer": "",
	})

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("failed to marshal event: %v", err)
	}

	// Verify JSON contains expected fields
	jsonStr := string(data)
	if !strings.Contains(jsonStr, `"site":"example.com"`) {
		t.Error("JSON missing site field")
	}
	if !strings.Contains(jsonStr, `"tab_id":"tab-1"`) {
		t.Error("JSON missing tab_id field")
	}
	if !strings.Contains(jsonStr, `"event_type":"page.navigate"`) {
		t.Error("JSON missing event_type field")
	}
	if !strings.Contains(jsonStr, `"timestamp"`) {
		t.Error("JSON missing timestamp field")
	}

	// Verify it can be unmarshaled back
	var decoded LogEvent
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal event: %v", err)
	}
	if decoded.Site != event.Site {
		t.Errorf("decoded Site mismatch: got %s, want %s", decoded.Site, event.Site)
	}
}

func TestNewSessionStartEvent(t *testing.T) {
	event := NewSessionStartEvent("session-123", 12345, "1.0.0")

	if event.EventType != EventMetaSessionStart {
		t.Errorf("expected EventType %s, got %s", EventMetaSessionStart, event.EventType)
	}
	if event.Site != "_meta" {
		t.Errorf("expected Site '_meta', got %s", event.Site)
	}
	if event.TabID != "_session" {
		t.Errorf("expected TabID '_session', got %s", event.TabID)
	}
	if event.Data["session_id"] != "session-123" {
		t.Errorf("expected session_id 'session-123', got %v", event.Data["session_id"])
	}
	if event.Data["chrome_pid"] != 12345 {
		t.Errorf("expected chrome_pid 12345, got %v", event.Data["chrome_pid"])
	}
	if event.Data["browser_tail_version"] != "1.0.0" {
		t.Errorf("expected version '1.0.0', got %v", event.Data["browser_tail_version"])
	}
}

func TestNewTabCreatedEvent(t *testing.T) {
	event := NewTabCreatedEvent("example.com", "tab-1", "session-123", "target-456", "Example", "https://example.com")

	if event.EventType != EventMetaTabCreated {
		t.Errorf("expected EventType %s, got %s", EventMetaTabCreated, event.EventType)
	}
	if event.Site != "example.com" {
		t.Errorf("expected Site 'example.com', got %s", event.Site)
	}
	if event.Data["session_id"] != "session-123" {
		t.Errorf("expected session_id 'session-123', got %v", event.Data["session_id"])
	}
	if event.Data["target_id"] != "target-456" {
		t.Errorf("expected target_id 'target-456', got %v", event.Data["target_id"])
	}
	if event.Data["title"] != "Example" {
		t.Errorf("expected title 'Example', got %v", event.Data["title"])
	}
	if event.Data["url"] != "https://example.com" {
		t.Errorf("expected url 'https://example.com', got %v", event.Data["url"])
	}
}

func TestNewTabClosedEvent(t *testing.T) {
	event := NewTabClosedEvent("example.com", "tab-1", "session-123", "target-456", 123.45)

	if event.EventType != EventMetaTabClosed {
		t.Errorf("expected EventType %s, got %s", EventMetaTabClosed, event.EventType)
	}
	if event.Data["duration_seconds"] != 123.45 {
		t.Errorf("expected duration_seconds 123.45, got %v", event.Data["duration_seconds"])
	}
}

func TestNewSiteChangedEvent(t *testing.T) {
	event := NewSiteChangedEvent("old.com", "tab-1", "new.com", "https://new.com/page")

	if event.EventType != EventMetaSiteChanged {
		t.Errorf("expected EventType %s, got %s", EventMetaSiteChanged, event.EventType)
	}
	if event.Site != "old.com" {
		t.Errorf("expected Site 'old.com', got %s", event.Site)
	}
	if event.Data["old_site"] != "old.com" {
		t.Errorf("expected old_site 'old.com', got %v", event.Data["old_site"])
	}
	if event.Data["new_site"] != "new.com" {
		t.Errorf("expected new_site 'new.com', got %v", event.Data["new_site"])
	}
	if event.Data["new_url"] != "https://new.com/page" {
		t.Errorf("expected new_url 'https://new.com/page', got %v", event.Data["new_url"])
	}
}

func TestNewSiteEnteredEvent(t *testing.T) {
	event := NewSiteEnteredEvent("new.com", "tab-1", "old.com", "https://new.com/page")

	if event.EventType != EventMetaSiteEntered {
		t.Errorf("expected EventType %s, got %s", EventMetaSiteEntered, event.EventType)
	}
	if event.Site != "new.com" {
		t.Errorf("expected Site 'new.com', got %s", event.Site)
	}
	if event.Data["from_site"] != "old.com" {
		t.Errorf("expected from_site 'old.com', got %v", event.Data["from_site"])
	}
}

func TestNewPageNavigateEvent(t *testing.T) {
	event := NewPageNavigateEvent("example.com", "tab-1", "https://example.com/page", "https://example.com", "link")

	if event.EventType != EventPageNavigate {
		t.Errorf("expected EventType %s, got %s", EventPageNavigate, event.EventType)
	}
	if event.Data["url"] != "https://example.com/page" {
		t.Errorf("expected url 'https://example.com/page', got %v", event.Data["url"])
	}
	if event.Data["referrer"] != "https://example.com" {
		t.Errorf("expected referrer 'https://example.com', got %v", event.Data["referrer"])
	}
	if event.Data["navigation_type"] != "link" {
		t.Errorf("expected navigation_type 'link', got %v", event.Data["navigation_type"])
	}
}

func TestNewPageLoadEvent(t *testing.T) {
	event := NewPageLoadEvent("example.com", "tab-1", "https://example.com")

	if event.EventType != EventPageLoad {
		t.Errorf("expected EventType %s, got %s", EventPageLoad, event.EventType)
	}
	if event.Data["url"] != "https://example.com" {
		t.Errorf("expected url 'https://example.com', got %v", event.Data["url"])
	}
}

func TestNewPageDOMReadyEvent(t *testing.T) {
	event := NewPageDOMReadyEvent("example.com", "tab-1", "https://example.com")

	if event.EventType != EventPageDOMReady {
		t.Errorf("expected EventType %s, got %s", EventPageDOMReady, event.EventType)
	}
	if event.Data["url"] != "https://example.com" {
		t.Errorf("expected url 'https://example.com', got %v", event.Data["url"])
	}
}

func TestEventTypeConstants(t *testing.T) {
	// Verify event type constants are properly namespaced
	tests := []struct {
		name     string
		constant string
		prefix   string
	}{
		{"session start", EventMetaSessionStart, "meta."},
		{"tab created", EventMetaTabCreated, "meta."},
		{"tab closed", EventMetaTabClosed, "meta."},
		{"page navigate", EventPageNavigate, "page."},
		{"page load", EventPageLoad, "page."},
		{"network request", EventNetworkRequest, "network."},
		{"network response", EventNetworkResponse, "network."},
		{"console log", EventConsoleLog, "console."},
		{"console error", EventConsoleError, "console."},
		{"error runtime", EventErrorRuntime, "error."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.HasPrefix(tt.constant, tt.prefix) {
				t.Errorf("constant %s should have prefix %s", tt.constant, tt.prefix)
			}
		})
	}
}
