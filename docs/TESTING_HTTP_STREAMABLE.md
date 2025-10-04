# Testing HTTP Streamable Transport

## Overview

This guide shows how to test the MCP 2025-06-18 HTTP Streamable transport implementation.

## Quick Test with curl

### 1. Start the Server

```bash
# Terminal 1: Start server
make run-stream

# Or with custom config
./mcpserver --env=.env.fs
```

The server should start on `http://localhost:3030/mcp`

### 2. Test Health Endpoint

```bash
curl http://localhost:3030/health
# Expected: OK

curl http://localhost:3030/ready
# Expected: READY
```

### 3. Open SSE Stream (GET Request)

In a new terminal, open an SSE connection:

```bash
# Terminal 2: Open SSE stream
curl -N \
  -H "Accept: text/event-stream" \
  -H "MCP-Protocol-Version: 2025-06-18" \
  http://localhost:3030/mcp
```

You should see an SSE event with a session endpoint:
```
event: endpoint
data: /mcp?sessionId=abc123xyz...
```

Keep this terminal open! The connection streams events from the server.

### 4. Initialize MCP Session (Required!)

**IMPORTANT:** MCP requires initialization before any tools can be used.

```bash
# Terminal 3: Initialize the session
curl -X POST http://localhost:3030/mcp?sessionId=YOUR_SESSION_ID \
  -H "Content-Type: application/json" \
  -H "MCP-Protocol-Version: 2025-06-18" \
  -d '{
    "jsonrpc": "2.0",
    "id": 0,
    "method": "initialize",
    "params": {
      "protocolVersion": "2025-06-18",
      "capabilities": {
        "tools": {},
        "resources": {},
        "prompts": {}
      },
      "clientInfo": {
        "name": "test-client",
        "version": "1.0.0"
      }
    }
  }'

# Then send initialized notification
curl -X POST http://localhost:3030/mcp?sessionId=YOUR_SESSION_ID \
  -H "Content-Type: application/json" \
  -H "MCP-Protocol-Version: 2025-06-18" \
  -d '{
    "jsonrpc": "2.0",
    "method": "notifications/initialized"
  }'
```

### 5. Send JSON-RPC Requests (POST)

Now you can use tools:

```bash
# Terminal 3: Send tools/list request
curl -X POST http://localhost:3030/mcp?sessionId=YOUR_SESSION_ID \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  -H "MCP-Protocol-Version: 2025-06-18" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/list"
  }'
```

**Note:** Replace `YOUR_SESSION_ID` with the actual session ID from the SSE event.

The response will appear in **Terminal 2** (the SSE stream) as an event:

```
event: message
data: {"jsonrpc":"2.0","id":1,"result":{"tools":[...]}}
```

## Test with Authentication

If you have authentication enabled:

```bash
# Terminal 2: Open SSE stream with API key
curl -N \
  -H "Accept: text/event-stream" \
  -H "MCP-Protocol-Version: 2025-06-18" \
  -H "X-API-Key: your-api-key" \
  http://localhost:3030/mcp

# Terminal 3: POST with API key
curl -X POST http://localhost:3030/mcp?sessionId=YOUR_SESSION_ID \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key" \
  -H "MCP-Protocol-Version: 2025-06-18" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/list"
  }'
```

## Complete Workflow Example

### Upload Content

```bash
# 1. Base64 encode some data
echo "Hello, World!" | base64
# Output: SGVsbG8sIFdvcmxkIQo=

# 2. Upload content
curl -X POST http://localhost:3030/mcp?sessionId=YOUR_SESSION_ID \
  -H "Content-Type: application/json" \
  -H "MCP-Protocol-Version: 2025-06-18" \
  -d '{
    "jsonrpc": "2.0",
    "id": 2,
    "method": "tools/call",
    "params": {
      "name": "upload_content",
      "arguments": {
        "owner_id": "550e8400-e29b-41d4-a716-446655440000",
        "name": "test.txt",
        "data": "SGVsbG8sIFdvcmxkIQo=",
        "tags": ["test", "example"]
      }
    }
  }'
```

### List Content

```bash
curl -X POST http://localhost:3030/mcp?sessionId=YOUR_SESSION_ID \
  -H "Content-Type: application/json" \
  -H "MCP-Protocol-Version: 2025-06-18" \
  -d '{
    "jsonrpc": "2.0",
    "id": 3,
    "method": "tools/call",
    "params": {
      "name": "list_content",
      "arguments": {
        "owner_id": "550e8400-e29b-41d4-a716-446655440000",
        "limit": 10
      }
    }
  }'
```

### Download Content

```bash
# Use content ID from upload response
curl -X POST http://localhost:3030/mcp?sessionId=YOUR_SESSION_ID \
  -H "Content-Type: application/json" \
  -H "MCP-Protocol-Version: 2025-06-18" \
  -d '{
    "jsonrpc": "2.0",
    "id": 4,
    "method": "tools/call",
    "params": {
      "name": "download_content",
      "arguments": {
        "id": "YOUR_CONTENT_ID"
      }
    }
  }'
```

## JavaScript/Browser Test

Create an HTML file to test from a browser:

