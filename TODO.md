# Browser Tail - Implementation TODO

**Status Legend:**
- ⬜ Not started
- 🟨 In progress
- ✅ Complete

---

## Phase 1: Foundation & Basic Logging

### Setup
- ⬜ Initialize Go module (`go mod init github.com/ajsharma/browser_tail`)
- ⬜ Install dependencies (chromedp, cobra, etc.)
- ⬜ Create project directory structure
- ⬜ Setup linting tools (golangci-lint, gosec)
- ⬜ Create `scripts/verify.sh` verification script

### Core Components
- ⬜ `internal/cdp/manager.go` - CDP Manager
  - ⬜ Chrome connection via RemoteAllocator
  - ⬜ Initial tab discovery (query /json once)
  - ⬜ Event-driven monitoring (Target.setDiscoverTargets)
  - ⬜ Target event handlers (created/destroyed/infoChanged)
- ⬜ `internal/cdp/launcher.go` - Chrome auto-launch
  - ⬜ Launch Chrome with --remote-debugging-port
  - ⬜ Create temp user-data-dir
- ⬜ `internal/cdp/discovery.go` - Tab discovery
  - ⬜ Query /json endpoint
  - ⬜ Parse target list
  - ⬜ Filter page targets
- ⬜ `internal/monitor/tab_monitor.go` - Tab monitoring
  - ⬜ Create chromedp.Context per tab
  - ⬜ Enable CDP domains (page, network, runtime, log)
  - ⬜ Event listeners for page events
  - ⬜ Handle tab close gracefully
- ⬜ `internal/logger/file_manager.go` - File management
  - ⬜ Line-buffered I/O with 8 KB buffer
  - ⬜ Smart flush strategy (meta events, buffer full, timer)
  - ⬜ Create directory structure: logs/<site>/<tab_id>/
  - ⬜ Handle tab close (flush, sync, close)
- ⬜ `internal/logger/path.go` - Path utilities
  - ⬜ Site name sanitization
  - ⬜ Tab ID generation (counter-based)
  - ⬜ Session ID generation (UUID)
- ⬜ `internal/events/types.go` - Event types
  - ⬜ LogEvent struct
  - ⬜ Meta event types
- ⬜ `cmd/browser_tail/main.go` - CLI
  - ⬜ Cobra command setup
  - ⬜ Basic flags (--port, --output, --launch)
  - ⬜ Session ID generation
  - ⬜ Log meta.session_start event

### Testing & Verification
- ⬜ Unit tests for logger/path.go (>80% coverage)
- ⬜ Unit tests for logger/file_manager.go (>80% coverage)
- ⬜ Functional test: Launch Chrome, navigate, verify logs
- ⬜ Performance test: Verify buffered I/O (syscall rate)
- ⬜ Verify tail latency <100ms
- ⬜ Run linter (must pass)
- ⬜ Run gosec (no HIGH issues)

### Documentation
- ⬜ README.md - Installation, basic usage
- ⬜ ARCHITECTURE.md - Phase 1 components
- ⬜ Package godocs for all packages

### Git
- ⬜ Commit Phase 1 with full test/lint status

---

## Phase 2: All Event Types + AI Control

### Event Transformers
- ⬜ `internal/events/transformer.go` - Base transformer
- ⬜ `internal/events/network.go` - Network events
  - ⬜ Request tracking by requestID
  - ⬜ Request/response correlation
  - ⬜ Duration calculation
  - ⬜ Body capture (after loadingFinished)
- ⬜ `internal/events/console.go` - Console events
- ⬜ `internal/events/error.go` - Error events
- ⬜ `internal/events/page.go` - Page lifecycle events

### Body Capture
- ⬜ RequestTracker implementation
- ⬜ shouldCaptureBody logic (content-type, size filters)
- ⬜ Capture after loadingFinished (preferred)
- ⬜ Fallback to responseReceived for small responses (<4KB)
- ⬜ Handle base64 encoding
- ⬜ Log as network.response_body event

### AI Control Mode
- ⬜ `internal/control/controller.go` - Control interface
- ⬜ `internal/control/actions.go` - High-level actions
  - ⬜ Navigate(url)
  - ⬜ Click(selector)
  - ⬜ Type(selector, text)
  - ⬜ Evaluate(js)
  - ⬜ WaitForSelector(selector, timeout)
- ⬜ Add --control CLI flag

### Testing & Verification
- ⬜ Unit tests for events/ package (>80% coverage)
- ⬜ Unit tests for control/ package (>80% coverage)
- ⬜ Functional test: Verify all event types logged
- ⬜ Functional test: AI control mode
- ⬜ Integration test: Full session with AI control
- ⬜ Run linter (must pass)
- ⬜ Run gosec (no HIGH issues)

### Documentation
- ⬜ README.md - AI control mode documentation
- ⬜ CONTROL.md - Control API reference
- ⬜ Example logs for each event type
- ⬜ Package godocs

### Git
- ⬜ Commit Phase 2 with full test/lint status

---

## Phase 3: Privacy & Configuration

### Redaction System
- ⬜ `internal/redact/redactor.go` - Core redaction
- ⬜ `internal/redact/patterns.go` - Default denylists
  - ⬜ Header denylist (cookie, authorization, etc.)
  - ⬜ Body field denylist (password, token, etc.)
- ⬜ Header redaction logic
- ⬜ Body redaction (JSON field scanning)
- ⬜ Configurable via CLI flags

### Configuration
- ⬜ `internal/config/config.go` - Config management
  - ⬜ CLI flag parsing
  - ⬜ Config file support (YAML)
  - ⬜ Flag validation
  - ⬜ Default values
