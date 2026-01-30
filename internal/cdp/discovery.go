package cdp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// TargetTypePage is the CDP target type for browser pages.
const TargetTypePage = "page"

// Tab represents a Chrome tab/target discovered via CDP.
type Tab struct {
	TargetID string
	Type     string
	Title    string
	URL      string
}

// BrowserInfo holds information about the connected Chrome instance.
type BrowserInfo struct {
	Browser              string `json:"Browser"`
	ProtocolVersion      string `json:"Protocol-Version"`
	UserAgent            string `json:"User-Agent"`
	V8Version            string `json:"V8-Version"`
	WebKitVersion        string `json:"WebKit-Version"`
	WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
}

// targetJSON represents the JSON response from /json endpoint.
type targetJSON struct {
	ID                   string `json:"id"`
	Type                 string `json:"type"`
	Title                string `json:"title"`
	URL                  string `json:"url"`
	WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
	DevtoolsFrontendURL  string `json:"devtoolsFrontendUrl"`
}

// DiscoverBrowserInfo queries the /json/version endpoint to get browser info.
func DiscoverBrowserInfo(port string) (*BrowserInfo, error) {
	client := &http.Client{Timeout: 5 * time.Second}

	resp, err := client.Get(fmt.Sprintf("http://localhost:%s/json/version", port))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Chrome on port %s: %w", port, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var info BrowserInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to decode browser info: %w", err)
	}

	return &info, nil
}

// DiscoverTabs queries the /json endpoint to discover all open tabs.
// This should be called ONCE for initial discovery; ongoing monitoring uses CDP events.
func DiscoverTabs(port string) ([]*Tab, error) {
	client := &http.Client{Timeout: 5 * time.Second}

	resp, err := client.Get(fmt.Sprintf("http://localhost:%s/json", port))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Chrome on port %s: %w", port, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var targets []targetJSON
	if err := json.NewDecoder(resp.Body).Decode(&targets); err != nil {
		return nil, fmt.Errorf("failed to decode targets: %w", err)
	}

	var tabs []*Tab
	for _, target := range targets {
		// Only include page targets
		if target.Type == TargetTypePage {
			tabs = append(tabs, &Tab{
				TargetID: target.ID,
				Type:     target.Type,
				Title:    target.Title,
				URL:      target.URL,
			})
		}
	}

	return tabs, nil
}

// WaitForChrome waits for Chrome to be available on the specified port.
// It waits for both the /json/version endpoint AND at least one page target.
func WaitForChrome(port string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 1 * time.Second}

	versionReady := false
	for time.Now().Before(deadline) {
		// First check if /json/version responds
		if !versionReady {
			resp, err := client.Get(fmt.Sprintf("http://localhost:%s/json/version", port))
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					versionReady = true
				}
			}
		}

		// Then check if there's at least one page target
		if versionReady {
			tabs, err := DiscoverTabs(port)
			if err == nil && len(tabs) > 0 {
				return nil
			}
		}

		time.Sleep(100 * time.Millisecond)
	}

	if !versionReady {
		return fmt.Errorf("chrome not available on port %s after %v", port, timeout)
	}
	return fmt.Errorf("chrome available but no page targets after %v", timeout)
}

// OpenNewTab opens a new tab in Chrome using the HTTP debugging API.
func OpenNewTab(port, targetURL string) error {
	client := &http.Client{Timeout: 5 * time.Second}

	// URL-encode the target URL as a query parameter
	apiURL := fmt.Sprintf("http://localhost:%s/json/new?%s", port, url.QueryEscape(targetURL))

	req, err := http.NewRequest(http.MethodPut, apiURL, nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	return nil
}

// closeTabViaHTTP closes a Chrome tab using the HTTP debugging API.
func closeTabViaHTTP(port, targetID string) error {
	client := &http.Client{Timeout: 5 * time.Second}

	req, err := http.NewRequest(http.MethodPut, fmt.Sprintf("http://localhost:%s/json/close/%s", port, targetID), nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	return nil
}
