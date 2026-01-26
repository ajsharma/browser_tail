# Browser Session Logging Tool — Requirements Document (Revised)

## 1. Overview

Build a local tool that captures detailed activity from a **user-controlled Chrome session** and continuously writes structured logs to disk.

The user drives the browser normally (clicking, typing, navigating).
The tool **observes and logs** activity without interfering.

Optionally, the tool may support AI-driven control **in addition to** user control, but user control is the default and required mode.

All events for a given **site + browser tab** are written to a **single append-only log file**.

---

## 2. Goals

### Primary Goals

* Capture rich browser activity without automating or hijacking user behavior
* Persist logs locally in a deterministic, minimal file structure
* Support long, interactive browsing sessions
* Produce logs that are easy to stream, tail, and ingest by agents

### Non-Goals

* No mandatory browser automation
* No UI beyond the browser itself
* No cloud upload or remote telemetry

---

## 3. Browser Control Model

### User-Controlled Session (Required)

* The browser is:

  * Fully interactive
  * Controlled by mouse/keyboard
  * Indistinguishable from normal Chrome usage
* The tool must **not** block or delay user actions

### AI-Controlled Session (Optional / Bonus)

* If implemented:

  * AI actions must be additive (not exclusive)
  * User can override or interrupt AI at any time
* Example:

  * User opens a tab
  * AI performs inspection or navigation inside that tab

---

## 4. Chrome Session Requirements

* Must run in a **dedicated Chrome profile**
* Must support:

  * Multiple tabs
  * Multiple sites per session
* Must uniquely identify:

  * Each browser tab (stable for its lifetime)
  * Site (derived from URL hostname)

Instrumentation may use:

* Chrome DevTools Protocol (preferred)
* Playwright or Puppeteer in **headed, user-driven mode**
* Chrome extension + native host (acceptable)

---

## 5. Log Directory Structure (Required)

All logs are written relative to the tool’s working directory.

```
logs/
 └── <site>/
      └── <browser_tab_id>/
           └── session.log
```

### Naming Rules

* `<site>`:

  * Derived from `URL.hostname`
  * Sanitized (e.g., `localhost_3000`, `app.internal`)
* `<browser_tab_id>`:

  * Unique per tab
  * Must not be reused during the same session

---

## 6. Log File (`session.log`) Requirements

### Format

* UTF-8 encoded text
* Append-only
* One event per line
* **JSON Lines (JSONL) strongly preferred**

Each log entry must include:

```json
{
  "timestamp": "ISO-8601 UTC",
  "site": "string",
  "tab_id": "string",
  "event_type": "string",
  "data": { }
}
```

---

## 7. Event Types (Single Stream)

All events—network, console, errors, navigation—are written to **the same log file**, differentiated by `event_type`.

### 7.1 Page & Navigation Events

Event types:

* `page.open`
* `page.navigate`
* `page.reload`
* `page.close`
* `page.visibility_change`

Example:

```json
{
  "timestamp": "2026-01-25T22:01:11.203Z",
  "site": "chatgpt.com",
  "tab_id": "tab-4",
  "event_type": "page.navigate",
  "data": {
    "url": "https://chatgpt.com/",
    "referrer": "https://google.com/",
    "navigation_type": "user"
  }
}
```

---

### 7.2 Network Events

Event types:

* `network.request`
* `network.response`
* `network.failure`

Data fields:

* `request_id`
* `method`
* `url`
* `status` (if available)
* `resource_type`
* `duration_ms`
* `error` (if applicable)

---

### 7.3 Console Events

Event types:

* `console.log`
* `console.warn`
* `console.info`
* `console.debug`

Data fields:

* `message`
* `args`
* `source` (file / line / column if available)

---

### 7.4 Error Events

Event types:

* `error.runtime`
* `error.unhandled_promise`
* `error.browser`

Data fields:

* `message`
* `stack`
* `source`
* `line`
* `column`

---

### 7.5 Meta / Environment Events

Event types:

* `meta.tab_created`
* `meta.tab_closed`
* `meta.environment`

Fields may include:

* `user_agent`
* `viewport`
* `device_scale_factor`
* `start_time`
* `end_time`

---

## 8. Runtime Behavior

* Logs must be written **in near real-time**
* No batching delays that would block `tail -f`
* File handles remain open while the tab is active
* On tab close:

  * Final `meta.tab_closed` event is written
  * File is flushed and closed

---

## 9. Safety & Privacy

* By default:

  * Cookies
  * Authorization headers
  * Request bodies
    are **excluded**
* Config flags may allow explicit inclusion

---

## 10. Configuration

Configurable via CLI or config file:

* Output directory
* Enable/disable event categories
* Redaction rules
* Headed vs headless mode
* Optional AI control toggle

---

## 11. Success Criteria

The tool is successful if:

* A user can browse naturally without noticing the logger
* Each tab produces a **single chronological log**
* Logs are cleanly separable by site and tab
* Another process can ingest logs incrementally without preprocessing
