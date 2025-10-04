# Simple Content MCP Server

A Model Context Protocol (MCP) server for the [simple-content](https://github.com/tendant/simple-content) library, enabling AI agents to manage content through standardized MCP tools.

## Overview

This MCP server exposes all core content management operations from the `simple-content` library as MCP tools, allowing AI agents like Claude to upload, manage, search, and download content programmatically.

### Features

- ✅ **14 MCP Tools** - 8 core + 2 derived + 2 status + 2 batch operations
- ✅ **3 MCP Resources** - URI-addressable data (content, schema, stats)
- ✅ **4 MCP Prompts** - Workflow guidance templates
- ✅ **Batch Operations** - Upload/fetch multiple items in parallel
- ✅ **stdio Transport** for local agent use
- ✅ **In-memory Storage** for testing and development
- ✅ **Type-safe** with JSON Schema validation
- ✅ **Clean Adapter Pattern** - no modification to core library

## Architecture

```
┌─────────────────────────────────────────┐
│         AI Agent (Claude, etc.)          │
└───────────────┬─────────────────────────┘
                │ MCP Protocol
┌───────────────▼─────────────────────────┐
│      simple-content-mcp (This Repo)      │
│  ┌─────────────────────────────────┐   │
│  │  MCP Server (official SDK)      │   │
│  ├─────────────────────────────────┤   │
│  │  Tools    Resources    Prompts  │   │
│  └──────────────┬──────────────────┘   │
└─────────────────┼──────────────────────┘
                  │ Service Interface
┌─────────────────▼──────────────────────┐
│       simple-content (Core Library)     │
└─────────────────────────────────────────┘
```

## Quick Start

### Build the Server

```bash
go build -o mcpserver ./cmd/mcpserver
```

### Configuration with .env File

For easier testing and development, you can use a `.env` file:

```bash
# Copy the example configuration
cp .env.example .env

# Edit .env with your settings
# The server automatically loads .env on startup

# Run the server
./mcpserver
```

You can also specify a custom env file:

```bash
# Use a specific env file
./mcpserver --env=.env.production --mode=sse --port=8080
```

### Run the Server

```bash
# Run in stdio mode (default)
./mcpserver --mode=stdio

# Run with .env file (auto-loaded)
./mcpserver

# Run with custom env file
./mcpserver --env=.env.production

# Show version
./mcpserver --version
```

### Run the Example

```bash
cd examples/basic
go build -o example main.go
./example
```

## MCP Capabilities

The server provides:
- **14 Tools** - Actions for managing content (including batch operations)
- **3 Resources** - URI-addressable data for agents
- **4 Prompts** - Workflow guidance templates

### Tools

#### Content Management (8 tools)
1. **upload_content** - Upload content with data in a single operation
2. **get_content** - Retrieve content metadata by ID
3. **get_content_details** - Get complete information including URLs
4. **list_content** - List content with filtering and pagination
5. **download_content** - Download content (URL or base64)
6. **update_content** - Update content metadata
7. **delete_content** - Soft delete content
8. **search_content** - Search by metadata, tags, or query

#### Derived Content (2 tools)
9. **list_derived_content** - List derived content (thumbnails, previews) for a parent
10. **get_thumbnails** - Get thumbnails by size (convenience wrapper)

#### Status Monitoring (2 tools)
11. **get_content_status** - Check content processing status and derived content availability
12. **list_by_status** - List content by lifecycle status (for monitoring/workers)

#### Batch Operations (2 tools)
13. **batch_upload** - Upload multiple content items in parallel (up to MaxBatchSize)
14. **batch_get_details** - Get details for multiple content IDs in parallel

### Resources

Resources are URI-addressable data that agents can read:

1. **content://{id}** - Content metadata (template)
2. **schema://content** - JSON schema for Content entity
3. **stats://system** - System statistics and health

### Prompts

Prompts provide workflow guidance to agents:

1. **upload-workflow** - Step-by-step guide for uploading content
2. **search-workflow** - Guide for searching and filtering content
3. **derived-content-workflow** - Guide for working with thumbnails/previews
4. **status-monitoring** - Guide for monitoring content status

### Input Formats

All tools accept JSON arguments. Data can be provided as:
- **Base64 encoded**: `"data": "SGVsbG8gV29ybGQ="`
- **URL**: `"data": "https://example.com/file.pdf"`

### Example Tool Usage

```go
// Upload content
result, err := session.CallTool(ctx, &mcp.CallToolParams{
    Name: "upload_content",
    Arguments: map[string]interface{}{
        "owner_id":  ownerID.String(),
        "name":      "example.txt",
        "data":      base64EncodedData,
        "file_name": "example.txt",
        "tags":      []string{"example", "test"},
    },
})

// List content
result, err := session.CallTool(ctx, &mcp.CallToolParams{
    Name: "list_content",
    Arguments: map[string]interface{}{
        "owner_id": ownerID.String(),
        "limit":    10,
    },
})
```

## Development

### Requirements

- Go 1.25.1+
- MCP SDK v1.0.0
- simple-content v0.1.23

### Project Structure

```
simple-content-mcp/
├── cmd/mcpserver/          # Server entrypoint
│   ├── main.go
│   └── config.go
├── pkg/mcpserver/          # Core MCP server
│   ├── server.go           # Server wrapper
│   ├── config.go           # Configuration
│   ├── tools.go            # Tool registration
│   ├── handlers.go         # Tool handlers
│   └── errors/             # Error mapping
├── examples/
│   └── basic/              # Example client
└── tests/
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./pkg/mcpserver
```

### Configuration

The server can be configured via environment variables or programmatically.

### Environment Variables

```bash
# Transport settings
MCP_MODE=stdio              # stdio, sse, or http
MCP_HOST=localhost          # Host for SSE/HTTP modes
MCP_PORT=8080               # Port for SSE/HTTP modes
MCP_BASE_URL=http://localhost:8080  # Base URL for SSE mode

# Batch and pagination
MCP_MAX_BATCH_SIZE=100      # Maximum items in batch operations
MCP_DEFAULT_PAGE_SIZE=50    # Default page size for list operations
MCP_MAX_PAGE_SIZE=1000      # Maximum page size

# Features
MCP_ENABLE_RESOURCES=true   # Enable MCP resources
MCP_ENABLE_PROMPTS=true     # Enable MCP prompts

# Authentication (Phase 5)
MCP_AUTH_ENABLED=false      # Enable authentication
MCP_API_KEY_1=mykey:550e8400-e29b-41d4-a716-446655440000::  # API key with owner_id
MCP_API_KEY_2=anotherkey:660e8400-e29b-41d4-a716-446655440001::  # Additional keys

# Storage (future - requires additional dependencies)
DATABASE_URL=postgresql://user:pass@localhost/dbname
STORAGE_BACKEND=memory      # memory, s3, or filesystem
```

### Programmatic Configuration

```go
config := mcpserver.DefaultConfig(service)
config.Mode = mcpserver.TransportHTTP
config.Port = 8080
config.AuthEnabled = true
config.Authenticator = auth.NewAPIKeyAuthenticator()
```

## Implementation Status

### Phase 1 ✅ (Completed)
- [x] Core 8 tools
- [x] stdio transport
- [x] Error handling
- [x] Unit tests
- [x] Example client

### Phase 2 ✅ (Completed)
- [x] Derived content tools (list_derived_content, get_thumbnails)
- [x] Status monitoring tools (get_content_status, list_by_status)
- [x] Unit tests for new tools
- [x] Updated example demonstrations

### Phase 3 ✅ (Completed)
- [x] MCP Resources (content://{id}, schema://content, stats://system)
- [x] Resource templates with URI parameters
- [x] MCP Prompts (4 workflow guides)
- [x] Unit tests for resources and prompts
- [x] Updated example demonstrations

### Phase 4 ✅ (Completed)
- [x] Batch upload tool (parallel upload of multiple items)
- [x] Batch get details tool (parallel fetch of multiple IDs)
- [x] MaxBatchSize enforcement (config-based limits)
- [x] Graceful error handling (partial failures reported)
- [x] Unit tests for batch operations
- [x] Updated example demonstrations

### Phase 5 ✅ (Completed)
- [x] Authentication infrastructure (API key auth)
- [x] Auth middleware for tool handlers
- [x] **Full SSE transport** with MCP protocol support
- [x] HTTP transport with health/ready endpoints
- [x] Environment variable configuration
- [x] Dotenv (.env file) support
- [x] Production-ready server setup

### Future Enhancements
- [ ] Full HTTP JSON-RPC MCP protocol implementation
- [ ] PostgreSQL and S3 storage backends (require additional dependencies)
- [ ] OAuth integration
- [ ] Rate limiting and quotas
- [ ] WebSocket transport

## Dependencies

- [MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk) v1.0.0
- [simple-content](https://github.com/tendant/simple-content) v0.1.23
- [google/uuid](https://github.com/google/uuid) v1.6.0
- [godotenv](https://github.com/joho/godotenv) v1.5.1 - Environment configuration

## License

See [LICENSE](LICENSE) file for details.

## References

- [MCP Specification](https://modelcontextprotocol.io)
- [MCP Go SDK Documentation](https://github.com/modelcontextprotocol/go-sdk)
- [Simple Content Library](https://github.com/tendant/simple-content)
- [Full Implementation Plan](MCP_INTEGRATION_PLAN.md)
