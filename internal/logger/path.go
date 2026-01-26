// Package logger provides log file management for browser_tail.
package logger

import (
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/google/uuid"
)

// UnknownSite is the default site name for unknown or invalid URLs.
const UnknownSite = "unknown"

var (
	sessionID   string
	sessionOnce sync.Once
	tabCounter  atomic.Int64
)

// GetSessionID returns the unique session ID for this browser_tail process.
// The session ID is generated once and remains constant for the process lifetime.
func GetSessionID() string {
	sessionOnce.Do(func() {
		sessionID = uuid.New().String()
	})
	return sessionID
}

// GenerateTabID creates a new unique tab ID.
// Tab IDs are sequential (tab-1, tab-2, etc.) and stable within the process lifetime.
func GenerateTabID() string {
	id := tabCounter.Add(1)
	return "tab-" + strings.TrimPrefix(string(rune('0'+id)), "0")
}

// TabRegistry maintains a mapping between CDP target IDs and stable tab IDs.
type TabRegistry struct {
	sessionID   string
	targetToTab map[string]string
	mu          sync.RWMutex
}

// NewTabRegistry creates a new tab registry for the current session.
func NewTabRegistry() *TabRegistry {
	return &TabRegistry{
		sessionID:   GetSessionID(),
		targetToTab: make(map[string]string),
	}
}

// GetOrCreateTabID returns the tab ID for a given target ID.
// If no tab ID exists, a new one is created and stored.
func (r *TabRegistry) GetOrCreateTabID(targetID string) string {
	r.mu.RLock()
	if tabID, exists := r.targetToTab[targetID]; exists {
		r.mu.RUnlock()
		return tabID
	}
	r.mu.RUnlock()

	r.mu.Lock()
	defer r.mu.Unlock()

	// Double-check after acquiring write lock
	if tabID, exists := r.targetToTab[targetID]; exists {
		return tabID
	}

	tabID := GenerateTabID()
	r.targetToTab[targetID] = tabID
	return tabID
}

// GetTabID returns the tab ID for a given target ID, or empty string if not found.
func (r *TabRegistry) GetTabID(targetID string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.targetToTab[targetID]
}

// RemoveTarget removes a target from the registry.
func (r *TabRegistry) RemoveTarget(targetID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.targetToTab, targetID)
}

// GetSessionID returns the session ID for this registry.
func (r *TabRegistry) GetSessionID() string {
	return r.sessionID
}

// SanitizeSiteName converts a URL hostname into a safe directory name.
func SanitizeSiteName(hostname string) string {
	if hostname == "" {
		return UnknownSite
	}

	// Handle localhost with port
	if strings.Contains(hostname, ":") {
		parts := strings.SplitN(hostname, ":", 2)
		host := parts[0]
		port := parts[1]

		// Include port in directory name for localhost
		if host == "localhost" || host == "127.0.0.1" || host == "0.0.0.0" {
			hostname = host + "_" + port
		} else {
			hostname = host
		}
	}

	// Replace invalid filesystem characters
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
		" ", "_",
	)
	result := replacer.Replace(hostname)

	// Truncate to 255 characters (filesystem limit)
	if len(result) > 255 {
		result = result[:255]
	}

	return result
}

// ExtractSite extracts and sanitizes the site name from a URL.
func ExtractSite(urlStr string) string {
	if urlStr == "" {
		return UnknownSite
	}

	u, err := url.Parse(urlStr)
	if err != nil {
		return UnknownSite
	}

	hostname := u.Hostname()
	if hostname == "" {
		// Handle special URLs like about:blank, chrome://
		if u.Scheme != "" {
			return SanitizeSiteName(u.Scheme + "_" + u.Opaque)
		}
		return UnknownSite
	}

	port := u.Port()

	// Include port for localhost
	if port != "" && (hostname == "localhost" || hostname == "127.0.0.1" || hostname == "0.0.0.0") {
		return SanitizeSiteName(hostname + ":" + port)
	}

	return SanitizeSiteName(hostname)
}

// GetLogPath returns the full path to the log file for a given site and tab ID.
func GetLogPath(baseDir, site, tabID string) string {
	return filepath.Join(baseDir, site, tabID, "session.log")
}
