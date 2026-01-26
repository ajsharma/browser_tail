package logger

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ajsharma/browser_tail/internal/events"
)

func TestNewFileManager(t *testing.T) {
	fm := NewFileManager("/test/dir")

	if fm.baseDir != "/test/dir" {
		t.Errorf("baseDir = %q, want %q", fm.baseDir, "/test/dir")
	}
	if fm.flushInterval != DefaultFlushInterval {
		t.Errorf("flushInterval = %v, want %v", fm.flushInterval, DefaultFlushInterval)
	}
	if fm.bufferSize != DefaultBufferSize {
		t.Errorf("bufferSize = %d, want %d", fm.bufferSize, DefaultBufferSize)
	}
	if fm.files == nil {
		t.Error("files map is nil")
	}
}

func TestFileManagerSetFlushInterval(t *testing.T) {
	fm := NewFileManager("/test")
	fm.SetFlushInterval(200 * time.Millisecond)

	if fm.flushInterval != 200*time.Millisecond {
		t.Errorf("flushInterval = %v, want %v", fm.flushInterval, 200*time.Millisecond)
	}
}

func TestFileManagerSetBufferSize(t *testing.T) {
	fm := NewFileManager("/test")
	fm.SetBufferSize(16384)

	if fm.bufferSize != 16384 {
		t.Errorf("bufferSize = %d, want %d", fm.bufferSize, 16384)
	}
}

func TestFileManagerWriteAndClose(t *testing.T) {
	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "file_manager_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	fm := NewFileManager(tmpDir)

	// Write an event
	event := events.NewLogEvent("example.com", "tab-1", "page.navigate", map[string]interface{}{
		"url": "https://example.com",
	})

	if err := fm.WriteEvent("tab-1", event); err != nil {
		t.Fatalf("WriteEvent failed: %v", err)
	}

	// Verify file was created
	expectedPath := filepath.Join(tmpDir, "example.com", "tab-1", "session.log")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("Log file was not created at %s", expectedPath)
	}

	// Close the file manager
	if err := fm.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Read and verify the file contents
	content, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	// Parse the JSON line
	var readEvent events.LogEvent
	if err := json.Unmarshal(content, &readEvent); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if readEvent.Site != "example.com" {
		t.Errorf("Site = %q, want %q", readEvent.Site, "example.com")
	}
	if readEvent.TabID != "tab-1" {
		t.Errorf("TabID = %q, want %q", readEvent.TabID, "tab-1")
	}
	if readEvent.EventType != "page.navigate" {
		t.Errorf("EventType = %q, want %q", readEvent.EventType, "page.navigate")
	}
}

func TestFileManagerMultipleEvents(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "file_manager_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	fm := NewFileManager(tmpDir)

	// Write multiple events to the same tab
	for i := 0; i < 5; i++ {
		event := events.NewLogEvent("example.com", "tab-1", "page.navigate", map[string]interface{}{
			"index": i,
		})
		if err := fm.WriteEvent("tab-1", event); err != nil {
			t.Fatalf("WriteEvent failed: %v", err)
		}
	}

	if err := fm.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Read and verify
	content, err := os.ReadFile(filepath.Join(tmpDir, "example.com", "tab-1", "session.log"))
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) != 5 {
		t.Errorf("Expected 5 lines, got %d", len(lines))
	}

	// Verify each line is valid JSON
	for i, line := range lines {
		var event events.LogEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Errorf("Line %d is not valid JSON: %v", i, err)
		}
	}
}

func TestFileManagerMultipleSites(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "file_manager_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	fm := NewFileManager(tmpDir)

	// Write events to different sites
	sites := []string{"example.com", "github.com", "google.com"}
	for _, site := range sites {
		event := events.NewLogEvent(site, "tab-1", "page.navigate", map[string]interface{}{
			"url": "https://" + site,
		})
		if err := fm.WriteEvent("tab-1", event); err != nil {
			t.Fatalf("WriteEvent failed: %v", err)
		}
	}

	// Verify GetOpenFiles
	if fm.GetOpenFiles() != 3 {
		t.Errorf("GetOpenFiles() = %d, want 3", fm.GetOpenFiles())
	}

	if err := fm.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Verify files were created for each site
	for _, site := range sites {
		path := filepath.Join(tmpDir, site, "tab-1", "session.log")
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Log file was not created for site %s at %s", site, path)
		}
	}
}

