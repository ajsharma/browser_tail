# Browser Tail Implementation Plan

This plan contains the complete design and implementation strategy for browser_tail.

See the full detailed plan at: `.claude/plans/async-shimmying-whale.md`

## Quick Reference

**Project:** Go-based CLI tool for capturing Chrome browser activity to structured JSONL logs

**Key Technologies:**
- Go 1.21+
- Chrome DevTools Protocol (via chromedp)
- Event-driven architecture (no polling)
- Line-buffered I/O for performance

**4 Implementation Phases:**
1. **Phase 1:** Foundation & Basic Logging - Chrome connection, tab monitoring, page events
2. **Phase 2:** All Event Types + AI Control - Network, console, errors + browser automation
3. **Phase 3:** Privacy & Configuration - Redaction, config files, CLI enhancements
4. **Phase 4:** Production Polish - Testing, docs, cross-platform builds, CI/CD

## Critical Implementation Notes

### Performance
- Line-buffered batching: 8 KB buffer, 100ms flush interval
- Sync() only on meta events and shutdown
- Handles 1000s events/sec with <100ms tail latency

### Architecture
- Event-driven tab discovery (NO polling /json)
- CDP Target.setDiscoverTargets() for real-time events
- One chromedp.Context per tab (goroutine per tab)

### Body Capture
- Default: Headers + metadata only (NO bodies)
- Opt-in via `--capture-bodies` flag
- Capture after loadingFinished, fallback to responseReceived for <4KB responses
- Filter by content-type and size

### Lifecycle
- Manager owns Target.targetDestroyed events
- Manager signals TabMonitor shutdown via context.cancel()
- Only monitor targets: Type=="page" AND Attached==false

### Site Changes
- Tab navigates to different site → new log file
- Same tab_id, different site folder
- Example: `logs/example.com/tab-4/` → `logs/github.com/tab-4/`

### Session Tracking
- Generate session_id (UUID) once per process
- Tab IDs stable within process lifetime only
- Include session_id in all events for correlation

## Verification Workflow

Every code change must pass:
1. go fmt && go vet
2. golangci-lint (strictest settings)
3. gosec (security scan)
4. Unit tests (>80% coverage)
5. Integration tests
6. Manual functional test
7. Documentation update

Run `./scripts/verify.sh` before every commit.

## Success Criteria

- ✅ User browses naturally without noticing logger
- ✅ Each tab → single chronological JSONL log
- ✅ Real-time logging supports `tail -f`
- ✅ Privacy redaction enabled by default
- ✅ Single binary, no runtime dependencies
- ✅ Cross-platform (Linux, macOS, Windows)

See full plan for detailed implementation specs, code examples, and testing procedures.
