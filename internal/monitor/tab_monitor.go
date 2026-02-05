// Package monitor provides per-tab event monitoring.
package monitor

import (
	"context"
	"encoding/json"
	"strings"
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
	"github.com/ajsharma/browser_tail/internal/redact"
)

// responseInfo stores response metadata for body capture.
type responseInfo struct {
	URL         string
	MimeType    string
	ContentSize float64
}

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
	redactor    *redact.Redactor

	// Request tracking for body capture.
	requestTracker map[network.RequestID]*responseInfo
	trackerMu      sync.RWMutex

	// Target context for CDP commands.
	targetCtx context.Context

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
		targetID:       targetID,
		tabID:          tabID,
		currentSite:    site,
		currentURL:     url,
		title:          title,
		sessionID:      sessionID,
		startTime:      time.Now(),
		fileManager:    fm,
		config:         cfg,
		redactor:       redact.New(cfg.Redact),
		requestTracker: make(map[network.RequestID]*responseInfo),
		ctx:            ctx,
		cancel:         cancel,
	}
}

// Start begins monitoring the tab.
func (tm *TabMonitor) Start(browserCtx context.Context) error {
	// Create chromedp.Context for this specific target
	targetCtx, cancel := chromedp.NewContext(browserCtx,
		chromedp.WithTargetID(target.ID(tm.targetID)),
	)
	defer cancel()

	// Store targetCtx for body capture
	tm.targetCtx = targetCtx

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

			// Apply redaction to headers.
			headers = tm.redactor.RedactHeaders(headers)

			tm.writeEvent(events.NewLogEvent(site, tabID, events.EventNetworkResponse, map[string]interface{}{
				"request_id":     ev.RequestID.String(),
				"url":            ev.Response.URL,
				"status":         ev.Response.Status,
				"status_text":    ev.Response.StatusText,
				"mime_type":      ev.Response.MimeType,
				"headers":        headers,
				"encoded_length": ev.Response.EncodedDataLength,
			}))

			// Store response info for body capture if enabled
			if cfg.CaptureBodies && tm.shouldCaptureBody(ev.Response.MimeType, ev.Response.EncodedDataLength) {
				tm.trackerMu.Lock()
				tm.requestTracker[ev.RequestID] = &responseInfo{
					URL:         ev.Response.URL,
					MimeType:    ev.Response.MimeType,
					ContentSize: ev.Response.EncodedDataLength,
				}
				tm.trackerMu.Unlock()
			}
		}

	case *network.EventLoadingFinished:
		// Capture body after loading finished (if configured)
		if cfg.EnableNetwork && cfg.CaptureBodies {
			tm.trackerMu.Lock()
			info, exists := tm.requestTracker[ev.RequestID]
			if exists {
				delete(tm.requestTracker, ev.RequestID)
			}
			tm.trackerMu.Unlock()

			if exists {
				go tm.captureBody(ev.RequestID, info, site, tabID)
			}
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
				args = append(args, extractRemoteObjectValue(arg))
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

// shouldCaptureBody checks if response body should be captured based on content type and size.
func (tm *TabMonitor) shouldCaptureBody(mimeType string, size float64) bool {
	// Check size limit
	maxSize := float64(tm.config.BodySizeLimitKB * 1024)
	if size > maxSize && size > 0 {
		return false
	}

	// Check content type against whitelist
	mimeType = strings.ToLower(mimeType)
	for _, allowed := range tm.config.BodyContentTypes {
		if matchContentType(mimeType, allowed) {
			return true
		}
	}

	return false
}

// matchContentType checks if a mime type matches a pattern (supports wildcards like "text/*").
func matchContentType(actual, pattern string) bool {
	// Remove parameters (e.g., "text/html; charset=utf-8" → "text/html")
	if idx := strings.Index(actual, ";"); idx != -1 {
		actual = strings.TrimSpace(actual[:idx])
	}

	// Handle wildcards
	if strings.HasSuffix(pattern, "/*") {
		prefix := strings.TrimSuffix(pattern, "/*")
		return strings.HasPrefix(actual, prefix+"/")
	}

	return actual == pattern
}

// captureBody retrieves and logs the response body.
func (tm *TabMonitor) captureBody(requestID network.RequestID, info *responseInfo, site, tabID string) {
	if tm.targetCtx == nil {
		return
	}

	// Get response body via CDP
	var body []byte
	var base64Encoded bool

	err := chromedp.Run(tm.targetCtx, chromedp.ActionFunc(func(ctx context.Context) error {
		result, err := network.GetResponseBody(requestID).Do(ctx)
		if err != nil {
			return err
		}
		body = result
		// Check if the response was base64 encoded (binary content)
		// The CDP returns raw bytes, we'll encode text as-is
		base64Encoded = false
		return nil
	}))
	if err != nil {
		// Body capture failed (response may have been cleared from cache)
		return
	}

	// Apply redaction to body content.
	bodyStr := string(body)
	bodyStr = tm.redactor.RedactBody(bodyStr)

	// Log the body as a separate event.
	tm.writeEvent(events.NewLogEvent(site, tabID, events.EventNetworkResponseBody, map[string]interface{}{
		"request_id":     requestID.String(),
		"url":            info.URL,
		"mime_type":      info.MimeType,
		"base64_encoded": base64Encoded,
		"body":           bodyStr,
	}))
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

// extractRemoteObjectValue extracts a usable value from a CDP RemoteObject.
// This handles primitives, objects, arrays, and special values like undefined/null.
func extractRemoteObjectValue(obj *runtime.RemoteObject) interface{} {
	if obj == nil {
		return nil
	}

	// Handle special unserializable values (Infinity, -Infinity, NaN, -0, bigint)
	if obj.UnserializableValue != "" {
		return string(obj.UnserializableValue)
	}

	// Handle primitive types with Value
	if obj.Value != nil {
		var v interface{}
		if err := json.Unmarshal(obj.Value, &v); err == nil {
			return v
		}
		// If unmarshal fails, return as string
		return string(obj.Value)
	}

	// Handle undefined
	if obj.Type == runtime.TypeUndefined {
		return "undefined"
	}

	// Handle null (subtype is "null")
	if obj.Subtype == runtime.SubtypeNull {
		return nil
	}

	// For objects/arrays/functions, use Preview if available for better detail
	if obj.Preview != nil {
		return extractObjectPreview(obj.Preview)
	}

	// Fallback to description (e.g., "[object Object]", "function foo()")
	if obj.Description != "" {
		return obj.Description
	}

	// Last resort: return the type
	return string(obj.Type)
}

// extractObjectPreview extracts a readable representation from an ObjectPreview.
func extractObjectPreview(preview *runtime.ObjectPreview) interface{} {
	if preview == nil {
		return nil
	}

	// For arrays, build an array representation
	if preview.Subtype == runtime.SubtypeArray {
		arr := make([]interface{}, 0, len(preview.Properties))
		for _, prop := range preview.Properties {
			arr = append(arr, extractPropertyValue(prop))
		}
		if preview.Overflow {
			arr = append(arr, "...")
		}
		return arr
	}

	// For objects, build a map representation
	obj := make(map[string]interface{})
	for _, prop := range preview.Properties {
		obj[prop.Name] = extractPropertyValue(prop)
	}
	if preview.Overflow {
		obj["..."] = "(truncated)"
	}
	return obj
}

// extractPropertyValue extracts a value from a PropertyPreview.
func extractPropertyValue(prop *runtime.PropertyPreview) interface{} {
	// Handle special unserializable values
	if prop.Value == "undefined" {
		return "undefined"
	}
	if prop.Value == "null" {
		return nil
	}

	// For primitive types, try to parse the value
	switch prop.Type {
	case runtime.TypeNumber:
		var v float64
		if err := json.Unmarshal([]byte(prop.Value), &v); err == nil {
			return v
		}
		return prop.Value
	case runtime.TypeBoolean:
		return prop.Value == "true"
	case runtime.TypeString:
		return prop.Value
	case runtime.TypeObject:
		if prop.Subtype == runtime.SubtypeNull {
			return nil
		}
		return prop.Value // e.g., "Object", "Array(3)"
	default:
		return prop.Value
	}
}
