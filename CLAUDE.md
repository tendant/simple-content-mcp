# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**simple-content-mcp** is a Model Context Protocol (MCP) adapter for the `simple-content` library. It exposes content management operations (upload, download, search, derived content) through standardized MCP tools, resources, and prompts, enabling AI agents to manage content programmatically.

**Key Design Principles:**
- **Adapter Pattern**: Wraps `github.com/tendant/simple-content` Service interface without modifying the core library
- **Thin Protocol Layer**: MCP server only handles protocol translation, contains no business logic
- **Type Safety**: Uses official MCP SDK types and JSON Schema validation

## Dependencies

```
simple-content-mcp/
├── github.com/modelcontextprotocol/go-sdk (official MCP SDK)
├── github.com/tendant/simple-content v0.1.23 (core library)
└── Go 1.25.1
```

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
│         Service Implementation          │
└─────────────────────────────────────────┘
```

## Planned Repository Structure

```
cmd/mcpserver/          # Server entrypoint and configuration
pkg/mcpserver/
  ├── server.go         # MCP server wrapper
  ├── tools/            # MCP tool implementations
  │   ├── content.go    # Core content tools (upload, get, list, etc.)
  │   ├── derived.go    # Derived content tools (thumbnails, previews)
  │   ├── search.go     # Search and filtering tools
  │   ├── status.go     # Status management tools
  │   └── batch.go      # Batch operations
  ├── resources/        # MCP resources (schemas, stats)
  ├── prompts/          # Workflow prompts for agents
  ├── auth/             # Authentication middleware
  └── errors/           # Error mapping to MCP error codes
```

## MCP Tools (Planned)

### Tool Naming Convention
- Lowercase with underscores: `upload_content`, `get_content_details`
- Group by domain: `content_*`, `derived_*`, `status_*`, `batch_*`
- Consistent verbs: get, list, create, update, delete, upload, download

### Core Content Tools (8 tools)
1. `upload_content` - Upload content with data in single operation
2. `get_content` - Retrieve content metadata by ID
3. `get_content_details` - Get complete info including URLs and metadata
4. `list_content` - List content with filtering and pagination
5. `download_content` - Download content data (URL or base64)
6. `update_content` - Update content metadata
7. `delete_content` - Soft delete content
8. `search_content` - Search by metadata, tags, or full-text

### Derived Content Tools (3 tools)
9. `create_derived_content` - Create derived content placeholder
10. `upload_derived_content` - Upload derived content (thumbnail, preview)
11. `list_derived_content` - List derived content with filtering

### Status Management Tools (3 tools)
12. `get_content_status` - Get content processing status
13. `update_content_status` - Update content lifecycle status
14. `list_by_status` - List all content by status

### Batch Operations (2 tools)
15. `batch_upload` - Upload multiple content items
16. `batch_get_details` - Get details for multiple content IDs

## MCP Resources (Planned)

Resources use URI-based addressing for discoverable context:

- `content://{content_id}` - Individual content metadata (application/json)
- `content://{content_id}/details` - Complete content details with URLs
- `derived://{parent_id}` - List of derived content for a parent
- `schema://content` - Content entity JSON schema
- `schema://derived-content` - Derived content relationship schema
- `backends://storage` - Available storage backends
- `stats://system` - System statistics and health

## MCP Prompts (Planned)

Workflow prompts to guide agents:

1. `upload-workflow` - Guide for uploading content
2. `thumbnail-generation` - Generate thumbnails for images
3. `async-processing` - Pattern for async content processing
4. `batch-upload` - Upload multiple files efficiently
5. `content-search` - Search and filter content

## Transport Modes

The MCP server will support multiple transport modes:

- **stdio**: Standard input/output (default, for local agent use)
- **SSE**: Server-Sent Events (for HTTP streaming)
- **HTTP**: Standard HTTP (planned)

## Key Implementation Details

### Data Encoding
Tools accept data in two formats:
- **Base64**: For direct binary data transfer (`"data": "iVBORw0KGgoAAAANS..."`)
- **URL**: For remote content (`"data": "https://example.com/file.pdf"`)

### Authentication (Planned)
- API key authentication (simple)
- OAuth integration via official SDK
- Owner/tenant scoping for multi-tenancy

### Error Handling
Map simple-content errors to MCP error codes:
- `-32600`: Invalid Request
- `-32602`: Invalid Params (validation error)
- `-32603`: Internal Error
- `40001`: Unauthorized
- `40003`: Forbidden
- `40004`: Not Found
- `50001`: Storage Error

## Development Commands

Since this is a new repository with minimal code:

```bash
# Install dependencies
go mod download

# Build server (once implemented)
go build -o mcpserver ./cmd/mcpserver

# Run tests (once implemented)
go test ./...

# Run in stdio mode (once implemented)
./mcpserver --mode=stdio
```

## Implementation Phases

Per MCP_INTEGRATION_PLAN.md:

1. **Phase 1** (Week 1): Foundation & core 8 tools
2. **Phase 2** (Week 2): Derived content & status tools
3. **Phase 3** (Week 3): Resources & prompts
4. **Phase 4** (Week 4): Batch operations
5. **Phase 5** (Week 5): Production hardening (auth, deployment, docs)

## Important Constraints

1. **Never modify the core library**: `github.com/tendant/simple-content` must remain unchanged
2. **Thin adapter only**: All business logic stays in the core library
3. **Use official SDK**: Only use `github.com/modelcontextprotocol/go-sdk`, not third-party alternatives
4. **JSON Schema validation**: All tool inputs/outputs must have proper JSON Schema (Draft 7)
5. **Context required**: All handlers must accept and pass context.Context

## Example Tool Handler Pattern

```go
func (s *Server) handleUploadContent(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    // 1. Parse and validate input
    params := req.Params.Arguments
    ownerID, err := parseUUID(params["owner_id"])
    if err != nil {
        return nil, newValidationError("owner_id", err)
    }

    // 2. Decode data (base64 or URL)
    reader, err := s.decodeData(params["data"])
    if err != nil {
        return nil, err
    }

    // 3. Call core service (no business logic here!)
    content, err := s.service.UploadContent(ctx, simplecontent.UploadContentRequest{...})
    if err != nil {
        return nil, s.mapError(err)
    }

    // 4. Format and return MCP result
    return mcp.NewToolResultText(formatJSON(content)), nil
}
```

## Testing Strategy

- **Unit tests**: Mock `simplecontent.Service` interface for tool handlers
- **Integration tests**: Use real service with memory backend
- **Example scripts**: Create agent example scripts in `examples/`

## References

- [MCP Specification](https://modelcontextprotocol.io/specification)
- [Official Go SDK](https://github.com/modelcontextprotocol/go-sdk)
- [Simple Content Library](https://github.com/tendant/simple-content)
- Full implementation plan: `MCP_INTEGRATION_PLAN.md`
