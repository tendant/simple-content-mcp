# HTTP Streamable Transport Guide (MCP 2025-06-18)

## Overview

The simple-content-mcp server implements **HTTP Streamable transport** compliant with **MCP Specification 2025-06-18**. This transport uses Server-Sent Events (SSE) for server-to-client streaming while supporting standard HTTP for client-to-server communication. The implementation uses the official MCP Go SDK's `SSEHandler` with additional middleware for 2025-06-18 spec compliance.

## What is HTTP Streamable Transport?

HTTP Streamable is the modern MCP transport protocol that combines:

- **Server → Client**: SSE (Server-Sent Events) for streaming responses and notifications
- **Client → Server**: HTTP POST with JSON-RPC messages to the `/mcp` endpoint

The key feature is a **single MCP endpoint** (`/mcp`) that supports both:
- **GET**: Opens an SSE stream for receiving server messages
- **POST**: Sends JSON-RPC requests/responses (server can reply with JSON or SSE stream)

This creates a bidirectional communication channel suitable for web applications, with better session management and flexibility than the older SSE-only transport.

## Starting the HTTP Streamable Server

### Basic Usage

```bash
# Start HTTP Streamable server on default port (8080)
./mcpserver --mode=sse

# Custom port
./mcpserver --mode=sse --port=3000

# With .env file
MCP_MODE=sse
MCP_PORT=3000
./mcpserver
```

**Note**: The mode is still called `sse` for backwards compatibility, but it implements the HTTP Streamable transport from MCP 2025-06-18 spec.

### With Authentication

```bash
# Using .env file
cat > .env << EOF
MCP_MODE=sse
MCP_PORT=8080
MCP_AUTH_ENABLED=true
MCP_API_KEY_1=your-key:550e8400-e29b-41d4-a716-446655440000::
EOF

./mcpserver
```

## Connecting Clients

### MCP Client Connection

The HTTP Streamable protocol works as follows:

1. **Client opens stream**: GET request to `/mcp` with `Accept: text/event-stream`
2. **Server responds**: SSE stream with session endpoint (e.g., `/mcp?sessionid=ABC123XYZ`)
3. **Client sends messages**: POST to session endpoint with JSON-RPC messages
4. **Server responds**: Either JSON response or SSE events (depending on message type)

### Example with curl

```bash
# 1. Open SSE stream (GET to /mcp endpoint)
curl -N -H "Accept: text/event-stream" http://localhost:8080/mcp

# Server responds with SSE events:
# event: endpoint
# data: /mcp?sessionid=ABC123XYZ

# 2. In another terminal, POST JSON-RPC to the session endpoint
curl -X POST http://localhost:8080/mcp?sessionid=ABC123XYZ \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/list"
  }'
```

### With MCP 2025-06-18 Headers

```bash
# Include MCP protocol version (required for 2025-06-18 compliance)
curl -N -H "MCP-Protocol-Version: 2025-06-18" \
     -H "Accept: text/event-stream" \
     http://localhost:8080/mcp

# With session ID (for resumable sessions)
curl -N -H "MCP-Protocol-Version: 2025-06-18" \
     -H "Mcp-Session-Id: abc123def456" \
     -H "Accept: text/event-stream" \
     http://localhost:8080/mcp
```

### With Authentication

```bash
# Include API key and protocol version
curl -N -H "X-API-Key: your-key" \
     -H "MCP-Protocol-Version: 2025-06-18" \
     -H "Accept: text/event-stream" \
     http://localhost:8080/mcp

# Or using Authorization header
curl -N -H "Authorization: Bearer your-key" \
     -H "MCP-Protocol-Version: 2025-06-18" \
     -H "Accept: text/event-stream" \
     http://localhost:8080/mcp
```

### JavaScript Client Example

