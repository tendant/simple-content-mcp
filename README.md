# Simple Content MCP Server

A Model Context Protocol (MCP) server for the [simple-content](https://github.com/tendant/simple-content) library, enabling AI agents to manage content through standardized MCP tools.

## Overview

This MCP server exposes all core content management operations from the `simple-content` library as MCP tools, allowing AI agents like Claude to upload, manage, search, and download content programmatically.

### Features

- ✅ **8 Core MCP Tools** for content management
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

### Run the Server

```bash
# Run in stdio mode (default)
./mcpserver --mode=stdio

# Show version
./mcpserver --version
```

### Run the Example

```bash
cd examples/basic
go build -o example main.go
./example
```

## MCP Tools

The server provides 8 core tools:

### Content Management
1. **upload_content** - Upload content with data in a single operation
2. **get_content** - Retrieve content metadata by ID
3. **get_content_details** - Get complete information including URLs
4. **list_content** - List content with filtering and pagination
5. **download_content** - Download content (URL or base64)
6. **update_content** - Update content metadata
7. **delete_content** - Soft delete content
8. **search_content** - Search by metadata, tags, or query

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

The server uses sensible defaults:

```go
config := mcpserver.DefaultConfig(service)
// Name:            "simple-content-mcp"
// Version:         "0.1.0"
// Mode:            TransportStdio
// MaxBatchSize:    100
// DefaultPageSize: 50
// MaxPageSize:     1000
```

## Implementation Status

### Phase 1 ✅ (Completed)
- [x] Core 8 tools
- [x] stdio transport
- [x] Error handling
- [x] Unit tests
- [x] Example client

### Future Phases
- [ ] Phase 2: Derived content & status tools
- [ ] Phase 3: Resources & prompts
- [ ] Phase 4: Batch operations
- [ ] Phase 5: Authentication, SSE/HTTP transports

## Dependencies

- [MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk) v1.0.0
- [simple-content](https://github.com/tendant/simple-content) v0.1.23
- [google/uuid](https://github.com/google/uuid) v1.6.0

## License

See [LICENSE](LICENSE) file for details.

## References

- [MCP Specification](https://modelcontextprotocol.io)
- [MCP Go SDK Documentation](https://github.com/modelcontextprotocol/go-sdk)
- [Simple Content Library](https://github.com/tendant/simple-content)
- [Full Implementation Plan](MCP_INTEGRATION_PLAN.md)
