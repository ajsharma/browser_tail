// Package events defines log event types and transformations.
package events

import (
	"time"
)

// LogEvent represents a single logged event in JSONL format.
type LogEvent struct {
	Timestamp string      `json:"timestamp"`
	Site      string      `json:"site"`
	TabID     string      `json:"tab_id"`
	EventType string      `json:"event_type"`
	Data      interface{} `json:"data"`
}

// NewLogEvent creates a new LogEvent with the current timestamp.
func NewLogEvent(site, tabID, eventType string, data interface{}) *LogEvent {
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

// SessionStartData holds data for meta.session_start events.
type SessionStartData struct {
	SessionID          string `json:"session_id"`
	ChromePID          int    `json:"chrome_pid"`
	BrowserTailVersion string `json:"browser_tail_version"`
	StartTime          string `json:"start_time"`
}

// TabCreatedData holds data for meta.tab_created events.
type TabCreatedData struct {
	SessionID string `json:"session_id"`
	TargetID  string `json:"target_id"`
	Title     string `json:"title"`
	URL       string `json:"url"`
}

// TabClosedData holds data for meta.tab_closed events.
type TabClosedData struct {
	SessionID       string  `json:"session_id"`
	TargetID        string  `json:"target_id"`
	DurationSeconds float64 `json:"duration_seconds"`
}

// SiteChangedData holds data for meta.site_changed events.
type SiteChangedData struct {
	OldSite string `json:"old_site"`
	NewSite string `json:"new_site"`
	NewURL  string `json:"new_url"`
}

// SiteEnteredData holds data for meta.site_entered events.
type SiteEnteredData struct {
	FromSite string `json:"from_site"`
	URL      string `json:"url"`
}

// PageNavigateData holds data for page.navigate events.
type PageNavigateData struct {
	URL            string `json:"url"`
	Referrer       string `json:"referrer"`
	NavigationType string `json:"navigation_type"`
}

// PageLoadData holds data for page.load events.
type PageLoadData struct {
	URL string `json:"url"`
}

// PageDOMReadyData holds data for page.dom_ready events.
type PageDOMReadyData struct {
	URL string `json:"url"`
}

// NetworkRequestData holds data for network.request events.
type NetworkRequestData struct {
	RequestID string `json:"request_id"`
	URL       string `json:"url"`
	Method    string `json:"method"`
	Type      string `json:"type"`
}

// NetworkResponseData holds data for network.response events.
type NetworkResponseData struct {
	RequestID     string                 `json:"request_id"`
	URL           string                 `json:"url"`
	Status        int64                  `json:"status"`
	StatusText    string                 `json:"status_text"`
	MimeType      string                 `json:"mime_type"`
	Headers       map[string]interface{} `json:"headers"`
	EncodedLength float64                `json:"encoded_length"`
}

// NetworkResponseBodyData holds data for network.response_body events.
type NetworkResponseBodyData struct {
	RequestID     string `json:"request_id"`
	URL           string `json:"url"`
	MimeType      string `json:"mime_type"`
	Base64Encoded bool   `json:"base64_encoded"`
	Body          string `json:"body"`
}

// NetworkFailureData holds data for network.failure events.
type NetworkFailureData struct {
	RequestID  string      `json:"request_id"`
	ErrorText  string      `json:"error_text"`
	Canceled   bool        `json:"canceled"`
	Blocked    string      `json:"blocked"`
	CORSError  interface{} `json:"cors_error"`
}

// ConsoleData holds data for console.* events.
type ConsoleData struct {
	Args []interface{} `json:"args"`
}

// RuntimeErrorData holds data for error.runtime events.
type RuntimeErrorData struct {
	Text     string `json:"text"`
	Line     int64  `json:"line"`
	Column   int64  `json:"column"`
	URL      string `json:"url"`
	ScriptID string `json:"script_id"`
}

// NewSessionStartEvent creates a meta.session_start event.
func NewSessionStartEvent(sessionID string, chromePID int, version string) *LogEvent {
	return NewLogEvent("_meta", "_session", EventMetaSessionStart, &SessionStartData{
		SessionID:          sessionID,
		ChromePID:          chromePID,
		BrowserTailVersion: version,
		StartTime:          time.Now().UTC().Format(time.RFC3339Nano),
	})
}

// NewTabCreatedEvent creates a meta.tab_created event.
func NewTabCreatedEvent(site, tabID, sessionID, targetID, title, url string) *LogEvent {
	return NewLogEvent(site, tabID, EventMetaTabCreated, &TabCreatedData{
		SessionID: sessionID,
		TargetID:  targetID,
		Title:     title,
		URL:       url,
	})
}

// NewTabClosedEvent creates a meta.tab_closed event.
func NewTabClosedEvent(site, tabID, sessionID, targetID string, durationSeconds float64) *LogEvent {
	return NewLogEvent(site, tabID, EventMetaTabClosed, &TabClosedData{
		SessionID:       sessionID,
		TargetID:        targetID,
		DurationSeconds: durationSeconds,
	})
}

// NewSiteChangedEvent creates a meta.site_changed event.
func NewSiteChangedEvent(oldSite, tabID, newSite, newURL string) *LogEvent {
	return NewLogEvent(oldSite, tabID, EventMetaSiteChanged, &SiteChangedData{
		OldSite: oldSite,
		NewSite: newSite,
		NewURL:  newURL,
	})
}

// NewSiteEnteredEvent creates a meta.site_entered event.
func NewSiteEnteredEvent(site, tabID, fromSite, url string) *LogEvent {
	return NewLogEvent(site, tabID, EventMetaSiteEntered, &SiteEnteredData{
		FromSite: fromSite,
		URL:      url,
	})
}

// NewPageNavigateEvent creates a page.navigate event.
func NewPageNavigateEvent(site, tabID, url, referrer, navigationType string) *LogEvent {
	return NewLogEvent(site, tabID, EventPageNavigate, &PageNavigateData{
		URL:            url,
		Referrer:       referrer,
		NavigationType: navigationType,
	})
}

// NewPageLoadEvent creates a page.load event.
func NewPageLoadEvent(site, tabID, url string) *LogEvent {
	return NewLogEvent(site, tabID, EventPageLoad, &PageLoadData{
		URL: url,
	})
}

// NewPageDOMReadyEvent creates a page.dom_ready event.
func NewPageDOMReadyEvent(site, tabID, url string) *LogEvent {
	return NewLogEvent(site, tabID, EventPageDOMReady, &PageDOMReadyData{
		URL: url,
	})
}
