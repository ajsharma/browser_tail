package cdp

// DemoPageHTML is an embedded HTML page that generates various browser events
// for demonstrating and validating browser_tail functionality.
const DemoPageHTML = `<!DOCTYPE html>
<html>
<head>
    <title>browser_tail Demo</title>
    <style>
        * { box-sizing: border-box; }
        body {
            font-family: system-ui, -apple-system, sans-serif;
            padding: 40px;
            background: linear-gradient(135deg, #1a1a2e 0%, #16213e 100%);
            color: #eee;
            min-height: 100vh;
            margin: 0;
        }
        .container { max-width: 800px; margin: 0 auto; }
        h1 { color: #4ecca3; margin-bottom: 10px; }
        .subtitle { color: #888; margin-bottom: 30px; }
        .card {
            background: rgba(255,255,255,0.05);
            border-radius: 8px;
            padding: 20px;
            margin-bottom: 20px;
        }
        .card h2 { color: #4ecca3; margin-top: 0; font-size: 18px; }
        .log-entry {
            font-family: monospace;
            font-size: 13px;
            padding: 8px 12px;
            margin: 4px 0;
            border-radius: 4px;
            background: rgba(0,0,0,0.3);
        }
        .log { border-left: 3px solid #4ecca3; }
        .warn { border-left: 3px solid #f9ca24; }
        .error { border-left: 3px solid #eb4d4b; }
        .info { border-left: 3px solid #74b9ff; }
        .timestamp { color: #888; }
        #event-count {
            font-size: 48px;
            color: #4ecca3;
            font-weight: bold;
        }
        .instructions {
            background: rgba(78, 204, 163, 0.1);
            border: 1px solid rgba(78, 204, 163, 0.3);
            border-radius: 8px;
            padding: 20px;
            margin-top: 30px;
        }
        .instructions code {
            background: rgba(0,0,0,0.3);
            padding: 2px 6px;
            border-radius: 4px;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>browser_tail Demo</h1>
        <p class="subtitle">This page generates browser events for testing</p>

        <div class="card">
            <h2>Events Generated</h2>
            <div id="event-count">0</div>
            <p>console events sent to browser_tail</p>
        </div>

        <div class="card">
            <h2>Recent Activity</h2>
            <div id="activity"></div>
        </div>

        <div class="instructions">
            <h2>Verify browser_tail is working</h2>
            <p>Check your logs directory:</p>
            <p><code>tail -f logs/*/*/session.log | grep -E "(console|error)"</code></p>
            <p>You should see events from this demo page appearing in real-time.</p>
        </div>
    </div>

    <script>
        let eventCount = 0;
        const activity = document.getElementById('activity');
        const countEl = document.getElementById('event-count');

        function addActivity(type, message) {
            eventCount++;
            countEl.textContent = eventCount;

            const entry = document.createElement('div');
            entry.className = 'log-entry ' + type;
            const time = new Date().toLocaleTimeString();
            entry.innerHTML = '<span class="timestamp">[' + time + ']</span> ' + message;

            activity.insertBefore(entry, activity.firstChild);

            // Keep only last 10 entries
            while (activity.children.length > 10) {
                activity.removeChild(activity.lastChild);
            }
        }

        // Initial burst of events
        console.log('[demo] Page loaded successfully');
        addActivity('log', 'console.log: Page loaded');

        console.info('[demo] browser_tail demo is running');
        addActivity('info', 'console.info: Demo running');

        console.warn('[demo] This is a warning message');
        addActivity('warn', 'console.warn: Warning message');

        console.error('[demo] This is an error message');
        addActivity('error', 'console.error: Error message');

        // Intentional global error after a short delay
        setTimeout(() => {
            addActivity('error', 'Throwing uncaught error...');
            throw new Error('[demo] Intentional uncaught error for testing');
        }, 2000);

        // Periodic events
        let iteration = 0;
        setInterval(() => {
            iteration++;
            const types = ['log', 'info', 'warn'];
            const type = types[iteration % 3];
            const message = '[demo] Periodic event #' + iteration + ' at ' + new Date().toISOString();

            if (type === 'log') {
                console.log(message);
                addActivity('log', 'console.log: Periodic #' + iteration);
            } else if (type === 'info') {
                console.info(message);
                addActivity('info', 'console.info: Periodic #' + iteration);
            } else {
                console.warn(message);
                addActivity('warn', 'console.warn: Periodic #' + iteration);
            }
        }, 5000);

        // Simulate a network request
        setTimeout(() => {
            console.log('[demo] Making fetch request...');
            addActivity('log', 'Making fetch request to example.com');
            fetch('https://httpbin.org/json')
                .then(r => r.json())
                .then(data => {
                    console.log('[demo] Fetch successful:', JSON.stringify(data).slice(0, 50) + '...');
                    addActivity('log', 'Fetch completed successfully');
                })
                .catch(err => {
                    console.error('[demo] Fetch failed:', err.message);
                    addActivity('error', 'Fetch failed: ' + err.message);
                });
        }, 3000);
    </script>
</body>
</html>`
