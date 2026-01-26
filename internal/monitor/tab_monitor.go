// Package monitor provides per-tab event monitoring.
package monitor

import (
	"context"
	"sync"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"

	"github.com/ajsharma/browser_tail/internal/config"
	"github.com/ajsharma/browser_tail/internal/events"
	"github.com/ajsharma/browser_tail/internal/logger"
)

// TabMonitor monitors a single browser tab.
type TabMonitor struct {
	targetID    string
	tabID       string
	currentSite string
	currentURL  string
	title       string
	sessionID   string
	startTime   time.Time

	fileManager *logger.FileManager
	config      *config.Config

	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.RWMutex
}

// NewTabMonitor creates a new tab monitor.
func NewTabMonitor(
	parentCtx context.Context,
	targetID, tabID, site, title, url, sessionID string,
	fm *logger.FileManager,
	cfg *config.Config,
) *TabMonitor {
	ctx, cancel := context.WithCancel(parentCtx)

	return &TabMonitor{
		targetID:    targetID,
		tabID:       tabID,
		currentSite: site,
		currentURL:  url,
		title:       title,
		sessionID:   sessionID,
		startTime:   time.Now(),
		fileManager: fm,
		config:      cfg,
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Start begins monitoring the tab.
func (tm *TabMonitor) Start(browserCtx context.Context) error {
	// Create chromedp.Context for this specific target
	targetCtx, cancel := chromedp.NewContext(browserCtx,
		chromedp.WithTargetID(target.ID(tm.targetID)),
	)
	defer cancel()

	// Enable required CDP domains
	if err := chromedp.Run(targetCtx,
		page.Enable(),
		runtime.Enable(),
	); err != nil {
		return err
	}

	// Enable network if configured
	if tm.config.EnableNetwork {
		if err := chromedp.Run(targetCtx, network.Enable()); err != nil {
			return err
		}
	}

	// Write tab created event
	tm.writeEvent(events.NewTabCreatedEvent(
		tm.currentSite,
		tm.tabID,
		tm.sessionID,
		tm.targetID,
		tm.title,
		tm.currentURL,
	))

	// Setup event listeners
	// NOTE: Do NOT listen for Target.targetDestroyed here
	// Manager owns lifecycle events and signals shutdown via context cancellation
	chromedp.ListenTarget(targetCtx, func(ev interface{}) {
		tm.handleEvent(ev)
	})

	// Wait for context cancellation
	select {
	case <-tm.ctx.Done():
	case <-targetCtx.Done():
	}

	return nil
}

// handleEvent processes CDP events.
func (tm *TabMonitor) handleEvent(ev interface{}) {
	tm.mu.RLock()
	site := tm.currentSite
	tabID := tm.tabID
	cfg := tm.config
	tm.mu.RUnlock()

	switch ev := ev.(type) {
	// Page events
	case *page.EventFrameNavigated:
		if cfg.EnablePage && ev.Frame.ParentID == "" { // Main frame only
			tm.mu.Lock()
			tm.currentURL = ev.Frame.URL
			tm.mu.Unlock()

			tm.writeEvent(events.NewPageNavigateEvent(
				site,
				tabID,
				ev.Frame.URL,
				"", // referrer not available in this event
				"navigation",
			))
		}

	case *page.EventLoadEventFired:
		if cfg.EnablePage {
			tm.mu.RLock()
			url := tm.currentURL
			tm.mu.RUnlock()

			tm.writeEvent(events.NewPageLoadEvent(site, tabID, url))
		}

	case *page.EventDomContentEventFired:
		if cfg.EnablePage {
			tm.mu.RLock()
			url := tm.currentURL
			tm.mu.RUnlock()

			tm.writeEvent(events.NewPageDOMReadyEvent(site, tabID, url))
		}

	// Network events
	case *network.EventRequestWillBeSent:
		if cfg.EnableNetwork {
			tm.writeEvent(events.NewLogEvent(site, tabID, events.EventNetworkRequest, map[string]interface{}{
				"request_id": ev.RequestID.String(),
				"url":        ev.Request.URL,
				"method":     ev.Request.Method,
				"type":       ev.Type.String(),
			}))
		}

	case *network.EventResponseReceived:
		if cfg.EnableNetwork {
			headers := make(map[string]interface{})
			for k, v := range ev.Response.Headers {
				headers[k] = v
			}

			tm.writeEvent(events.NewLogEvent(site, tabID, events.EventNetworkResponse, map[string]interface{}{
				"request_id":     ev.RequestID.String(),
				"url":            ev.Response.URL,
				"status":         ev.Response.Status,
				"status_text":    ev.Response.StatusText,
				"mime_type":      ev.Response.MimeType,
				"headers":        headers,
				"encoded_length": ev.Response.EncodedDataLength,
			}))
		}

	case *network.EventLoadingFailed:
		if cfg.EnableNetwork {
			tm.writeEvent(events.NewLogEvent(site, tabID, events.EventNetworkFailure, map[string]interface{}{
				"request_id": ev.RequestID.String(),
				"error_text": ev.ErrorText,
				"canceled":   ev.Canceled,
				"blocked":    ev.BlockedReason.String(),
				"cors_error": ev.CorsErrorStatus,
			}))
		}

	// Console events
	case *runtime.EventConsoleAPICalled:
		if cfg.EnableConsole {
			eventType := events.EventConsoleLog
			switch ev.Type {
			case runtime.APITypeWarning:
				eventType = events.EventConsoleWarn
			case runtime.APITypeError:
				eventType = events.EventConsoleError
			case runtime.APITypeInfo:
				eventType = events.EventConsoleInfo
			case runtime.APITypeDebug:
				eventType = events.EventConsoleDebug
			}

			args := make([]interface{}, 0, len(ev.Args))
			for _, arg := range ev.Args {
				if arg.Value != nil {
					args = append(args, arg.Value)
				} else if arg.Description != "" {
					args = append(args, arg.Description)
				}
			}

			tm.writeEvent(events.NewLogEvent(site, tabID, eventType, map[string]interface{}{
				"args": args,
			}))
		}

	// Error events
	case *runtime.EventExceptionThrown:
		if cfg.EnableErrors {
			details := ev.ExceptionDetails
			tm.writeEvent(events.NewLogEvent(site, tabID, events.EventErrorRuntime, map[string]interface{}{
				"text":      details.Text,
				"line":      details.LineNumber,
				"column":    details.ColumnNumber,
				"url":       details.URL,
				"script_id": details.ScriptID,
			}))
		}
	}
}

// writeEvent writes an event to the log file.
func (tm *TabMonitor) writeEvent(ev *events.LogEvent) {
	// Errors are non-fatal - monitoring continues even if writes fail
	_ = tm.fileManager.WriteEvent(tm.tabID, ev)
}

// HandleSiteChange handles navigation to a different site.
// Returns true if the site actually changed.
func (tm *TabMonitor) HandleSiteChange(newSite, newURL string) bool {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if newSite == tm.currentSite {
		tm.currentURL = newURL
		return false
	}

	oldSite := tm.currentSite

	// Write meta event to old log (errors are non-fatal)
	if err := tm.fileManager.WriteEvent(tm.tabID, events.NewSiteChangedEvent(
		oldSite,
		tm.tabID,
		newSite,
		newURL,
	)); err != nil {
		// Log continues even if write fails
		_ = err
	}

	// Close old log file (errors are non-fatal)
	if err := tm.fileManager.CloseTab(tm.tabID, oldSite); err != nil {
		_ = err
	}

	// Update current site
	tm.currentSite = newSite
	tm.currentURL = newURL

	// Write meta event to new log (errors are non-fatal)
	if err := tm.fileManager.WriteEvent(tm.tabID, events.NewSiteEnteredEvent(
		newSite,
		tm.tabID,
		oldSite,
		newURL,
	)); err != nil {
		_ = err
	}

	return true
}

// Stop gracefully stops the tab monitor.
func (tm *TabMonitor) Stop() {
	tm.mu.RLock()
	site := tm.currentSite
	tabID := tm.tabID
	sessionID := tm.sessionID
	targetID := tm.targetID
	startTime := tm.startTime
	tm.mu.RUnlock()

	// Write tab closed event
	duration := time.Since(startTime).Seconds()
	tm.writeEvent(events.NewTabClosedEvent(
		site,
		tabID,
		sessionID,
		targetID,
		duration,
	))

	// Close log file (errors are non-fatal during shutdown)
	if err := tm.fileManager.CloseTab(tabID, site); err != nil {
		_ = err
	}

	// Cancel context
	tm.cancel()
}

// TabID returns the tab ID.
func (tm *TabMonitor) TabID() string {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.tabID
}

// CurrentSite returns the current site.
func (tm *TabMonitor) CurrentSite() string {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.currentSite
}

// CurrentURL returns the current URL.
func (tm *TabMonitor) CurrentURL() string {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.currentURL
}
