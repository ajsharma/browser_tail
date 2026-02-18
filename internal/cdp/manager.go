package cdp

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"

	"github.com/ajsharma/browser_tail/internal/config"
	"github.com/ajsharma/browser_tail/internal/events"
	"github.com/ajsharma/browser_tail/internal/logger"
	"github.com/ajsharma/browser_tail/internal/monitor"
)

// Manager orchestrates CDP connections and tab monitoring.
type Manager struct {
	config           *config.Config
	fileManager      *logger.FileManager
	tabRegistry      *logger.TabRegistry
	chromeProcess    *ChromeProcess
	tabMonitors      map[string]*monitor.TabMonitor // targetID -> monitor
	mu               sync.RWMutex
	allocatorCtx     context.Context
	allocatorCancel  context.CancelFunc
	browserCtx       context.Context
	browserCancel    context.CancelFunc
	internalTargetID string // our internal anchor tab
	connected        bool
}

// Connection retry settings.
const (
	reconnectInterval = 5 * time.Second
	maxReconnectWait  = 30 * time.Second
)

// NewManager creates a new CDP Manager.
func NewManager(cfg *config.Config, fm *logger.FileManager) *Manager {
	return &Manager{
		config:      cfg,
		fileManager: fm,
		tabRegistry: logger.NewTabRegistry(),
		tabMonitors: make(map[string]*monitor.TabMonitor),
	}
}

// Start begins monitoring Chrome with automatic reconnection.
func (m *Manager) Start(ctx context.Context) error {
	// Auto-launch Chrome if requested
	if m.config.AutoLaunch {
		var err error
		m.chromeProcess, err = LaunchChrome(m.config.ChromePort)
		if err != nil {
			return fmt.Errorf("failed to launch chrome: %w", err)
		}

		// Wait for Chrome to be ready
		if err := WaitForChrome(m.config.ChromePort, 30*time.Second); err != nil {
			if stopErr := m.chromeProcess.Stop(); stopErr != nil {
				log.Printf("Warning: failed to stop Chrome during cleanup: %v", stopErr)
			}
			return fmt.Errorf("chrome not ready: %w", err)
		}

		slog.Info("Launched Chrome", "pid", m.chromeProcess.PID(), "port", m.config.ChromePort)
	}

	// Log session start
	chromePID := 0
	if m.chromeProcess != nil {
		chromePID = m.chromeProcess.PID()
	}
	sessionEvent := events.NewSessionStartEvent(
		m.tabRegistry.GetSessionID(),
		chromePID,
		config.Version,
	)
	if err := m.fileManager.WriteEvent("_session", sessionEvent); err != nil {
		slog.Warn("Failed to write session start event", "error", err)
	}

	// Reconnection loop
	retryWait := reconnectInterval
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		err := m.connect(ctx)
		if err == nil {
			// Connection successful, reset retry wait
			retryWait = reconnectInterval
			m.connected = true

			// Wait for disconnection or context cancellation
			select {
			case <-ctx.Done():
				return nil
			case <-m.browserCtx.Done():
				// Browser disconnected, will retry
				m.connected = false
				slog.Warn("Chrome disconnected, will retry", "wait", retryWait)
			}
		} else {
			slog.Error("Failed to connect to Chrome", "error", err)
		}

		// Wait before retrying (with backoff)
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(retryWait):
			// Increase retry wait with backoff, up to max
			retryWait = retryWait * 2
			if retryWait > maxReconnectWait {
				retryWait = maxReconnectWait
			}
		}
	}
}