func TestFileManagerCloseTab(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "file_manager_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	fm := NewFileManager(tmpDir)

	// Write an event
	event := events.NewLogEvent("example.com", "tab-1", "page.navigate", map[string]interface{}{
		"url": "https://example.com",
	})
	if err := fm.WriteEvent("tab-1", event); err != nil {
		t.Fatalf("WriteEvent failed: %v", err)
	}

	if fm.GetOpenFiles() != 1 {
		t.Errorf("GetOpenFiles() = %d, want 1", fm.GetOpenFiles())
	}

	// Close the specific tab
	if err := fm.CloseTab("tab-1", "example.com"); err != nil {
		t.Fatalf("CloseTab failed: %v", err)
	}

	if fm.GetOpenFiles() != 0 {
		t.Errorf("After CloseTab, GetOpenFiles() = %d, want 0", fm.GetOpenFiles())
	}

	// Verify file exists and has content
	path := filepath.Join(tmpDir, "example.com", "tab-1", "session.log")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}
	if len(content) == 0 {
		t.Error("Log file is empty after CloseTab")
	}
}

func TestFileManagerCloseAllForTab(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "file_manager_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	fm := NewFileManager(tmpDir)

	// Write events to multiple sites for the same tab
	sites := []string{"example.com", "github.com"}
	for _, site := range sites {
		event := events.NewLogEvent(site, "tab-1", "page.navigate", nil)
		if err := fm.WriteEvent("tab-1", event); err != nil {
			t.Fatalf("WriteEvent failed: %v", err)
		}
	}

	// Write to a different tab
	event := events.NewLogEvent("google.com", "tab-2", "page.navigate", nil)
	if err := fm.WriteEvent("tab-2", event); err != nil {
		t.Fatalf("WriteEvent failed: %v", err)
	}

	if fm.GetOpenFiles() != 3 {
		t.Errorf("GetOpenFiles() = %d, want 3", fm.GetOpenFiles())
	}

	// Close all files for tab-1
	if err := fm.CloseAllForTab("tab-1"); err != nil {
		t.Fatalf("CloseAllForTab failed: %v", err)
	}

	// Only tab-2 should remain
	if fm.GetOpenFiles() != 1 {
		t.Errorf("After CloseAllForTab, GetOpenFiles() = %d, want 1", fm.GetOpenFiles())
	}

	// Clean up
	if err := fm.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

func TestFileManagerMetaEventFlush(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "file_manager_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	fm := NewFileManager(tmpDir)

	// Write a meta event - should be flushed immediately
	event := events.NewLogEvent("example.com", "tab-1", "meta.tab_created", map[string]interface{}{
		"target_id": "abc123",
	})
	if err := fm.WriteEvent("tab-1", event); err != nil {
		t.Fatalf("WriteEvent failed: %v", err)
	}

	// Read the file immediately - meta events should be synced
	path := filepath.Join(tmpDir, "example.com", "tab-1", "session.log")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if len(content) == 0 {
		t.Error("Meta event should be flushed immediately but file is empty")
	}

	// Parse and verify the event
	var readEvent events.LogEvent
	if err := json.Unmarshal(content, &readEvent); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if readEvent.EventType != "meta.tab_created" {
		t.Errorf("EventType = %q, want %q", readEvent.EventType, "meta.tab_created")
	}

	if err := fm.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

func TestFileManagerCloseNonExistent(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "file_manager_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	fm := NewFileManager(tmpDir)

	// Close a tab that doesn't exist - should not error
	if err := fm.CloseTab("non-existent", "example.com"); err != nil {
		t.Errorf("CloseTab for non-existent tab should not error: %v", err)
	}

	if err := fm.CloseAllForTab("non-existent"); err != nil {
		t.Errorf("CloseAllForTab for non-existent tab should not error: %v", err)
	}
}

func TestFileKey(t *testing.T) {
	tests := []struct {
		tabID    string
		site     string
		expected string
	}{
		{"tab-1", "example.com", "tab-1:example.com"},
		{"tab-42", "localhost_3000", "tab-42:localhost_3000"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := fileKey(tt.tabID, tt.site)
			if result != tt.expected {
				t.Errorf("fileKey(%q, %q) = %q, want %q",
					tt.tabID, tt.site, result, tt.expected)
			}
		})
	}
}