- ⬜ Add config flags:
  - ⬜ --redact / --no-redact
  - ⬜ --capture-bodies
  - ⬜ --body-size-limit
  - ⬜ --body-content-types
  - ⬜ --flush-interval
  - ⬜ --buffer-size
  - ⬜ --no-network / --no-console / --no-errors
  - ⬜ --config <file>
  - ⬜ --version

### Testing & Verification
- ⬜ Unit tests for redact/ package (>90% coverage)
- ⬜ Unit tests for config/ package (>85% coverage)
- ⬜ Functional test: Redaction enabled (verify [REDACTED])
- ⬜ Functional test: Redaction disabled
- ⬜ Functional test: Event filtering
- ⬜ Functional test: Config file
- ⬜ Error handling tests (invalid config, invalid port, permission denied)
- ⬜ Performance test: <100MB memory, <10% CPU
- ⬜ Run linter (must pass)
- ⬜ Run gosec (no HIGH issues)

### Documentation
- ⬜ README.md - Complete usage guide
- ⬜ CONFIGURATION.md - Config file format
- ⬜ PRIVACY.md - Redaction documentation
- ⬜ examples/config.yaml - Example config
- ⬜ FAQ section in README
- ⬜ Package godocs

### Git
- ⬜ Commit Phase 3 with full test/lint status

---

## Phase 4: Production Polish

### Error Handling & Recovery
- ⬜ Graceful Chrome disconnect handling
- ⬜ Disk full error handling
- ⬜ Permission error handling
- ⬜ Tab crash handling
- ⬜ Panic recovery in goroutines
- ⬜ Structured error logging
- ⬜ Clean shutdown on SIGINT/SIGTERM

### Comprehensive Testing
- ⬜ Unit tests for all packages (>80% coverage each)
- ⬜ Integration test: Full session with AI control
- ⬜ Chaos test: Chrome crashes during monitoring
- ⬜ Chaos test: Disk full simulation
- ⬜ Chaos test: Permission denied
- ⬜ Chaos test: Tab crash (chrome://crash)
- ⬜ Chaos test: Network interruption
- ⬜ Load test: 20 tabs, high network activity
- ⬜ Load test: Verify <200MB memory, <15% CPU
- ⬜ Load test: No goroutine leaks (pprof)
- ⬜ Load test: No file descriptor leaks (lsof)
- ⬜ Manual testing: 5 comprehensive scenarios
- ⬜ Manual testing: Document results

### Performance Profiling
- ⬜ CPU profiling (no obvious bottlenecks)
- ⬜ Memory profiling (no memory leaks)
- ⬜ Add profiling endpoints (optional)

### Cross-Platform
- ⬜ Build for Linux amd64
- ⬜ Build for macOS amd64
- ⬜ Build for macOS arm64
- ⬜ Build for Windows amd64
- ⬜ Test each binary on target platforms

### Documentation
- ⬜ README.md - Complete with all features
- ⬜ ARCHITECTURE.md - Full system design
- ⬜ CONFIGURATION.md - All options explained
- ⬜ PRIVACY.md - Privacy considerations
- ⬜ CONTROL.md - AI control API
- ⬜ TROUBLESHOOTING.md - Common issues
- ⬜ CONTRIBUTING.md - Development setup
- ⬜ CHANGELOG.md - Version history
- ⬜ LICENSE - Choose appropriate license
- ⬜ Verify all code has godocs
- ⬜ examples/ directory with sample logs and configs
- ⬜ scripts/ directory with helper scripts

### CI/CD
- ⬜ .github/workflows/ci.yml
  - ⬜ Run on push and PR
  - ⬜ Test on Linux, macOS, Windows
  - ⬜ Run linters
  - ⬜ Run all tests
  - ⬜ Check coverage >85%
  - ⬜ Build binaries
  - ⬜ Run security scan
- ⬜ .github/workflows/release.yml
  - ⬜ Trigger on tag push (v*.*.*)
  - ⬜ Cross-compile binaries
  - ⬜ Create GitHub release
  - ⬜ Upload binaries
  - ⬜ Generate changelog

### Security Audit
- ⬜ Review: No secrets in logs (unless --no-redact)
- ⬜ Review: No arbitrary code execution vulnerabilities
- ⬜ Review: Safe file path handling (no directory traversal)
- ⬜ Review: Safe Chrome process spawning (no shell injection)
- ⬜ Review: Dependencies up to date
- ⬜ Document security model in SECURITY.md
- ⬜ Run govulncheck (no vulnerabilities)
- ⬜ Run gosec (no HIGH or MEDIUM issues)

### User Acceptance Testing
- ⬜ Developer testing (advanced usage)
- ⬜ QA tester testing (edge cases)
- ⬜ Non-technical user testing (basic usage)
- ⬜ Collect feedback on installation, docs, performance
- ⬜ Address critical feedback

### Release Preparation
- ⬜ All tests passing
- ⬜ All documentation complete
- ⬜ CI/CD working
- ⬜ Binaries for all platforms
- ⬜ CHANGELOG.md updated
- ⬜ Installation instructions tested
- ⬜ Tag v1.0.0 release
- ⬜ Push to GitHub
- ⬜ Create GitHub release
- ⬜ Upload binaries

### Git
- ⬜ Commit Phase 4 with full status

---

## Ongoing / Maintenance

- ⬜ Monitor GitHub issues
- ⬜ Respond to user feedback
- ⬜ Security updates for dependencies
- ⬜ Performance optimizations
- ⬜ Consider future enhancements (log rotation, WebSocket streaming, etc.)

---

**Last Updated:** 2026-01-25
**Current Phase:** Phase 1 (Not Started)
