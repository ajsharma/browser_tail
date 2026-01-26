package logger

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ajsharma/browser_tail/internal/events"
)

const (
	// DefaultBufferSize is the default buffer size for log writers (8 KB).
	DefaultBufferSize = 8 * 1024

	// DefaultFlushInterval is the default interval between automatic flushes.
	DefaultFlushInterval = 100 * time.Millisecond
)

// tabWriter manages a single log file for a tab.
type tabWriter struct {
	file       *os.File
	writer     *bufio.Writer
	flushTimer *time.Timer
	mu         sync.Mutex
	site       string
	tabID      string
}

// FileManager manages log files for all tabs.
type FileManager struct {
	baseDir       string
	files         map[string]*tabWriter // key: tabID + ":" + site
	mu            sync.RWMutex
	flushInterval time.Duration
	bufferSize    int
}

// NewFileManager creates a new FileManager with the specified base directory.
func NewFileManager(baseDir string) *FileManager {
	return &FileManager{
		baseDir:       baseDir,
		files:         make(map[string]*tabWriter),
		flushInterval: DefaultFlushInterval,
		bufferSize:    DefaultBufferSize,
	}
}

// SetFlushInterval sets the flush interval for automatic flushing.
func (fm *FileManager) SetFlushInterval(interval time.Duration) {
	fm.flushInterval = interval
}

// SetBufferSize sets the buffer size for new writers.
func (fm *FileManager) SetBufferSize(size int) {
	fm.bufferSize = size
}

// fileKey returns the key used to identify a file in the files map.
func fileKey(tabID, site string) string {
	return tabID + ":" + site
}

// getWriter returns the writer for the given tab and site, creating it if necessary.
func (fm *FileManager) getWriter(tabID, site string) (*tabWriter, error) {
	key := fileKey(tabID, site)

	fm.mu.RLock()
	if tw, exists := fm.files[key]; exists {
		fm.mu.RUnlock()
		return tw, nil
	}
	fm.mu.RUnlock()

	fm.mu.Lock()
	defer fm.mu.Unlock()

	// Double-check after acquiring write lock
	if tw, exists := fm.files[key]; exists {
		return tw, nil
	}

	// Create log file
	path := GetLogPath(fm.baseDir, site, tabID)
	dir := filepath.Dir(path)

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}

	tw := &tabWriter{
		file:   f,
		writer: bufio.NewWriterSize(f, fm.bufferSize),
		site:   site,
		tabID:  tabID,
	}

	fm.files[key] = tw
	return tw, nil
}

// WriteEvent writes a log event to the appropriate file.
func (fm *FileManager) WriteEvent(tabID string, event *events.LogEvent) error {
	tw, err := fm.getWriter(tabID, event.Site)
	if err != nil {
		return err
	}

	tw.mu.Lock()
	defer tw.mu.Unlock()

	// Marshal to JSON
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	// Write with newline
	if _, err := tw.writer.Write(append(data, '\n')); err != nil {
		return err
	}

	// Smart flush strategy based on event type and buffer state
	return fm.handleFlush(tw, event.EventType)
}

// handleFlush determines and executes the appropriate flush strategy.
func (fm *FileManager) handleFlush(tw *tabWriter, eventType string) error {
	isMeta := strings.HasPrefix(eventType, "meta.")
	bufferFull := tw.writer.Buffered() > tw.writer.Size()*3/4

	switch {
	case isMeta:
		// Meta events MUST be synced immediately (tab lifecycle critical)
		if err := tw.writer.Flush(); err != nil {
			return err
		}
		if err := tw.file.Sync(); err != nil {
			return err
		}
		tw.cancelFlushTimer()
	case bufferFull:
		// Buffer nearly full, flush to OS (but don't sync to disk)
		if err := tw.writer.Flush(); err != nil {
			return err
		}
		tw.cancelFlushTimer()
	default:
		// Schedule deferred flush
		tw.scheduleFlush(fm.flushInterval)
	}

	return nil
}

// scheduleFlush schedules a flush after the given interval.
func (tw *tabWriter) scheduleFlush(interval time.Duration) {
	if tw.flushTimer != nil {
		return // Timer already scheduled
	}

	tw.flushTimer = time.AfterFunc(interval, func() {
		tw.mu.Lock()
		defer tw.mu.Unlock()
		_ = tw.writer.Flush()
		tw.flushTimer = nil
	})
}

// cancelFlushTimer cancels any pending flush timer.
func (tw *tabWriter) cancelFlushTimer() {
	if tw.flushTimer != nil {
		tw.flushTimer.Stop()
		tw.flushTimer = nil
	}
}

// CloseTab closes the log file for a specific tab and site.
func (fm *FileManager) CloseTab(tabID, site string) error {
	key := fileKey(tabID, site)

	fm.mu.Lock()
	tw, exists := fm.files[key]
	if !exists {
		fm.mu.Unlock()
		return nil
	}
	delete(fm.files, key)
	fm.mu.Unlock()

	tw.mu.Lock()
	defer tw.mu.Unlock()

	// Final flush and sync
	tw.cancelFlushTimer()

	if err := tw.writer.Flush(); err != nil {
		return err
	}
	if err := tw.file.Sync(); err != nil {
		return err
	}

	return tw.file.Close()
}

// CloseAllForTab closes all log files for a specific tab (all sites).
func (fm *FileManager) CloseAllForTab(tabID string) error {
	fm.mu.Lock()
	var toClose []*tabWriter
	var keysToDelete []string

	for key, tw := range fm.files {
		if tw.tabID == tabID {
			toClose = append(toClose, tw)
			keysToDelete = append(keysToDelete, key)
		}
	}

	for _, key := range keysToDelete {
		delete(fm.files, key)
	}
	fm.mu.Unlock()

	var lastErr error
	for _, tw := range toClose {
		tw.mu.Lock()
		tw.cancelFlushTimer()

		if err := tw.writer.Flush(); err != nil {
			lastErr = err
		}
		if err := tw.file.Sync(); err != nil {
			lastErr = err
		}
		if err := tw.file.Close(); err != nil {
			lastErr = err
		}
		tw.mu.Unlock()
	}

	return lastErr
}

// Close closes all open log files.
func (fm *FileManager) Close() error {
	fm.mu.Lock()
	writers := make([]*tabWriter, 0, len(fm.files))
	for _, tw := range fm.files {
		writers = append(writers, tw)
	}
	fm.files = make(map[string]*tabWriter)
	fm.mu.Unlock()

	var lastErr error
	for _, tw := range writers {
		tw.mu.Lock()
		tw.cancelFlushTimer()

		if err := tw.writer.Flush(); err != nil {
			lastErr = err
		}
		if err := tw.file.Sync(); err != nil {
			lastErr = err
		}
		if err := tw.file.Close(); err != nil {
			lastErr = err
		}
		tw.mu.Unlock()
	}

	return lastErr
}

// GetOpenFiles returns the number of currently open log files.
func (fm *FileManager) GetOpenFiles() int {
	fm.mu.RLock()
	defer fm.mu.RUnlock()
	return len(fm.files)
}