// connect establishes a connection to Chrome and starts monitoring.
func (m *Manager) connect(ctx context.Context) error {
	// Step 1: Initial discovery via /json (ONE-TIME)
	initialTabs, err := DiscoverTabs(m.config.ChromePort)
	if err != nil {
		return fmt.Errorf("failed to discover initial tabs: %w", err)
	}

	slog.Info("Discovered existing tabs", "count", len(initialTabs))

	// Get browser info for WebSocket URL
	browserInfo, err := DiscoverBrowserInfo(m.config.ChromePort)
	if err != nil {
		return fmt.Errorf("failed to get browser info: %w", err)
	}

	// Step 2: Connect to browser-level CDP
	// Keep allocator context alive - it represents the browser connection
	m.allocatorCtx, m.allocatorCancel = chromedp.NewRemoteAllocator(ctx, browserInfo.WebSocketDebuggerURL)

	// Create browser context
	m.browserCtx, m.browserCancel = chromedp.NewContext(m.allocatorCtx)

	// Step 3: Enable target discovery (this also initializes the browser connection)
	// IMPORTANT: Must call Run to initialize c.Browser before ListenBrowser will work
	if err := chromedp.Run(m.browserCtx, target.SetDiscoverTargets(true)); err != nil {
		m.browserCancel()
		m.allocatorCancel()
		return fmt.Errorf("failed to enable target discovery: %w", err)
	}

	// Track our internal target ID so we don't monitor it as a user tab
	c := chromedp.FromContext(m.browserCtx)
	if c.Target != nil {
		m.internalTargetID = string(c.Target.TargetID)
		slog.Debug("Using internal anchor tab", "target_id", m.internalTargetID[:8])
	}

	// Navigate anchor tab to test page instead of leaving it as about:blank
	testPageURL := "data:text/html;base64," + base64.StdEncoding.EncodeToString([]byte(TestPageHTML))
	if err := chromedp.Run(m.browserCtx, chromedp.Navigate(testPageURL)); err != nil {
		slog.Warn("Could not load test page", "error", err)
		// Non-fatal - about:blank still works
	}

	// Clean up any orphaned about:blank tabs from previous runs
	// Keep only our internal anchor tab
	m.cleanupOrphanedAnchorTabs(initialTabs)

	// Step 4: Set up listener for target events
	// ListenTarget receives target domain events (targetCreated, targetDestroyed, etc.)
	// This listener is tied to our internal anchor tab - survives user tab closures
	chromedp.ListenTarget(m.browserCtx, func(ev interface{}) {
		switch ev := ev.(type) {
		case *target.EventTargetCreated:
			// Only monitor page targets, skip internal/blank/test page tabs
			if ev.TargetInfo.Type == TargetTypePage &&
				!isInternalURL(ev.TargetInfo.URL) &&
				string(ev.TargetInfo.TargetID) != m.internalTargetID {
				m.handleNewTarget(ctx, ev.TargetInfo)
			}

		case *target.EventTargetDestroyed:
			// Skip destruction of our internal anchor
			if string(ev.TargetID) != m.internalTargetID {
				m.handleTargetDestroyed(string(ev.TargetID))
			}

		case *target.EventTargetInfoChanged:
			// Tab URL/title changed, skip internal/blank/test page tabs
			if ev.TargetInfo.Type == TargetTypePage &&
				!isInternalURL(ev.TargetInfo.URL) &&
				string(ev.TargetInfo.TargetID) != m.internalTargetID {
				m.handleTargetInfoChanged(ev.TargetInfo)
			}
		}
	})

	// Clean up on disconnect
	go func() {
		<-m.browserCtx.Done()
		// Close our internal anchor tab to avoid leaving orphaned about:blank tabs
		if m.internalTargetID != "" {
			// Try to close the anchor tab (best effort, may fail if already closed)
			_ = target.CloseTarget(target.ID(m.internalTargetID)).Do(m.allocatorCtx)
		}
		m.allocatorCancel()
		m.clearTabMonitors()
	}()

	// Start monitoring existing tabs (filter out internal/blank/test page tabs)
	for _, tab := range initialTabs {
		if tab.TargetID != m.internalTargetID && !isInternalURL(tab.URL) {
			m.handleNewTarget(ctx, &target.Info{
				TargetID: target.ID(tab.TargetID),
				Type:     tab.Type,
				Title:    tab.Title,
				URL:      tab.URL,
				Attached: false,
			})
		}
	}

	slog.Info("Monitoring started", "session", m.tabRegistry.GetSessionID())

	return nil
}

// clearTabMonitors stops and removes all tab monitors.
func (m *Manager) clearTabMonitors() {
	m.mu.Lock()
	monitors := make([]*monitor.TabMonitor, 0, len(m.tabMonitors))
	for _, mon := range m.tabMonitors {
		monitors = append(monitors, mon)
	}
	m.tabMonitors = make(map[string]*monitor.TabMonitor)
	m.mu.Unlock()

	for _, mon := range monitors {
		mon.Stop()
	}
}

