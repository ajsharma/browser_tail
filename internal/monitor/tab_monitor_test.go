package monitor

import (
	"context"
	"encoding/json"
	"sync"
	"testing"

	"github.com/chromedp/cdproto/runtime"

	"github.com/ajsharma/browser_tail/internal/config"
	"github.com/ajsharma/browser_tail/internal/events"
)

// fakeEventWriter records all events written to it.
type fakeEventWriter struct {
	mu     sync.Mutex
	events []*events.LogEvent
	closed map[string]string // tabID:site pairs that were closed
}

func newFakeEventWriter() *fakeEventWriter {
	return &fakeEventWriter{
		closed: make(map[string]string),
	}
}

func (f *fakeEventWriter) WriteEvent(tabID string, event *events.LogEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, event)
	return nil
}

func (f *fakeEventWriter) CloseTab(tabID, site string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closed[tabID] = site
	return nil
}

func (f *fakeEventWriter) getEvents() []*events.LogEvent {
	f.mu.Lock()
	defer f.mu.Unlock()
	cp := make([]*events.LogEvent, len(f.events))
	copy(cp, f.events)
	return cp
}

func newTestMonitor(fw *fakeEventWriter, cfg *config.Config) *TabMonitor {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	return NewTabMonitor(
		context.Background(),
		"target-abc123",
		"tab-1",
		"example.com",
		"Example",
		"https://example.com",
		"session-123",
		fw,
		cfg,
	)
}

func TestHandleSiteChange(t *testing.T) {
	fw := newFakeEventWriter()
	tm := newTestMonitor(fw, nil)

	changed := tm.HandleSiteChange("github.com", "https://github.com")
	if !changed {
		t.Fatal("expected HandleSiteChange to return true")
	}

	evts := fw.getEvents()
	if len(evts) != 2 {
		t.Fatalf("expected 2 events (site_changed + site_entered), got %d", len(evts))
	}

	if evts[0].EventType != events.EventMetaSiteChanged {
		t.Errorf("first event type = %q, want %q", evts[0].EventType, events.EventMetaSiteChanged)
	}
	if evts[0].Site != "example.com" {
		t.Errorf("site_changed event site = %q, want example.com", evts[0].Site)
	}

	if evts[1].EventType != events.EventMetaSiteEntered {
		t.Errorf("second event type = %q, want %q", evts[1].EventType, events.EventMetaSiteEntered)
	}
	if evts[1].Site != "github.com" {
		t.Errorf("site_entered event site = %q, want github.com", evts[1].Site)
	}

	// Verify old site log was closed
	if fw.closed["tab-1"] != "example.com" {
		t.Errorf("expected CloseTab for example.com, got %q", fw.closed["tab-1"])
	}

	// Verify current state updated
	if tm.CurrentSite() != "github.com" {
		t.Errorf("CurrentSite() = %q, want github.com", tm.CurrentSite())
	}
}

func TestHandleSiteChange_SameSite(t *testing.T) {
	fw := newFakeEventWriter()
	tm := newTestMonitor(fw, nil)

	changed := tm.HandleSiteChange("example.com", "https://example.com/other")
	if changed {
		t.Fatal("expected HandleSiteChange to return false for same site")
	}

	evts := fw.getEvents()
	if len(evts) != 0 {
		t.Fatalf("expected 0 events for same-site navigation, got %d", len(evts))
	}
}

func TestStop(t *testing.T) {
	fw := newFakeEventWriter()
	tm := newTestMonitor(fw, nil)

	tm.Stop()

	evts := fw.getEvents()
	if len(evts) != 1 {
		t.Fatalf("expected 1 event (tab_closed), got %d", len(evts))
	}

	if evts[0].EventType != events.EventMetaTabClosed {
		t.Errorf("event type = %q, want %q", evts[0].EventType, events.EventMetaTabClosed)
	}
}

func TestShouldCaptureBody(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.CaptureBodies = true
	cfg.BodySizeLimitKB = 10
	cfg.BodyContentTypes = []string{"text/*", "application/json"}

	fw := newFakeEventWriter()
	tm := newTestMonitor(fw, cfg)

	tests := []struct {
		name     string
		mimeType string
		size     float64
		want     bool
	}{
		{"text/html within limit", "text/html", 1000, true},
		{"application/json within limit", "application/json", 5000, true},
		{"text/plain within limit", "text/plain; charset=utf-8", 100, true},
		{"image/png not allowed", "image/png", 1000, false},
		{"text/html over limit", "text/html", 20000, false},
		{"zero size allowed", "text/html", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tm.shouldCaptureBody(tt.mimeType, tt.size)
			if got != tt.want {
				t.Errorf("shouldCaptureBody(%q, %v) = %v, want %v", tt.mimeType, tt.size, got, tt.want)
			}
		})
	}
}

func TestMatchContentType(t *testing.T) {
	tests := []struct {
		actual  string
		pattern string
		want    bool
	}{
		{"text/html", "text/*", true},
		{"text/plain", "text/*", true},
		{"application/json", "text/*", false},
		{"application/json", "application/json", true},
		{"text/html; charset=utf-8", "text/html", true},
		{"text/html; charset=utf-8", "text/*", true},
		{"image/png", "text/*", false},
	}

	for _, tt := range tests {
		t.Run(tt.actual+"_vs_"+tt.pattern, func(t *testing.T) {
			got := matchContentType(tt.actual, tt.pattern)
			if got != tt.want {
				t.Errorf("matchContentType(%q, %q) = %v, want %v", tt.actual, tt.pattern, got, tt.want)
			}
		})
	}
}