```javascript
// Connect to MCP endpoint with HTTP Streamable transport
// For MCP 2025-06-18 compliance with custom headers, use fetch API

const sessionId = crypto.randomUUID();

// Open GET request to /mcp endpoint to establish SSE stream
const response = await fetch('http://localhost:8080/mcp', {
  method: 'GET',
  headers: {
    'MCP-Protocol-Version': '2025-06-18',
    'Mcp-Session-Id': sessionId,
    'X-API-Key': 'your-key',  // if auth enabled
    'Accept': 'text/event-stream'
  }
});

// Read SSE stream
const reader = response.body.getReader();
const decoder = new TextDecoder();

// Alternative: Use EventSource for browsers (without custom headers)
// Note: EventSource doesn't support custom headers
const eventSource = new EventSource('http://localhost:8080/mcp');

let sessionEndpoint = null;

// Listen for the endpoint event
eventSource.addEventListener('endpoint', (event) => {
  sessionEndpoint = event.data;
  console.log('Session endpoint:', sessionEndpoint);

  // Now we can send requests
  sendToolListRequest();
});

// Listen for message events (server responses)
eventSource.addEventListener('message', (event) => {
  const response = JSON.parse(event.data);
  console.log('Server response:', response);
});

// Send a request via POST
async function sendToolListRequest() {
  const response = await fetch(`http://localhost:8080${sessionEndpoint}`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-API-Key': 'your-key'  // if auth enabled
    },
    body: JSON.stringify({
      jsonrpc: '2.0',
      id: 1,
      method: 'tools/list'
    })
  });

  // Response will arrive via SSE message event
}

// Call a tool
async function uploadContent(ownerID, name, data) {
  await fetch(`http://localhost:8080${sessionEndpoint}`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-API-Key': 'your-key'
    },
    body: JSON.stringify({
      jsonrpc: '2.0',
      id: 2,
      method: 'tools/call',
      params: {
        name: 'upload_content',
        arguments: {
          owner_id: ownerID,
          name: name,
          data: data
        }
      }
    })
  });
}
```

## Endpoints

The HTTP Streamable server exposes the following endpoints:

### `/mcp` - MCP HTTP Streamable Endpoint
- **GET**: Opens SSE stream, server returns session endpoint via SSE event
- **POST**: Sends JSON-RPC messages to active session (via session endpoint with `?sessionid=...`)
- **Headers**:
  - `MCP-Protocol-Version`: Protocol version (defaults to `2025-03-26`)
  - `Mcp-Session-Id`: Optional session ID for resumable connections
  - `X-API-Key` or `Authorization`: API key (if auth enabled)
  - `Accept`: Should include `text/event-stream` for GET, `application/json` or `text/event-stream` for POST

### `/health` - Health Check
- **GET**: Returns "OK" if server is running
- **No authentication required**

### `/ready` - Readiness Check
- **GET**: Returns "READY" if server is ready to accept connections
- **No authentication required**

## MCP Protocol Over SSE

### Supported Methods

All 14 MCP tools are available over SSE:

**Content Management:**
- `tools/call` with `upload_content`
- `tools/call` with `get_content`
- `tools/call` with `get_content_details`
- `tools/call` with `list_content`
- `tools/call` with `download_content`
- `tools/call` with `update_content`
- `tools/call` with `delete_content`
- `tools/call` with `search_content`

**Derived Content:**
- `tools/call` with `list_derived_content`
- `tools/call` with `get_thumbnails`

**Status:**
- `tools/call` with `get_content_status`
- `tools/call` with `list_by_status`

**Batch Operations:**
- `tools/call` with `batch_upload`
- `tools/call` with `batch_get_details`

**Resources:**
- `resources/read` - Read content, schema, or stats resources

**Prompts:**
- `prompts/list` - List available prompts
- `prompts/get` - Get prompt template

### Request Format

All requests follow JSON-RPC 2.0 format:

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "upload_content",
    "arguments": {
      "owner_id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "example.txt",
      "data": "SGVsbG8gV29ybGQ=",
      "tags": ["example"]
    }
  }
}
```

### Response Format

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "{\"id\":\"...\",\"status\":\"uploaded\"}"
      }
    ]
  }
}
```

## Configuration

### Environment Variables

```bash
# SSE Transport
MCP_MODE=sse
MCP_HOST=0.0.0.0      # Listen on all interfaces
MCP_PORT=8080         # Server port
MCP_BASE_URL=http://localhost:8080  # Base URL for sessions

# Authentication
MCP_AUTH_ENABLED=true
MCP_API_KEY_1=key:owner_id::

# Features
MCP_ENABLE_RESOURCES=true
MCP_ENABLE_PROMPTS=true
```

### .env File

```bash
# Copy example
cp .env.example .env

# Edit for SSE
nano .env