// cleanupOrphanedAnchorTabs closes any internal tabs (about:blank, test pages) that aren't our internal anchor.
// This cleans up tabs left over from previous browser_tail runs.
func (m *Manager) cleanupOrphanedAnchorTabs(tabs []*Tab) {
	for _, tab := range tabs {
		// Close internal tabs (about:blank, data: URLs) that aren't our internal anchor
		if isInternalURL(tab.URL) && tab.TargetID != m.internalTargetID {
			slog.Debug("Closing orphaned internal tab", "target_id", tab.TargetID[:8])
			// Use HTTP API to close the tab (CDP CloseTarget requires owning the context)
			if err := closeTabViaHTTP(m.config.ChromePort, tab.TargetID); err != nil {
				slog.Warn("Failed to close orphaned tab", "error", err)
			}
		}
	}
}

// handleNewTarget starts monitoring a new tab.
func (m *Manager) handleNewTarget(ctx context.Context, info *target.Info) {
	targetID := string(info.TargetID)

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if already monitoring
	if _, exists := m.tabMonitors[targetID]; exists {
		return
	}

	// Get or create tab ID
	tabID := m.tabRegistry.GetOrCreateTabID(targetID)
	site := logger.ExtractSite(info.URL)

	// Create tab monitor
	mon := monitor.NewTabMonitor(
		ctx,
		targetID,
		tabID,
		site,
		info.Title,
		info.URL,
		m.tabRegistry.GetSessionID(),
		m.fileManager,
		m.config,
	)

	m.tabMonitors[targetID] = mon

	// Start monitoring in goroutine
	go func() {
		if err := mon.Start(m.browserCtx); err != nil {
			slog.Error("Tab monitor error", "tab", tabID, "error", err)
		}
	}()

	slog.Info("Started monitoring tab", "tab", tabID, "target_id", targetID[:8], "url", info.URL)
}

// handleTargetDestroyed handles a tab being closed.
func (m *Manager) handleTargetDestroyed(targetID string) {
	m.mu.Lock()
	mon, exists := m.tabMonitors[targetID]
	if !exists {
		m.mu.Unlock()
		return
	}

	delete(m.tabMonitors, targetID)
	m.mu.Unlock()

	// Signal TabMonitor to shut down
	mon.Stop()

	slog.Info("Tab closed", "tab", mon.TabID())
}

// handleTargetInfoChanged handles URL/title changes.
func (m *Manager) handleTargetInfoChanged(info *target.Info) {
	targetID := string(info.TargetID)

	m.mu.RLock()
	mon, exists := m.tabMonitors[targetID]
	m.mu.RUnlock()

	if !exists {
		return
	}

	// Check if site changed
	newSite := logger.ExtractSite(info.URL)
	if mon.HandleSiteChange(newSite, info.URL) {
		slog.Info("Tab navigated to new site", "tab", mon.TabID(), "site", newSite)
	}
}

// Stop gracefully shuts down the manager.
func (m *Manager) Stop() {
	slog.Info("Shutting down...")

	// Cancel browser context to stop event listening
	if m.browserCancel != nil {
		m.browserCancel()
	}

	// Cancel allocator context
	if m.allocatorCancel != nil {
		m.allocatorCancel()
	}

	// Stop all tab monitors
	m.mu.Lock()
	monitors := make([]*monitor.TabMonitor, 0, len(m.tabMonitors))
	for _, mon := range m.tabMonitors {
		monitors = append(monitors, mon)
	}
	m.tabMonitors = make(map[string]*monitor.TabMonitor)
	m.mu.Unlock()

	for _, mon := range monitors {
		mon.Stop()
	}

	// Close all log files
	if err := m.fileManager.Close(); err != nil {
		slog.Error("Error closing log files", "error", err)
	}

	// Stop Chrome if we launched it
	if m.chromeProcess != nil {
		if err := m.chromeProcess.Stop(); err != nil {
			slog.Error("Error stopping Chrome", "error", err)
		}
	}

	slog.Info("Shutdown complete")
}

// GetActiveTabCount returns the number of actively monitored tabs.
func (m *Manager) GetActiveTabCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.tabMonitors)
}

// IsConnected returns whether the manager is connected to Chrome.
func (m *Manager) IsConnected() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.connected
}

// isInternalURL checks if a URL belongs to an internal browser_tail tab.
// This includes about:blank, data: URLs (test page), and URLs containing browser_tail.
func isInternalURL(url string) bool {
	return url == "about:blank" ||
		strings.HasPrefix(url, "data:") ||
		strings.Contains(url, "browser_tail")
}
