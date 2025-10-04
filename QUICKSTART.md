# Quick Start Guide

## 1. Build the Server

```bash
make build
```

## 2. Choose Your Mode

### Option A: Development (In-Memory)

```bash
make run-stream
```

Server starts on `http://localhost:3030/mcp`

### Option B: With Filesystem Storage

```bash
./mcpserver --env=.env.fs
```

Files stored in `./data/storage/`

### Option C: Production (PostgreSQL + Filesystem)

```bash
# 1. Create database
createdb simple_content

# 2. Configure .env.postgres
cp .env.postgres .env
# Edit DATABASE_URL in .env

# 3. Run
./mcpserver --env=.env.postgres
```

## 3. Test It

### Quick Health Check

```bash
curl http://localhost:3030/health
# Expected: OK
```

### Interactive Test

```bash
# In terminal 1: Start server
make run-stream

# In terminal 2: Run interactive test
make test-streamable
```

### Manual Test with curl

```bash
# Terminal 1: Open SSE stream
curl -N \
  -H "Accept: text/event-stream" \
  -H "MCP-Protocol-Version: 2025-06-18" \
  http://localhost:3030/mcp

# Note the session endpoint from output: /mcp?sessionId=...

# Terminal 2: Initialize session first (REQUIRED!)
curl -X POST http://localhost:3030/mcp?sessionId=SESSION_ID \
  -H "Content-Type: application/json" \
  -H "MCP-Protocol-Version: 2025-06-18" \
  -d '{
    "jsonrpc": "2.0",
    "id": 0,
    "method": "initialize",
    "params": {
      "protocolVersion": "2025-06-18",
      "capabilities": {},
      "clientInfo": {"name": "test-client", "version": "1.0"}
    }
  }'

# Send initialized notification
curl -X POST http://localhost:3030/mcp?sessionId=SESSION_ID \
  -H "Content-Type: application/json" \
  -H "MCP-Protocol-Version: 2025-06-18" \
  -d '{"jsonrpc":"2.0","method":"notifications/initialized"}'

# NOW you can use tools!
curl -X POST http://localhost:3030/mcp?sessionId=SESSION_ID \
  -H "Content-Type: application/json" \
  -H "MCP-Protocol-Version: 2025-06-18" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/list"
  }'

# Response appears in Terminal 1!
```

## 4. Upload Your First File

```bash
# Base64 encode your content
echo "Hello, World!" | base64
# Output: SGVsbG8sIFdvcmxkIQo=

# Upload (use your session ID)
curl -X POST http://localhost:3030/mcp?sessionId=YOUR_SESSION \
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
        "name": "hello.txt",
        "data": "SGVsbG8sIFdvcmxkIQo=",
        "tags": ["test"]
      }
    }
  }'
```

## Available Tools

Run `tools/list` to see all 14 available tools:

**Content Management:**
- `upload_content` - Upload with data
- `get_content` - Get metadata
- `list_content` - List with filters
- `download_content` - Download file
- `update_content` - Update metadata
- `delete_content` - Soft delete
- `search_content` - Search by query
- `get_content_details` - Full details

**Derived Content:**
- `list_derived_content` - List thumbnails/previews
- `get_thumbnails` - Get by size

**Status:**
- `get_content_status` - Check status
- `list_by_status` - List by lifecycle

**Batch:**
- `batch_upload` - Upload multiple
- `batch_get_details` - Get multiple details

## Configuration Files

- `.env.example` - Template with all options
- `.env.fs` - Filesystem storage only
- `.env.postgres` - PostgreSQL + filesystem
- `.env.test` - Test configuration

## Useful Commands

```bash
# Build
make build

# Run modes
make run-stdio      # Standard I/O (for MCP clients)
make run-stream     # HTTP Streamable (port 3030)
make run-http       # HTTP JSON-RPC (port 3030)

# Testing
make test           # Unit tests
make test-compliance    # MCP spec compliance
make test-streamable    # Interactive transport test

# Code quality
make fmt            # Format code
make vet            # Run go vet
make check          # All checks

# Help
make help           # Show all targets
```

## Next Steps

1. **Read the docs:**
   - [HTTP Streamable Transport](docs/HTTP_STREAMABLE_TRANSPORT.md)
   - [PostgreSQL Setup](docs/POSTGRESQL_SETUP.md)
   - [Testing Guide](docs/TESTING_HTTP_STREAMABLE.md)
   - [Authentication](docs/AUTHENTICATION.md)

2. **Try the example client:**
   ```bash
   make run-example
   ```

3. **Set up production:**
   - Configure PostgreSQL
   - Set up filesystem or S3 storage
   - Enable authentication
   - Configure reverse proxy (nginx/Caddy)

## Troubleshooting

**Server won't start:**
```bash
# Check if port is in use
lsof -i :3030

# Try different port
./mcpserver --mode=sse --port=8080
```

**Can't connect:**
```bash
# Check server is running
curl http://localhost:3030/health

# Check logs
./mcpserver --env=.env 2>&1 | tee server.log
```

**Database errors:**
```bash
# Test connection
psql $DATABASE_URL -c "SELECT 1"

# Check permissions
psql $DATABASE_URL -c "\dt"
```

## Support

- [GitHub Issues](https://github.com/tendant/simple-content-mcp/issues)
- [MCP Specification](https://modelcontextprotocol.io/specification/2025-06-18)
- [Full Documentation](README.md)
