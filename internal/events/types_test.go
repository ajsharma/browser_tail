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
	data, ok := event.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected Data to be map[string]interface{}, got %T", event.Data)
	}
	if data["key"] != "value" {
		t.Errorf("expected Data['key'] 'value', got %v", data["key"])
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
	event := NewLogEvent("example.com", "tab-1", "page.navigate", &PageNavigateData{
		URL:      "https://example.com",
		Referrer: "",
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

	data, ok := event.Data.(*SessionStartData)
	if !ok {
		t.Fatalf("expected Data to be *SessionStartData, got %T", event.Data)
	}
	if data.SessionID != "session-123" {
		t.Errorf("expected SessionID 'session-123', got %v", data.SessionID)
	}
	if data.ChromePID != 12345 {
		t.Errorf("expected ChromePID 12345, got %v", data.ChromePID)
	}
	if data.BrowserTailVersion != "1.0.0" {
		t.Errorf("expected BrowserTailVersion '1.0.0', got %v", data.BrowserTailVersion)
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

	data, ok := event.Data.(*TabCreatedData)
	if !ok {
		t.Fatalf("expected Data to be *TabCreatedData, got %T", event.Data)
	}
	if data.SessionID != "session-123" {
		t.Errorf("expected SessionID 'session-123', got %v", data.SessionID)
	}
	if data.TargetID != "target-456" {
		t.Errorf("expected TargetID 'target-456', got %v", data.TargetID)
	}
	if data.Title != "Example" {
		t.Errorf("expected Title 'Example', got %v", data.Title)
	}
	if data.URL != "https://example.com" {
		t.Errorf("expected URL 'https://example.com', got %v", data.URL)
	}
}

func TestNewTabClosedEvent(t *testing.T) {
	event := NewTabClosedEvent("example.com", "tab-1", "session-123", "target-456", 123.45)

	if event.EventType != EventMetaTabClosed {
		t.Errorf("expected EventType %s, got %s", EventMetaTabClosed, event.EventType)
	}

	data, ok := event.Data.(*TabClosedData)
	if !ok {
		t.Fatalf("expected Data to be *TabClosedData, got %T", event.Data)
	}
	if data.DurationSeconds != 123.45 {
		t.Errorf("expected DurationSeconds 123.45, got %v", data.DurationSeconds)
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

	data, ok := event.Data.(*SiteChangedData)
	if !ok {
		t.Fatalf("expected Data to be *SiteChangedData, got %T", event.Data)
	}
	if data.OldSite != "old.com" {
		t.Errorf("expected OldSite 'old.com', got %v", data.OldSite)
	}
	if data.NewSite != "new.com" {
		t.Errorf("expected NewSite 'new.com', got %v", data.NewSite)
	}
	if data.NewURL != "https://new.com/page" {
		t.Errorf("expected NewURL 'https://new.com/page', got %v", data.NewURL)
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

	data, ok := event.Data.(*SiteEnteredData)
	if !ok {
		t.Fatalf("expected Data to be *SiteEnteredData, got %T", event.Data)
	}
	if data.FromSite != "old.com" {
		t.Errorf("expected FromSite 'old.com', got %v", data.FromSite)
	}
}

func TestNewPageNavigateEvent(t *testing.T) {
	event := NewPageNavigateEvent("example.com", "tab-1", "https://example.com/page", "https://example.com", "link")

	if event.EventType != EventPageNavigate {
		t.Errorf("expected EventType %s, got %s", EventPageNavigate, event.EventType)
	}

	data, ok := event.Data.(*PageNavigateData)
	if !ok {
		t.Fatalf("expected Data to be *PageNavigateData, got %T", event.Data)
	}
	if data.URL != "https://example.com/page" {
		t.Errorf("expected URL 'https://example.com/page', got %v", data.URL)
	}
	if data.Referrer != "https://example.com" {
		t.Errorf("expected Referrer 'https://example.com', got %v", data.Referrer)
	}
	if data.NavigationType != "link" {
		t.Errorf("expected NavigationType 'link', got %v", data.NavigationType)
	}
}

func TestNewPageLoadEvent(t *testing.T) {
	event := NewPageLoadEvent("example.com", "tab-1", "https://example.com")

	if event.EventType != EventPageLoad {
		t.Errorf("expected EventType %s, got %s", EventPageLoad, event.EventType)
	}

	data, ok := event.Data.(*PageLoadData)
	if !ok {
		t.Fatalf("expected Data to be *PageLoadData, got %T", event.Data)
	}
	if data.URL != "https://example.com" {
		t.Errorf("expected URL 'https://example.com', got %v", data.URL)
	}
}

func TestNewPageDOMReadyEvent(t *testing.T) {
	event := NewPageDOMReadyEvent("example.com", "tab-1", "https://example.com")

	if event.EventType != EventPageDOMReady {
		t.Errorf("expected EventType %s, got %s", EventPageDOMReady, event.EventType)
	}

	data, ok := event.Data.(*PageDOMReadyData)
	if !ok {
		t.Fatalf("expected Data to be *PageDOMReadyData, got %T", event.Data)
	}
	if data.URL != "https://example.com" {
		t.Errorf("expected URL 'https://example.com', got %v", data.URL)
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
