# browser_tail

Capture Chrome browser activity to structured JSONL logs via Chrome DevTools Protocol (CDP).

## Features

- **Real-time logging**: Captures browser events as they happen with minimal latency
- **Structured output**: All events are logged in JSONL format for easy parsing
- **Per-site organization**: Logs are organized by site and tab ID (`logs/<site>/<tab_id>/session.log`)
- **Event types**: Network requests/responses, console messages, errors, page navigation
- **Privacy redaction**: Sensitive headers (cookies, auth) and body fields (passwords, tokens) are redacted by default
- **Body capture**: Optionally capture response bodies for text/JSON content
- **Browser automation**: Control mode for automated testing via CLI commands

## Installation

### Homebrew (macOS/Linux)

```bash
brew install ajsharma/tap/browser_tail
```

### Go install

```bash
go install github.com/ajsharma/browser_tail/cmd/browser_tail@latest
```

### Build from source

```bash
git clone https://github.com/ajsharma/browser_tail
cd browser_tail
go build -o browser_tail ./cmd/browser_tail
```

## Quick Start

### Connect to existing Chrome

Start Chrome with remote debugging enabled:

```bash
# macOS
/Applications/Google\ Chrome.app/Contents/MacOS/Google\ Chrome --remote-debugging-port=9222

# Linux
google-chrome --remote-debugging-port=9222

# Windows
chrome.exe --remote-debugging-port=9222
```

Then run browser_tail:

```bash
browser_tail
```

### Auto-launch Chrome

```bash
browser_tail --launch
```

### View logs in real-time

```bash
tail -f logs/*/*/session.log
```

## Usage

```
browser_tail [flags]

Flags:
  Connection:
    -p, --port string         Chrome remote debugging port (default "9222")
        --launch              Auto-launch Chrome with debugging enabled

  Output:
    -o, --output string       Output directory for log files (default "./logs")
        --flush-interval      Flush interval for log buffering (default 100ms)
        --buffer-size int     Buffer size per tab in bytes (default 8192)

  Privacy:
    -r, --redact              Enable header/body redaction (default true)
        --no-redact           Disable redaction
        --capture-bodies      Capture request/response bodies
        --body-size-limit int Max body size to capture in KB (default 10)

  Event Filtering:
        --network             Enable network events (default true)
        --console             Enable console events (default true)
        --errors              Enable error events (default true)
        --page                Enable page events (default true)
        --no-network          Disable network events
        --no-console          Disable console events
        --no-errors           Disable error events
        --no-page             Disable page events

  Configuration:
        --config string       Path to YAML config file

  General:
        --version             Show version info
    -h, --help                Show help
```

## Configuration File

Create a YAML config file for persistent settings:

```yaml
# config.yaml
chrome_port: "9222"
auto_launch: false
output_dir: "./logs"
flush_interval: 100ms
buffer_size: 8192

# Privacy
redact: true
capture_bodies: false
body_size_limit_kb: 10
body_content_types:
  - "text/*"
  - "application/json"

# Event filtering
enable_network: true
enable_console: true
enable_errors: true
enable_page: true
```

Use with:

```bash
browser_tail --config config.yaml
```

## Browser Control Mode

Control the browser programmatically for automated testing:

```bash
# Navigate to a URL
browser_tail control navigate --url https://example.com

# Click an element
browser_tail control click --selector "button#submit"

# Type into an input
browser_tail control type --selector "input[name=search]" --text "query"

# Execute JavaScript
browser_tail control eval --js "document.title"

# Take a screenshot
browser_tail control screenshot --output screenshot.png

# Get page info
browser_tail control title
browser_tail control url
browser_tail control text --selector "h1"
```

## Log Format

Events are logged in JSONL format (one JSON object per line):

```json
{"timestamp":"2024-01-15T10:30:00.123Z","site":"example.com","tab_id":"tab-1","event_type":"page.navigate","data":{"url":"https://example.com/page","referrer":"","type":"navigation"}}
{"timestamp":"2024-01-15T10:30:00.456Z","site":"example.com","tab_id":"tab-1","event_type":"network.request","data":{"request_id":"123","url":"https://example.com/api/data","method":"GET","type":"XHR"}}
{"timestamp":"2024-01-15T10:30:00.789Z","site":"example.com","tab_id":"tab-1","event_type":"network.response","data":{"request_id":"123","url":"https://example.com/api/data","status":200,"mime_type":"application/json","headers":{"content-type":"application/json","cookie":"[REDACTED]"}}}
```

### Event Types

| Event Type | Description |
|------------|-------------|
| `meta.session_start` | Session started |
| `meta.tab_created` | New tab opened |
| `meta.tab_closed` | Tab closed |
| `meta.site_changed` | Tab navigated to different site |
| `meta.site_entered` | Tab entered a site |
| `page.navigate` | Page navigation |
| `page.load` | Page load complete |
| `page.dom_ready` | DOM content loaded |
| `network.request` | Network request sent |
| `network.response` | Network response received |
| `network.response_body` | Response body captured |
| `network.failure` | Network request failed |
| `console.log` | console.log() |
| `console.warn` | console.warn() |
| `console.error` | console.error() |
| `console.info` | console.info() |
| `console.debug` | console.debug() |
| `error.runtime` | JavaScript runtime error |

## Privacy & Redaction

By default, sensitive data is redacted:

**Headers redacted:**
- Cookie, Set-Cookie
- Authorization, Proxy-Authorization
- X-API-Key, X-Auth-Token
- X-CSRF-Token, X-XSRF-Token

**Body fields redacted (in JSON responses):**
- password, passwd, secret
- token, apikey, api_key
- accesstoken, access_token
- refreshtoken, refresh_token
- private_key, client_secret
- credential, auth, ssn
- credit_card, card_number, cvv, pin

To disable redaction:

```bash
browser_tail --no-redact
```

## Directory Structure

```
logs/
├── example.com/
│   ├── tab-1/
│   │   └── session.log
│   └── tab-2/
│       └── session.log
├── github.com/
│   └── tab-1/
│       └── session.log
└── localhost_3000/
    └── tab-3/
        └── session.log
```

## Requirements

- Go 1.21 or later
- Google Chrome or Chromium with remote debugging enabled

## License

MIT
