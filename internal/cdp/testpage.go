package cdp

// TestPageHTML is embedded HTML served as the anchor tab content.
// This replaces the blank about:blank page with a useful status page
// that shows browser_tail is active and generates console events for testing.
const TestPageHTML = `<!DOCTYPE html>
<html>
<head>
    <title>browser_tail - Monitoring Active</title>
    <style>
        body { font-family: system-ui; padding: 40px; background: #1a1a2e; color: #eee; }
        .status { color: #4ecca3; font-size: 24px; }
        .info { color: #888; margin-top: 20px; }
    </style>
</head>
<body>
    <h1 class="status">browser_tail is monitoring this browser</h1>
    <p class="info">This tab is used internally for CDP connection.</p>
    <p class="info">You can close this tab.</p>
    <script>
        console.log('[browser_tail] Test page loaded');
        console.info('[browser_tail] Console logging is working');
        console.warn('[browser_tail] Warning test');

        // Periodic heartbeat for testing
        setInterval(() => {
            console.log('[browser_tail] heartbeat:', new Date().toISOString());
        }, 30000);
    </script>
</body>
</html>`