func TestExtractRemoteObjectValue(t *testing.T) {
	t.Run("nil object", func(t *testing.T) {
		if got := extractRemoteObjectValue(nil); got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})

	t.Run("undefined", func(t *testing.T) {
		obj := &runtime.RemoteObject{Type: runtime.TypeUndefined}
		if got := extractRemoteObjectValue(obj); got != "undefined" {
			t.Errorf("expected 'undefined', got %v", got)
		}
	})

	t.Run("null subtype", func(t *testing.T) {
		obj := &runtime.RemoteObject{Type: runtime.TypeObject, Subtype: runtime.SubtypeNull}
		if got := extractRemoteObjectValue(obj); got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})

	t.Run("string value", func(t *testing.T) {
		val, _ := json.Marshal("hello")
		obj := &runtime.RemoteObject{Type: runtime.TypeString, Value: val}
		if got := extractRemoteObjectValue(obj); got != "hello" {
			t.Errorf("expected 'hello', got %v", got)
		}
	})

	t.Run("number value", func(t *testing.T) {
		val, _ := json.Marshal(42.0)
		obj := &runtime.RemoteObject{Type: runtime.TypeNumber, Value: val}
		got, ok := extractRemoteObjectValue(obj).(float64)
		if !ok || got != 42.0 {
			t.Errorf("expected 42.0, got %v", got)
		}
	})

	t.Run("boolean value", func(t *testing.T) {
		val, _ := json.Marshal(true)
		obj := &runtime.RemoteObject{Type: runtime.TypeBoolean, Value: val}
		got, ok := extractRemoteObjectValue(obj).(bool)
		if !ok || !got {
			t.Errorf("expected true, got %v", got)
		}
	})

	t.Run("unserializable NaN", func(t *testing.T) {
		obj := &runtime.RemoteObject{UnserializableValue: "NaN"}
		if got := extractRemoteObjectValue(obj); got != "NaN" {
			t.Errorf("expected 'NaN', got %v", got)
		}
	})

	t.Run("description fallback", func(t *testing.T) {
		obj := &runtime.RemoteObject{Type: runtime.TypeObject, Description: "Array(3)"}
		if got := extractRemoteObjectValue(obj); got != "Array(3)" {
			t.Errorf("expected 'Array(3)', got %v", got)
		}
	})

	t.Run("type fallback", func(t *testing.T) {
		obj := &runtime.RemoteObject{Type: runtime.TypeFunction}
		if got := extractRemoteObjectValue(obj); got != "function" {
			t.Errorf("expected 'function', got %v", got)
		}
	})
}

func TestExtractPropertyValue(t *testing.T) {
	tests := []struct {
		name string
		prop *runtime.PropertyPreview
		want interface{}
	}{
		{"undefined", &runtime.PropertyPreview{Value: "undefined"}, "undefined"},
		{"null", &runtime.PropertyPreview{Value: "null"}, nil},
		{"number", &runtime.PropertyPreview{Type: runtime.TypeNumber, Value: "42"}, float64(42)},
		{"boolean true", &runtime.PropertyPreview{Type: runtime.TypeBoolean, Value: "true"}, true},
		{"boolean false", &runtime.PropertyPreview{Type: runtime.TypeBoolean, Value: "false"}, false},
		{"string", &runtime.PropertyPreview{Type: runtime.TypeString, Value: "hello"}, "hello"},
		{"null object", &runtime.PropertyPreview{Type: runtime.TypeObject, Subtype: runtime.SubtypeNull, Value: "null"}, nil},
		{"object ref", &runtime.PropertyPreview{Type: runtime.TypeObject, Value: "Object"}, "Object"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractPropertyValue(tt.prop)
			if got != tt.want {
				t.Errorf("extractPropertyValue() = %v (%T), want %v (%T)", got, got, tt.want, tt.want)
			}
		})
	}
}

func TestExtractObjectPreview(t *testing.T) {
	t.Run("nil preview", func(t *testing.T) {
		if got := extractObjectPreview(nil); got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})

	t.Run("array preview", func(t *testing.T) {
		preview := &runtime.ObjectPreview{
			Subtype: runtime.SubtypeArray,
			Properties: []*runtime.PropertyPreview{
				{Type: runtime.TypeNumber, Value: "1"},
				{Type: runtime.TypeNumber, Value: "2"},
			},
		}
		got := extractObjectPreview(preview)
		arr, ok := got.([]interface{})
		if !ok {
			t.Fatalf("expected []interface{}, got %T", got)
		}
		if len(arr) != 2 {
			t.Errorf("expected 2 elements, got %d", len(arr))
		}
	})

	t.Run("object preview", func(t *testing.T) {
		preview := &runtime.ObjectPreview{
			Properties: []*runtime.PropertyPreview{
				{Name: "key", Type: runtime.TypeString, Value: "value"},
			},
		}
		got := extractObjectPreview(preview)
		obj, ok := got.(map[string]interface{})
		if !ok {
			t.Fatalf("expected map[string]interface{}, got %T", got)
		}
		if obj["key"] != "value" {
			t.Errorf("expected key='value', got %v", obj["key"])
		}
	})

	t.Run("overflow array", func(t *testing.T) {
		preview := &runtime.ObjectPreview{
			Subtype:  runtime.SubtypeArray,
			Overflow: true,
			Properties: []*runtime.PropertyPreview{
				{Type: runtime.TypeNumber, Value: "1"},
			},
		}
		got := extractObjectPreview(preview)
		arr := got.([]interface{})
		if len(arr) != 2 || arr[1] != "..." {
			t.Errorf("expected overflow marker, got %v", arr)
		}
	})
}
