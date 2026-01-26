// Package events defines log event types and transformations.
package events

import (
	"time"
)

// LogEvent represents a single logged event in JSONL format.
type LogEvent struct {
	Timestamp string                 `json:"timestamp"`
	Site      string                 `json:"site"`
	TabID     string                 `json:"tab_id"`
	EventType string                 `json:"event_type"`
	Data      map[string]interface{} `json:"data"`
}

// NewLogEvent creates a new LogEvent with the current timestamp.
func NewLogEvent(site, tabID, eventType string, data map[string]interface{}) *LogEvent {
	return &LogEvent{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Site:      site,
		TabID:     tabID,
		EventType: eventType,
		Data:      data,
	}
}

// Event type constants for meta events.
const (
	EventMetaSessionStart = "meta.session_start"
	EventMetaTabCreated   = "meta.tab_created"
	EventMetaTabClosed    = "meta.tab_closed"
	EventMetaSiteChanged  = "meta.site_changed"
	EventMetaSiteEntered  = "meta.site_entered"
	EventMetaEnvironment  = "meta.environment"
)

// Event type constants for page events.
const (
	EventPageOpen     = "page.open"
	EventPageNavigate = "page.navigate"
	EventPageReload   = "page.reload"
	EventPageClose    = "page.close"
	EventPageLoad     = "page.load"
	EventPageDOMReady = "page.dom_ready"
)

// Event type constants for network events.
const (
	EventNetworkRequest      = "network.request"
	EventNetworkResponse     = "network.response"
	EventNetworkResponseBody = "network.response_body"
	EventNetworkFailure      = "network.failure"
)

// Event type constants for console events.
const (
	EventConsoleLog     = "console.log"
	EventConsoleWarn    = "console.warn"
	EventConsoleInfo    = "console.info"
	EventConsoleError   = "console.error"
	EventConsoleDebug   = "console.debug"
	EventConsoleVerbose = "console.verbose"
)

// Event type constants for error events.
const (
	EventErrorRuntime          = "error.runtime"
	EventErrorUnhandledPromise = "error.unhandled_promise"
)

// NewSessionStartEvent creates a meta.session_start event.
func NewSessionStartEvent(sessionID string, chromePID int, version string) *LogEvent {
	return NewLogEvent("_meta", "_session", EventMetaSessionStart, map[string]interface{}{
		"session_id":           sessionID,
		"chrome_pid":           chromePID,
		"browser_tail_version": version,
		"start_time":           time.Now().UTC().Format(time.RFC3339Nano),
	})
}

// NewTabCreatedEvent creates a meta.tab_created event.
func NewTabCreatedEvent(site, tabID, sessionID, targetID, title, url string) *LogEvent {
	return NewLogEvent(site, tabID, EventMetaTabCreated, map[string]interface{}{
		"session_id": sessionID,
		"target_id":  targetID,
		"title":      title,
		"url":        url,
	})
}

// NewTabClosedEvent creates a meta.tab_closed event.
func NewTabClosedEvent(site, tabID, sessionID, targetID string, durationSeconds float64) *LogEvent {
	return NewLogEvent(site, tabID, EventMetaTabClosed, map[string]interface{}{
		"session_id":       sessionID,
		"target_id":        targetID,
		"duration_seconds": durationSeconds,
	})
}

// NewSiteChangedEvent creates a meta.site_changed event.
func NewSiteChangedEvent(oldSite, tabID, newSite, newURL string) *LogEvent {
	return NewLogEvent(oldSite, tabID, EventMetaSiteChanged, map[string]interface{}{
		"old_site": oldSite,
		"new_site": newSite,
		"new_url":  newURL,
	})
}

// NewSiteEnteredEvent creates a meta.site_entered event.
func NewSiteEnteredEvent(site, tabID, fromSite, url string) *LogEvent {
	return NewLogEvent(site, tabID, EventMetaSiteEntered, map[string]interface{}{
		"from_site": fromSite,
		"url":       url,
	})
}

// NewPageNavigateEvent creates a page.navigate event.
func NewPageNavigateEvent(site, tabID, url, referrer, navigationType string) *LogEvent {
	return NewLogEvent(site, tabID, EventPageNavigate, map[string]interface{}{
		"url":             url,
		"referrer":        referrer,
		"navigation_type": navigationType,
	})
}

// NewPageLoadEvent creates a page.load event.
func NewPageLoadEvent(site, tabID, url string) *LogEvent {
	return NewLogEvent(site, tabID, EventPageLoad, map[string]interface{}{
		"url": url,
	})
}

// NewPageDOMReadyEvent creates a page.dom_ready event.
func NewPageDOMReadyEvent(site, tabID, url string) *LogEvent {
	return NewLogEvent(site, tabID, EventPageDOMReady, map[string]interface{}{
		"url": url,
	})
}