```html
<!DOCTYPE html>
<html>
<head>
    <title>MCP HTTP Streamable Test</title>
</head>
<body>
    <h1>MCP HTTP Streamable Transport Test</h1>
    <div id="status">Connecting...</div>
    <div id="output"></div>

    <script>
        const sessionId = crypto.randomUUID();
        let sessionEndpoint = null;

        // Open SSE connection
        const eventSource = new EventSource('http://localhost:3030/mcp');

        eventSource.addEventListener('endpoint', (event) => {
            sessionEndpoint = event.data;
            document.getElementById('status').textContent =
                'Connected! Session: ' + sessionEndpoint;

            // Test: List tools
            testToolsList();
        });

        eventSource.addEventListener('message', (event) => {
            const response = JSON.parse(event.data);
            document.getElementById('output').innerHTML +=
                '<pre>' + JSON.stringify(response, null, 2) + '</pre>';
        });

        eventSource.onerror = (error) => {
            document.getElementById('status').textContent = 'Error: ' + error;
        };

        async function testToolsList() {
            const response = await fetch(`http://localhost:3030${sessionEndpoint}`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'MCP-Protocol-Version': '2025-06-18'
                },
                body: JSON.stringify({
                    jsonrpc: '2.0',
                    id: 1,
                    method: 'tools/list'
                })
            });

            // Response comes via SSE, not here
            console.log('Request sent, waiting for SSE response...');
        }
    </script>
</body>
</html>
```

Save as `test.html` and open in a browser. You'll need to handle CORS if testing from a different origin.

## Go Client Example

Create a simple Go client:

```go
package main

import (
    "bufio"
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "strings"
)

func main() {
    // Open SSE stream
    req, _ := http.NewRequest("GET", "http://localhost:3030/mcp", nil)
    req.Header.Set("Accept", "text/event-stream")
    req.Header.Set("MCP-Protocol-Version", "2025-06-18")

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        panic(err)
    }
    defer resp.Body.Close()

    // Read SSE events
    reader := bufio.NewReader(resp.Body)
    var sessionEndpoint string

    // Read first event (endpoint)
    for {
        line, err := reader.ReadString('\n')
        if err != nil {
            panic(err)
        }

        if strings.HasPrefix(line, "data: ") {
            sessionEndpoint = strings.TrimSpace(strings.TrimPrefix(line, "data: "))
            fmt.Println("Session endpoint:", sessionEndpoint)
            break
        }
    }

    // Send tools/list request in separate goroutine
    go func() {
        reqBody := map[string]interface{}{
            "jsonrpc": "2.0",
            "id":      1,
            "method":  "tools/list",
        }

        jsonData, _ := json.Marshal(reqBody)

        postReq, _ := http.NewRequest("POST",
            "http://localhost:3030"+sessionEndpoint,
            bytes.NewBuffer(jsonData))
        postReq.Header.Set("Content-Type", "application/json")
        postReq.Header.Set("MCP-Protocol-Version", "2025-06-18")

        postResp, err := client.Do(postReq)
        if err != nil {
            fmt.Println("POST error:", err)
            return
        }
        defer postResp.Body.Close()

        body, _ := io.ReadAll(postResp.Body)
        fmt.Println("POST response:", string(body))
    }()

    // Continue reading SSE events
    for {
        line, err := reader.ReadString('\n')
        if err != nil {
            break
        }

        if strings.HasPrefix(line, "data: ") {
            data := strings.TrimSpace(strings.TrimPrefix(line, "data: "))
            fmt.Println("SSE message:", data)
        }
    }
}
```

Run with:
```bash
go run test_client.go
```

## Automated Testing Script

Create a test script:

```bash
#!/bin/bash
# test_streamable.sh

set -e

PORT=3030
BASE_URL="http://localhost:$PORT"

echo "Testing HTTP Streamable Transport"
echo "=================================="

# 1. Health check
echo -n "1. Health check... "
curl -s $BASE_URL/health | grep -q "OK" && echo "✓" || echo "✗"

# 2. Ready check
echo -n "2. Ready check... "
curl -s $BASE_URL/ready | grep -q "READY" && echo "✓" || echo "✗"

# 3. Protocol version validation
echo -n "3. Invalid protocol version (should return 400)... "
STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
    -H "MCP-Protocol-Version: 1999-01-01" \
    -H "Accept: text/event-stream" \
    $BASE_URL/mcp)
[ "$STATUS" = "400" ] && echo "✓" || echo "✗ (got $STATUS)"

# 4. Open SSE connection and get session
echo "4. Opening SSE connection..."
timeout 2 curl -N \
    -H "Accept: text/event-stream" \
    -H "MCP-Protocol-Version: 2025-06-18" \
    $BASE_URL/mcp 2>/dev/null | head -5

echo ""
echo "Manual test required for full workflow:"
echo "1. Open SSE stream in one terminal"
echo "2. Send POST requests in another terminal"
echo "3. Observe responses in SSE stream"
```

Run with:
```bash
chmod +x test_streamable.sh
./test_streamable.sh
```

## Common Issues

### Issue: Connection Refused
```
curl: (7) Failed to connect to localhost port 3030
```
**Solution:** Make sure server is running with `make run-stream`

### Issue: 401 Unauthorized
```
401 Unauthorized: API key required
```
**Solution:** Add `-H "X-API-Key: your-key"` to curl commands

### Issue: Session ID Not Found
```
404 page not found
```
**Solution:** Make sure to use the exact session ID from the SSE endpoint event

### Issue: CORS Error (Browser)
```
Access to fetch blocked by CORS policy
```
**Solution:** Add CORS headers or use a reverse proxy (nginx, Caddy)

## Next Steps

- Try uploading files with actual file data
- Test batch operations
- Test resource endpoints
- Test prompt endpoints
- Monitor server logs for debugging

## See Also

- [HTTP Streamable Transport Guide](HTTP_STREAMABLE_TRANSPORT.md)
- [MCP 2025-06-18 Compliance](MCP_2025_06_18_COMPLIANCE.md)
- [Authentication Guide](AUTHENTICATION.md)
