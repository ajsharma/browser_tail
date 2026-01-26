package cdp

import (
	"context"
	"fmt"
	"log"
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
	config        *config.Config
	fileManager   *logger.FileManager
	tabRegistry   *logger.TabRegistry
	chromeProcess *ChromeProcess
	tabMonitors   map[string]*monitor.TabMonitor // targetID -> monitor
	mu            sync.RWMutex
	browserCtx    context.Context
	browserCancel context.CancelFunc
}

// NewManager creates a new CDP Manager.
func NewManager(cfg *config.Config, fm *logger.FileManager) *Manager {
	return &Manager{
		config:      cfg,
		fileManager: fm,
		tabRegistry: logger.NewTabRegistry(),
		tabMonitors: make(map[string]*monitor.TabMonitor),
	}
}

// Start begins monitoring Chrome.
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
			_ = m.chromeProcess.Stop() // Best effort cleanup
			return fmt.Errorf("chrome not ready: %w", err)
		}

		log.Printf("Launched Chrome (PID: %d) on port %s", m.chromeProcess.PID(), m.config.ChromePort)
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
		log.Printf("Warning: failed to write session start event: %v", err)
	}

	// Step 1: Initial discovery via /json (ONE-TIME)
	initialTabs, err := DiscoverTabs(m.config.ChromePort)
	if err != nil {
		return fmt.Errorf("failed to discover initial tabs: %w", err)
	}

	log.Printf("Discovered %d existing tab(s)", len(initialTabs))

	// Get browser info for WebSocket URL
	browserInfo, err := DiscoverBrowserInfo(m.config.ChromePort)
	if err != nil {
		return fmt.Errorf("failed to get browser info: %w", err)
	}

	// Step 2: Connect to browser-level CDP for target events
	allocatorCtx, allocatorCancel := chromedp.NewRemoteAllocator(ctx, browserInfo.WebSocketDebuggerURL)
	defer allocatorCancel()

	m.browserCtx, m.browserCancel = chromedp.NewContext(allocatorCtx)
	defer m.browserCancel()

	// Step 3: Enable target discovery (NO POLLING)
	if err := chromedp.Run(m.browserCtx, target.SetDiscoverTargets(true)); err != nil {
		return fmt.Errorf("failed to enable target discovery: %w", err)
	}

	// Step 4: Listen for target events (REAL-TIME)
	chromedp.ListenTarget(m.browserCtx, func(ev interface{}) {
		switch ev := ev.(type) {
		case *target.EventTargetCreated:
			// Only monitor page targets that are not yet attached
			if ev.TargetInfo.Type == TargetTypePage && !ev.TargetInfo.Attached {
				m.handleNewTarget(ctx, ev.TargetInfo)
			}

		case *target.EventTargetDestroyed:
			// Tab closed - Manager owns this event
			m.handleTargetDestroyed(string(ev.TargetID))

		case *target.EventTargetInfoChanged:
			// Tab URL/title changed
			if ev.TargetInfo.Type == TargetTypePage {
				m.handleTargetInfoChanged(ev.TargetInfo)
			}
		}
	})

	// Start monitoring existing tabs
	for _, tab := range initialTabs {
		m.handleNewTarget(ctx, &target.Info{
			TargetID: target.ID(tab.TargetID),
			Type:     tab.Type,
			Title:    tab.Title,
			URL:      tab.URL,
			Attached: false,
		})
	}

	log.Printf("Monitoring started (session: %s)", m.tabRegistry.GetSessionID())

	// Block until context cancelled
	<-ctx.Done()

	return nil
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
			log.Printf("Tab monitor error (tab %s): %v", tabID, err)
		}
	}()

	log.Printf("Started monitoring tab %s (%s) - %s", tabID, targetID[:8], info.URL)
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

	log.Printf("Tab closed: %s", mon.TabID())
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
		log.Printf("Tab %s navigated to new site: %s", mon.TabID(), newSite)
	}
}

// Stop gracefully shuts down the manager.
func (m *Manager) Stop() {
	log.Println("Shutting down...")

	// Cancel browser context to stop event listening
	if m.browserCancel != nil {
		m.browserCancel()
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
		log.Printf("Error closing log files: %v", err)
	}

	// Stop Chrome if we launched it
	if m.chromeProcess != nil {
		if err := m.chromeProcess.Stop(); err != nil {
			log.Printf("Error stopping Chrome: %v", err)
		}
	}

	log.Println("Shutdown complete")
}

// GetActiveTabCount returns the number of actively monitored tabs.
func (m *Manager) GetActiveTabCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.tabMonitors)
}