# Set mode and port
MCP_MODE=sse
MCP_PORT=8080
```

## CORS (Cross-Origin Requests)

For web applications, you may need to configure CORS. Currently, CORS headers are not set. For production use, consider adding a reverse proxy (nginx, Caddy) with CORS configuration, or add CORS middleware to the server.

### Example nginx Configuration

```nginx
server {
    listen 443 ssl;
    server_name mcp.example.com;

    location / {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;

        # SSE specific
        proxy_set_header Connection '';
        proxy_buffering off;
        proxy_cache off;

        # CORS headers
        add_header Access-Control-Allow-Origin *;
        add_header Access-Control-Allow-Methods "GET, POST, OPTIONS";
        add_header Access-Control-Allow-Headers "Content-Type, X-API-Key, Authorization";
    }
}
```

## Performance Considerations

- **Connection Pooling**: Each SSE connection maintains an open HTTP connection
- **Timeouts**: SSE connections can stay open indefinitely
- **Scalability**: For high-scale deployments, consider:
  - Load balancing with sticky sessions
  - Connection limits per client
  - Redis for session management across instances

## Troubleshooting

### Connection Drops

If SSE connections drop frequently:
- Check network timeouts
- Ensure proxies support SSE (no buffering)
- Implement reconnection logic in client

### Authentication Errors

```
401 Unauthorized: API key required
```
- Ensure `X-API-Key` or `Authorization` header is sent
- Verify API key is valid and not expired
- Check `MCP_AUTH_ENABLED=true` in server config

### CORS Errors

If browser shows CORS errors:
- Add reverse proxy with CORS headers
- Or use server from same origin
- Or disable CORS in browser for development

## MCP 2025-06-18 Specification Compliance

The server implements the following features from the MCP 2025-06-18 specification:

### Protocol Version Negotiation
- **`MCP-Protocol-Version` header**: Clients should include this header to specify the protocol version
- **Default version**: If not specified, defaults to `2025-03-26`
- **Supported versions**: `2024-11-05`, `2025-03-26`, `2025-06-18`
- **Version validation**: Server returns 400 Bad Request for unsupported versions

### Session Management
- **`Mcp-Session-Id` header**: Clients can provide a session ID for resumable connections
- **Session tracking**: Session IDs are added to request context for downstream handlers
- **Cryptographically secure IDs**: Use UUIDs or similar secure identifiers

### Security Features
- **Origin validation**: Server validates `Origin` header for localhost servers
- **Authentication**: API key validation via `X-API-Key` or `Authorization` headers
- **Localhost binding**: Recommended to bind only to localhost for local development

### Backwards Compatibility
The implementation maintains backwards compatibility with older MCP specifications by:
- Accepting connections without protocol version headers (defaults to 2025-03-26)
- Supporting multiple protocol versions simultaneously
- Using the SDK's SSEHandler which implements the core 2024-11-05 spec

## Security Best Practices

1. **Use HTTPS** in production
2. **Validate API keys** on every request
3. **Set rate limits** per API key
4. **Monitor connections** for abuse
5. **Use secure session IDs** (SDK handles this)
6. **Implement timeouts** for inactive connections
7. **Validate Origin headers** for CORS security (implemented for 2025-06-18)
8. **Bind to localhost** for local development

## Example Web Application

See `examples/web-client/` (coming soon) for a complete web application that uses SSE transport to interact with the MCP server.

## Comparison: SSE vs stdio

| Feature | stdio | SSE |
|---------|-------|-----|
| Use Case | Local CLI tools | Web applications |
| Transport | stdin/stdout | HTTP |
| Authentication | Process-based | API keys |
| Scaling | Single process | Multi-client |
| Firewall | No network | Requires open port |
| CORS | N/A | May need proxy |

## Additional Documentation

- **[MCP 2025-06-18 Compliance Report](MCP_2025_06_18_COMPLIANCE.md)** - Detailed compliance verification
- **[Authentication Guide](AUTHENTICATION.md)** - API key authentication setup
- **[Main README](../README.md)** - Project overview and features

## Next Steps

- ✅ SSE transport fully implemented with MCP 2025-06-18 compliance
- ⏳ HTTP JSON-RPC transport (coming soon)
- ⏳ WebSocket transport (future)
- ⏳ Example web client application
