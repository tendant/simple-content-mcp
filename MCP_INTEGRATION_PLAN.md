# MCP Integration Plan for Simple Content

**Target Repository:** `github.com/tendant/simple-content-mcp` (separate library)
**Base Library:** `github.com/tendant/simple-content`
**MCP SDK:** `github.com/modelcontextprotocol/go-sdk`
**Version:** v1.0.0-alpha
**Date:** 2025-10-02

---

## Table of Contents

1. [Overview](#overview)
2. [Architecture](#architecture)
3. [Repository Structure](#repository-structure)
4. [MCP Tools Specification](#mcp-tools-specification)
5. [MCP Resources Specification](#mcp-resources-specification)
6. [MCP Prompts Specification](#mcp-prompts-specification)
7. [Implementation Phases](#implementation-phases)
8. [API Design](#api-design)
9. [Security & Authentication](#security--authentication)
10. [Testing Strategy](#testing-strategy)
11. [Deployment](#deployment)
12. [Migration Guide](#migration-guide)

---

## Overview

### Purpose

Create a Model Context Protocol (MCP) adapter for the `simple-content` library, enabling AI agents to manage content through standardized MCP tools, resources, and prompts.

### Goals

- ✅ Expose all core content operations as MCP tools
- ✅ Provide discoverable resources for schemas and statistics
- ✅ Offer workflow prompts for common tasks
- ✅ Support multiple deployment modes (stdio, SSE, HTTP)
- ✅ Enable secure, multi-tenant agent access
- ✅ Maintain clean separation from core library

### Non-Goals

- ❌ Modify the core `simple-content` library
- ❌ Implement custom MCP features beyond specification
- ❌ Replace the existing HTTP API server

---

## Architecture

### Design Principles

1. **Adapter Pattern**: Wrap `simplecontent.Service` interface without modification
2. **Thin Layer**: MCP server only handles protocol translation, no business logic
3. **Type Safety**: Use official MCP SDK types and schemas
4. **Extensibility**: Easy to add new tools as the core library evolves
5. **Security First**: Authentication and authorization built-in

### Component Diagram

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
│  ┌─────────────────────────────────┐   │
│  │  Service Implementation         │   │
│  ├─────────────────────────────────┤   │
│  │  Repository  │  BlobStore       │   │
│  └─────────────────────────────────┘   │
└─────────────────────────────────────────┘
```

### Dependency Tree

```
simple-content-mcp/
├── github.com/modelcontextprotocol/go-sdk
├── github.com/tendant/simple-content
└── Standard Library
```

---

## Repository Structure

```
simple-content-mcp/
├── go.mod
├── go.sum
├── README.md
├── LICENSE
├── .github/
│   └── workflows/
│       ├── ci.yml
│       └── release.yml
├── cmd/
│   └── mcpserver/
│       ├── main.go              # Server entrypoint
│       └── config.go            # Configuration
├── pkg/
│   └── mcpserver/
│       ├── server.go            # MCP server wrapper
│       ├── config.go            # Server configuration
│       ├── tools/
│       │   ├── content.go       # Content management tools
│       │   ├── derived.go       # Derived content tools
│       │   ├── search.go        # Search and list tools
│       │   ├── status.go        # Status management tools
│       │   └── batch.go         # Batch operations tools
│       ├── resources/
│       │   ├── resources.go     # Resource provider
│       │   ├── schemas.go       # Schema resources
│       │   └── stats.go         # Statistics resources
│       ├── prompts/
│       │   └── prompts.go       # Workflow prompts
│       ├── auth/
│       │   ├── middleware.go    # Authentication middleware
│       │   └── validator.go     # Token validation
│       └── errors/
│           └── errors.go        # Error mapping
├── examples/
│   ├── basic/
│   │   └── main.go              # Basic usage example
│   ├── agent-upload/
│   │   └── main.go              # Agent upload workflow
│   └── batch-processing/
│       └── main.go              # Batch operations
├── docs/
│   ├── TOOLS.md                 # Tool catalog
│   ├── RESOURCES.md             # Resource catalog
│   ├── PROMPTS.md               # Prompt catalog
│   ├── DEPLOYMENT.md            # Deployment guide
│   └── AGENT_GUIDE.md           # Agent integration guide
└── tests/
    ├── integration/
    │   └── server_test.go
    └── testdata/
        └── fixtures/
```

---

## MCP Tools Specification

### Tool Naming Convention

- Use lowercase with underscores: `upload_content`, `get_content_details`
- Group by domain: content_, derived_, status_, batch_
- Keep verbs consistent: get, list, create, update, delete, upload, download

### Core Content Tools

#### 1. `upload_content`

**Description:** Upload content with data in a single operation

**Input Schema:**
```json
{
  "type": "object",
  "properties": {
    "owner_id": {"type": "string", "format": "uuid"},
    "tenant_id": {"type": "string", "format": "uuid"},
    "name": {"type": "string"},
    "description": {"type": "string"},
    "document_type": {"type": "string"},
    "storage_backend": {"type": "string", "default": "default"},
    "data": {
      "oneOf": [
        {"type": "string", "description": "Base64 encoded data"},
        {"type": "string", "format": "uri", "description": "URL to download"}
      ]
    },
    "file_name": {"type": "string"},
    "tags": {"type": "array", "items": {"type": "string"}},
    "metadata": {"type": "object"}
  },
  "required": ["owner_id", "name", "data"]
}
```

**Output Schema:**
```json
{
  "type": "object",
  "properties": {
    "id": {"type": "string", "format": "uuid"},
    "status": {"type": "string", "enum": ["uploaded"]},
    "download_url": {"type": "string", "format": "uri"},
    "created_at": {"type": "string", "format": "date-time"}
  }
}
```

**Implementation:**
```go
func (s *Server) handleUploadContent(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    // Parse input
    params := req.Params.Arguments

    // Validate required fields
    ownerID, err := parseUUID(params["owner_id"])
    if err != nil {
        return nil, newValidationError("owner_id", err)
    }

    // Decode data (base64 or URL)
    reader, err := s.decodeData(params["data"])
    if err != nil {
        return nil, err
    }

    // Call service
    content, err := s.service.UploadContent(ctx, simplecontent.UploadContentRequest{
        OwnerID:      ownerID,
        TenantID:     parseTenantID(params["tenant_id"]),
        Name:         params["name"].(string),
        Description:  getStringOr(params, "description", ""),
        DocumentType: getStringOr(params, "document_type", "application/octet-stream"),
        Reader:       reader,
        FileName:     getStringOr(params, "file_name", ""),
        Tags:         getStringSlice(params, "tags"),
        CustomMetadata: getMap(params, "metadata"),
    })
    if err != nil {
        return nil, s.mapError(err)
    }

    // Get download URL
    details, err := s.service.GetContentDetails(ctx, content.ID)
    if err != nil {
        return nil, s.mapError(err)
    }

    return mcp.NewToolResultText(formatJSON(map[string]interface{}{
        "id": content.ID,
        "status": content.Status,
        "download_url": details.DownloadURL,
        "created_at": content.CreatedAt,
    })), nil
}
```

#### 2. `get_content`

**Description:** Retrieve content metadata by ID

**Input Schema:**
```json
{
  "type": "object",
  "properties": {
    "content_id": {"type": "string", "format": "uuid"}
  },
  "required": ["content_id"]
}
```

**Output Schema:**
```json
{
  "type": "object",
  "properties": {
    "id": {"type": "string", "format": "uuid"},
    "owner_id": {"type": "string", "format": "uuid"},
    "tenant_id": {"type": "string", "format": "uuid"},
    "name": {"type": "string"},
    "description": {"type": "string"},
    "status": {"type": "string"},
    "derivation_type": {"type": "string"},
    "created_at": {"type": "string", "format": "date-time"},
    "updated_at": {"type": "string", "format": "date-time"}
  }
}
```

#### 3. `get_content_details`

**Description:** Get complete content information including URLs and metadata

**Input Schema:**
```json
{
  "type": "object",
  "properties": {
    "content_id": {"type": "string", "format": "uuid"},
    "include_upload_url": {"type": "boolean", "default": false}
  },
  "required": ["content_id"]
}
```

**Output Schema:**
```json
{
  "type": "object",
  "properties": {
    "id": {"type": "string", "format": "uuid"},
    "download_url": {"type": "string", "format": "uri"},
    "upload_url": {"type": "string", "format": "uri"},
    "thumbnail_url": {"type": "string", "format": "uri"},
    "thumbnails": {
      "type": "object",
      "additionalProperties": {"type": "string", "format": "uri"}
    },
    "preview_url": {"type": "string", "format": "uri"},
    "file_name": {"type": "string"},
    "file_size": {"type": "integer"},
    "mime_type": {"type": "string"},
    "tags": {"type": "array", "items": {"type": "string"}},
    "status": {"type": "string"},
    "ready": {"type": "boolean"},
    "metadata": {"type": "object"},
    "created_at": {"type": "string", "format": "date-time"},
    "updated_at": {"type": "string", "format": "date-time"}
  }
}
```

#### 4. `list_content`

**Description:** List content with filtering and pagination

**Input Schema:**
```json
{
  "type": "object",
  "properties": {
    "owner_id": {"type": "string", "format": "uuid"},
    "tenant_id": {"type": "string", "format": "uuid"},
    "status": {"type": "string"},
    "tags": {"type": "array", "items": {"type": "string"}},
    "limit": {"type": "integer", "default": 50, "maximum": 1000},
    "offset": {"type": "integer", "default": 0}
  }
}
```

**Output Schema:**
```json
{
  "type": "object",
  "properties": {
    "items": {
      "type": "array",
      "items": {"$ref": "#/content"}
    },
    "total": {"type": "integer"},
    "limit": {"type": "integer"},
    "offset": {"type": "integer"}
  }
}
```

#### 5. `download_content`

**Description:** Download content data (returns download URL or base64)

**Input Schema:**
```json
{
  "type": "object",
  "properties": {
    "content_id": {"type": "string", "format": "uuid"},
    "format": {
      "type": "string",
      "enum": ["url", "base64"],
      "default": "url"
    }
  },
  "required": ["content_id"]
}
```

**Output Schema:**
```json
{
  "type": "object",
  "properties": {
    "download_url": {"type": "string", "format": "uri"},
    "data": {"type": "string", "description": "Base64 encoded data"},
    "file_name": {"type": "string"},
    "mime_type": {"type": "string"},
    "size": {"type": "integer"}
  }
}
```

#### 6. `update_content`

**Description:** Update content metadata

**Input Schema:**
```json
{
  "type": "object",
  "properties": {
    "content_id": {"type": "string", "format": "uuid"},
    "name": {"type": "string"},
    "description": {"type": "string"},
    "tags": {"type": "array", "items": {"type": "string"}},
    "metadata": {"type": "object"}
  },
  "required": ["content_id"]
}
```

#### 7. `delete_content`

**Description:** Soft delete content

**Input Schema:**
```json
{
  "type": "object",
  "properties": {
    "content_id": {"type": "string", "format": "uuid"}
  },
  "required": ["content_id"]
}
```

**Output Schema:**
```json
{
  "type": "object",
  "properties": {
    "success": {"type": "boolean"},
    "deleted_at": {"type": "string", "format": "date-time"}
  }
}
```

#### 8. `search_content`

**Description:** Search content by metadata, tags, or full-text

**Input Schema:**
```json
{
  "type": "object",
  "properties": {
    "query": {"type": "string"},
    "owner_id": {"type": "string", "format": "uuid"},
    "tenant_id": {"type": "string", "format": "uuid"},
    "tags": {"type": "array", "items": {"type": "string"}},
    "status": {"type": "array", "items": {"type": "string"}},
    "created_after": {"type": "string", "format": "date-time"},
    "created_before": {"type": "string", "format": "date-time"},
    "limit": {"type": "integer", "default": 50},
    "offset": {"type": "integer", "default": 0}
  }
}
```

### Derived Content Tools

#### 9. `create_derived_content`

**Description:** Create derived content placeholder (for async processing)

**Input Schema:**
```json
{
  "type": "object",
  "properties": {
    "parent_id": {"type": "string", "format": "uuid"},
    "owner_id": {"type": "string", "format": "uuid"},
    "tenant_id": {"type": "string", "format": "uuid"},
    "derivation_type": {"type": "string"},
    "variant": {"type": "string"},
    "initial_status": {
      "type": "string",
      "enum": ["created", "processing"],
      "default": "created"
    },
    "metadata": {"type": "object"}
  },
  "required": ["parent_id", "derivation_type", "variant"]
}
```

#### 10. `upload_derived_content`

**Description:** Upload derived content (thumbnail, preview, etc.)

**Input Schema:**
```json
{
  "type": "object",
  "properties": {
    "parent_id": {"type": "string", "format": "uuid"},
    "owner_id": {"type": "string", "format": "uuid"},
    "tenant_id": {"type": "string", "format": "uuid"},
    "derivation_type": {"type": "string"},
    "variant": {"type": "string"},
    "data": {"type": "string", "description": "Base64 encoded"},
    "file_name": {"type": "string"},
    "tags": {"type": "array", "items": {"type": "string"}},
    "metadata": {"type": "object"}
  },
  "required": ["parent_id", "derivation_type", "variant", "data"]
}
```

#### 11. `list_derived_content`

**Description:** List derived content with filtering

**Input Schema:**
```json
{
  "type": "object",
  "properties": {
    "parent_id": {"type": "string", "format": "uuid"},
    "derivation_type": {"type": "string"},
    "variant": {"type": "string"},
    "status": {"type": "string"},
    "include_urls": {"type": "boolean", "default": false},
    "limit": {"type": "integer", "default": 50}
  }
}
```

### Status Management Tools

#### 12. `get_content_status`

**Description:** Get content processing status

**Input Schema:**
```json
{
  "type": "object",
  "properties": {
    "content_id": {"type": "string", "format": "uuid"}
  },
  "required": ["content_id"]
}
```

**Output Schema:**
```json
{
  "type": "object",
  "properties": {
    "id": {"type": "string", "format": "uuid"},
    "status": {"type": "string"},
    "ready": {"type": "boolean"},
    "updated_at": {"type": "string", "format": "date-time"}
  }
}
```

#### 13. `update_content_status`

**Description:** Update content lifecycle status

**Input Schema:**
```json
{
  "type": "object",
  "properties": {
    "content_id": {"type": "string", "format": "uuid"},
    "status": {
      "type": "string",
      "enum": ["created", "uploading", "uploaded", "processing", "processed", "failed"]
    }
  },
  "required": ["content_id", "status"]
}
```

#### 14. `list_by_status`

**Description:** List all content by status (for monitoring/workers)

**Input Schema:**
```json
{
  "type": "object",
  "properties": {
    "status": {
      "type": "string",
      "enum": ["created", "processing", "failed", "uploaded", "processed"]
    },
    "limit": {"type": "integer", "default": 100}
  },
  "required": ["status"]
}
```

### Batch Operations Tools

#### 15. `batch_upload`

**Description:** Upload multiple content items in one operation

**Input Schema:**
```json
{
  "type": "object",
  "properties": {
    "items": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "name": {"type": "string"},
          "data": {"type": "string"},
          "file_name": {"type": "string"},
          "tags": {"type": "array"}
        }
      }
    },
    "owner_id": {"type": "string", "format": "uuid"},
    "tenant_id": {"type": "string", "format": "uuid"}
  },
  "required": ["items", "owner_id"]
}
```

**Output Schema:**
```json
{
  "type": "object",
  "properties": {
    "results": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "success": {"type": "boolean"},
          "content_id": {"type": "string", "format": "uuid"},
          "error": {"type": "string"}
        }
      }
    },
    "total": {"type": "integer"},
    "successful": {"type": "integer"},
    "failed": {"type": "integer"}
  }
}
```

#### 16. `batch_get_details`

**Description:** Get details for multiple content IDs

**Input Schema:**
```json
{
  "type": "object",
  "properties": {
    "content_ids": {
      "type": "array",
      "items": {"type": "string", "format": "uuid"}
    }
  },
  "required": ["content_ids"]
}
```

---

## MCP Resources Specification

Resources provide discoverable context for AI agents. They use URI-based addressing.

### Resource URIs

#### 1. `content://{content_id}`

**Description:** Individual content metadata
**MIME Type:** `application/json`

**Response:**
```json
{
  "uri": "content://123e4567-e89b-12d3-a456-426614174000",
  "name": "My Document",
  "mimeType": "application/json",
  "text": "{...content JSON...}"
}
```

#### 2. `content://{content_id}/details`

**Description:** Complete content details with URLs
**MIME Type:** `application/json`

#### 3. `derived://{parent_id}`

**Description:** List of derived content for a parent
**MIME Type:** `application/json`

#### 4. `schema://content`

**Description:** Content entity JSON schema
**MIME Type:** `application/schema+json`

**Response:**
```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "id": {"type": "string", "format": "uuid"},
    "owner_id": {"type": "string", "format": "uuid"},
    "name": {"type": "string"},
    "status": {"type": "string", "enum": ["created", "uploaded", "processed"]},
    ...
  }
}
```

#### 5. `schema://derived-content`

**Description:** Derived content relationship schema
**MIME Type:** `application/schema+json`

#### 6. `backends://storage`

**Description:** Available storage backends
**MIME Type:** `application/json`

**Response:**
```json
{
  "backends": [
    {
      "name": "default",
      "type": "s3",
      "available": true,
      "supports_presigned": true
    },
    {
      "name": "filesystem",
      "type": "fs",
      "available": true,
      "supports_presigned": false
    }
  ]
}
```

#### 7. `stats://system`

**Description:** System statistics and health
**MIME Type:** `application/json`

**Response:**
```json
{
  "content_count": {
    "total": 1234,
    "by_status": {
      "uploaded": 1000,
      "processing": 50,
      "failed": 5,
      "created": 179
    }
  },
  "derived_count": {
    "total": 3456,
    "by_type": {
      "thumbnail": 2000,
      "preview": 1456
    }
  },
  "storage_usage": {
    "total_bytes": 12345678900
  }
}
```

### Resource Templates

Support URI templates for dynamic resources:

- `content://{id}` - Single content
- `derived://{parent_id}?type={derivation_type}` - Filtered derivatives
- `stats://content?owner_id={owner_id}` - Owner-specific stats

---

## MCP Prompts Specification

Prompts guide agents through common workflows.

### Prompt Templates

#### 1. `upload-workflow`

**Description:** Guide for uploading content
**Arguments:**
- `content_type` (optional): Type of content being uploaded

**Template:**
```
To upload content to the system:

1. Prepare your content data (encode as base64 if binary)
2. Call `upload_content` tool with:
   - owner_id: Your user/organization ID
   - name: Descriptive name for the content
   - data: Base64 encoded content or URL
   - file_name: Original filename (optional)
   - tags: Array of tags for categorization
3. The tool returns:
   - content_id: Use this for future operations
   - download_url: Direct download link
   - status: Should be "uploaded"

Example:
{
  "owner_id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "Company Logo",
  "data": "iVBORw0KGgoAAAANS...",
  "file_name": "logo.png",
  "tags": ["branding", "images"]
}

For large files, consider using the presigned upload workflow instead.
```

#### 2. `thumbnail-generation`

**Description:** Generate thumbnails for images
**Arguments:**
- `parent_id`: ID of the original image content
- `sizes`: Array of thumbnail sizes (e.g., ["256", "512"])

**Template:**
```
To generate thumbnails for an image:

1. Ensure parent content is uploaded (status="uploaded")
2. For each thumbnail size:
   a. Download the original image using `download_content`
   b. Resize the image to target dimensions
   c. Encode resized image as base64
   d. Call `upload_derived_content` with:
      - parent_id: Original image ID
      - derivation_type: "thumbnail"
      - variant: "thumbnail_256" (or appropriate size)
      - data: Base64 encoded thumbnail
3. Verify all thumbnails with `list_derived_content`

The system automatically associates thumbnails with the parent.
Use `get_content_details` to retrieve all thumbnail URLs.
```

#### 3. `async-processing`

**Description:** Pattern for async content processing
**Arguments:**
- `operation_type`: Type of processing (thumbnail, transcode, etc.)

**Template:**
```
Async processing pattern (for workers):

1. Create placeholder:
   - Call `create_derived_content` with initial_status="processing"
   - Store the returned content_id

2. Perform processing:
   - Download source content
   - Execute your processing logic
   - Handle errors and retries

3. Upload result:
   - Call `upload_object_for_content` with processed data
   - Or call `upload_derived_content` if starting fresh

4. Update status:
   - Call `update_content_status` to "processed" on success
   - Or to "failed" with error metadata

5. Monitor:
   - Workers can query `list_by_status` for pending jobs
   - Agents can poll `get_content_status` for completion
```

#### 4. `batch-upload`

**Description:** Upload multiple files efficiently
**Arguments:**
- `file_count`: Number of files to upload

**Template:**
```
For uploading multiple files:

Option 1: Use batch_upload tool (recommended for <100 files)
- Prepare all files as base64
- Call `batch_upload` with array of items
- Receive array of results with success/failure per item

Option 2: Sequential upload (for large batches)
- Loop through files
- Call `upload_content` for each
- Collect results and handle errors individually

Option 3: Parallel upload (fastest)
- Split files into chunks
- Upload chunks concurrently
- Aggregate results

The batch_upload tool automatically handles parallelization
and provides a summary of successful/failed uploads.
```

#### 5. `content-search`

**Description:** Search and filter content
**Arguments:**
- `search_type`: Type of search (tags, metadata, full-text)

**Template:**
```
To search for content:

By Tags:
- Call `search_content` with tags array
- Example: {"tags": ["invoice", "2024"]}

By Status:
- Call `list_by_status` with desired status
- Useful for finding failed uploads or pending processing

By Metadata:
- Call `search_content` with query string
- Searches in name, description, and metadata fields

By Date Range:
- Use created_after/created_before parameters
- Example: {"created_after": "2024-01-01T00:00:00Z"}

All search methods support pagination with limit/offset.
```

---

## Implementation Phases

### Phase 1: Foundation & Core Tools (Week 1)

**Goals:**
- ✅ Repository setup with official MCP SDK
- ✅ Server wrapper implementation
- ✅ 8 core content tools
- ✅ Error handling and validation

**Deliverables:**
1. `pkg/mcpserver/server.go` - MCP server wrapper
2. `pkg/mcpserver/tools/content.go` - Core tools:
   - upload_content
   - get_content
   - get_content_details
   - list_content
   - download_content
   - update_content
   - delete_content
   - search_content
3. `pkg/mcpserver/errors/errors.go` - Error mapping
4. `cmd/mcpserver/main.go` - Server entrypoint
5. Basic documentation

**Success Criteria:**
- Server starts in stdio mode
- All 8 tools callable and functional
- Errors properly formatted as MCP errors
- Example client can upload and download content

### Phase 2: Derived Content & Status Tools (Week 2)

**Goals:**
- ✅ Derived content tools
- ✅ Status management tools
- ✅ Enhanced error handling

**Deliverables:**
1. `pkg/mcpserver/tools/derived.go`:
   - create_derived_content
   - upload_derived_content
   - list_derived_content
2. `pkg/mcpserver/tools/status.go`:
   - get_content_status
   - update_content_status
   - list_by_status
3. Input/output schema validation
4. Tool documentation

**Success Criteria:**
- Agents can generate thumbnails
- Async processing workflow works
- Status transitions validated

### Phase 3: Resources & Prompts (Week 3)

**Goals:**
- ✅ MCP resources implementation
- ✅ MCP prompts for workflows
- ✅ Discovery features

**Deliverables:**
1. `pkg/mcpserver/resources/resources.go`:
   - content:// URIs
   - schema:// URIs
   - stats:// URIs
2. `pkg/mcpserver/prompts/prompts.go`:
   - upload-workflow
   - thumbnail-generation
   - async-processing
   - batch-upload
   - content-search
3. Resource templates for dynamic URIs
4. Prompt argument handling

**Success Criteria:**
- Agents discover schemas via resources
- Prompts guide agents through workflows
- Resource URIs resolve correctly

### Phase 4: Batch Operations & Advanced Features (Week 4)

**Goals:**
- ✅ Batch operations
- ✅ Performance optimization
- ✅ Advanced filtering

**Deliverables:**
1. `pkg/mcpserver/tools/batch.go`:
   - batch_upload
   - batch_get_details
2. Pagination support for all list operations
3. Advanced filtering options
4. Cursor-based iteration

**Success Criteria:**
- Can upload 100 files in one batch operation
- List operations handle 10,000+ items efficiently
- Advanced filters work correctly

### Phase 5: Production Hardening (Week 5)

**Goals:**
- ✅ Authentication and security
- ✅ Deployment artifacts
- ✅ Comprehensive testing
- ✅ Documentation

**Deliverables:**
1. `pkg/mcpserver/auth/`:
   - API key authentication
   - OAuth integration (via SDK)
   - Owner/tenant scoping
2. Docker image
3. Kubernetes manifests
4. Integration tests
5. Complete documentation:
   - TOOLS.md
   - RESOURCES.md
   - PROMPTS.md
   - DEPLOYMENT.md
   - AGENT_GUIDE.md

**Success Criteria:**
- Production-ready Docker image
- All tools have tests
- Documentation complete
- Security audit passed

---

## API Design

### Server Configuration

```go
// Config holds server configuration
type Config struct {
    // Core dependencies
    Service         simplecontent.Service

    // Server settings
    Name            string
    Version         string

    // Transport settings
    Mode            TransportMode  // stdio, sse, http
    Host            string
    Port            int
    BaseURL         string         // For SSE mode

    // Authentication
    AuthEnabled     bool
    APIKeys         []string
    OAuthConfig     *OAuthConfig

    // Behavior
    MaxBatchSize    int
    DefaultPageSize int
    MaxPageSize     int

    // Feature flags
    EnableResources bool
    EnablePrompts   bool
}

type TransportMode string

const (
    TransportStdio TransportMode = "stdio"
    TransportSSE   TransportMode = "sse"
    TransportHTTP  TransportMode = "http"
)
```

### Server Interface

```go
package mcpserver

import (
    "context"

    "github.com/modelcontextprotocol/go-sdk/mcp"
    "github.com/tendant/simple-content/pkg/simplecontent"
)

// Server wraps a simple-content Service and exposes it via MCP
type Server struct {
    service    simplecontent.Service
    mcpServer  *mcp.Server
    config     Config
}

// New creates a new MCP server
func New(config Config) (*Server, error) {
    if config.Service == nil {
        return nil, fmt.Errorf("service is required")
    }

    s := &Server{
        service: config.Service,
        config:  config,
    }

    // Create MCP server
    impl := &mcp.Implementation{
        Name:    config.Name,
        Version: config.Version,
    }

    s.mcpServer = mcp.NewServer(impl, nil)

    // Register tools
    if err := s.registerTools(); err != nil {
        return nil, err
    }

    // Register resources if enabled
    if config.EnableResources {
        if err := s.registerResources(); err != nil {
            return nil, err
        }
    }

    // Register prompts if enabled
    if config.EnablePrompts {
        if err := s.registerPrompts(); err != nil {
            return nil, err
        }
    }

    return s, nil
}

// Serve starts the MCP server with the configured transport
func (s *Server) Serve(ctx context.Context) error {
    switch s.config.Mode {
    case TransportStdio:
        return s.serveStdio(ctx)
    case TransportSSE:
        return s.serveSSE(ctx)
    case TransportHTTP:
        return s.serveHTTP(ctx)
    default:
        return fmt.Errorf("unknown transport mode: %s", s.config.Mode)
    }
}

func (s *Server) serveStdio(ctx context.Context) error {
    transport := &mcp.StdioTransport{}
    return s.mcpServer.Run(ctx, transport)
}

func (s *Server) serveSSE(ctx context.Context) error {
    // Implement SSE transport
    // Use official SDK's SSE support when available
    return fmt.Errorf("SSE transport not implemented yet")
}

func (s *Server) serveHTTP(ctx context.Context) error {
    // Implement HTTP transport
    return fmt.Errorf("HTTP transport not implemented yet")
}
```

### Tool Registration

```go
func (s *Server) registerTools() error {
    // Content management tools
    tools := []*mcp.Tool{
        {
            Name:        "upload_content",
            Description: "Upload content with data in a single operation",
            InputSchema: uploadContentSchema,
        },
        {
            Name:        "get_content",
            Description: "Retrieve content metadata by ID",
            InputSchema: getContentSchema,
        },
        // ... more tools
    }

    for _, tool := range tools {
        handler := s.getToolHandler(tool.Name)
        if handler == nil {
            return fmt.Errorf("no handler for tool: %s", tool.Name)
        }
        mcp.AddTool(s.mcpServer, tool, handler)
    }

    return nil
}

func (s *Server) getToolHandler(name string) mcp.ToolHandler {
    switch name {
    case "upload_content":
        return s.handleUploadContent
    case "get_content":
        return s.handleGetContent
    // ... more cases
    default:
        return nil
    }
}
```

### Helper Functions

```go
// decodeData handles both base64 and URL data sources
func (s *Server) decodeData(data interface{}) (io.Reader, error) {
    dataStr, ok := data.(string)
    if !ok {
        return nil, fmt.Errorf("data must be a string")
    }

    // Check if it's a URL
    if strings.HasPrefix(dataStr, "http://") || strings.HasPrefix(dataStr, "https://") {
        return s.downloadFromURL(dataStr)
    }

    // Otherwise treat as base64
    decoded, err := base64.StdEncoding.DecodeString(dataStr)
    if err != nil {
        return nil, fmt.Errorf("invalid base64 data: %w", err)
    }

    return bytes.NewReader(decoded), nil
}

// parseUUID safely parses UUID from interface{}
func parseUUID(v interface{}) (uuid.UUID, error) {
    s, ok := v.(string)
    if !ok {
        return uuid.Nil, fmt.Errorf("expected string, got %T", v)
    }
    return uuid.Parse(s)
}

// getStringOr returns string value or default
func getStringOr(params map[string]interface{}, key, defaultVal string) string {
    if v, ok := params[key]; ok {
        if s, ok := v.(string); ok {
            return s
        }
    }
    return defaultVal
}

// getStringSlice safely extracts string array
func getStringSlice(params map[string]interface{}, key string) []string {
    v, ok := params[key]
    if !ok {
        return nil
    }

    arr, ok := v.([]interface{})
    if !ok {
        return nil
    }

    result := make([]string, 0, len(arr))
    for _, item := range arr {
        if s, ok := item.(string); ok {
            result = append(result, s)
        }
    }
    return result
}
```

---

## Security & Authentication

### Authentication Strategies

#### 1. API Key Authentication (Simple)

```go
type APIKeyAuth struct {
    validKeys map[string]*KeyInfo
}

type KeyInfo struct {
    Key       string
    OwnerID   uuid.UUID
    TenantID  uuid.UUID
    ExpiresAt *time.Time
}

func (a *APIKeyAuth) Validate(ctx context.Context, apiKey string) (*KeyInfo, error) {
    info, ok := a.validKeys[apiKey]
    if !ok {
        return nil, ErrInvalidAPIKey
    }

    if info.ExpiresAt != nil && time.Now().After(*info.ExpiresAt) {
        return nil, ErrExpiredAPIKey
    }

    return info, nil
}
```

#### 2. OAuth Integration (Official SDK)

Use the official SDK's `auth` package for OAuth flows:

```go
import "github.com/modelcontextprotocol/go-sdk/auth"

// Configure OAuth
oauthConfig := &auth.Config{
    ClientID:     "your-client-id",
    ClientSecret: "your-client-secret",
    RedirectURL:  "https://your-app.com/callback",
    Scopes:       []string{"content:read", "content:write"},
}
```

### Authorization Middleware

```go
func (s *Server) withAuth(handler mcp.ToolHandler) mcp.ToolHandler {
    return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
        // Extract API key from context or request metadata
        apiKey := extractAPIKey(ctx)

        // Validate
        keyInfo, err := s.auth.Validate(ctx, apiKey)
        if err != nil {
            return nil, &mcp.Error{
                Code:    mcp.ErrorCodeUnauthorized,
                Message: "Invalid or missing authentication",
            }
        }

        // Inject key info into context
        ctx = context.WithValue(ctx, keyInfoKey, keyInfo)

        // Call original handler
        return handler(ctx, req)
    }
}
```

### Owner/Tenant Scoping

```go
func (s *Server) enforceOwnership(ctx context.Context, ownerID uuid.UUID) error {
    keyInfo := ctx.Value(keyInfoKey).(*KeyInfo)

    // Check if API key has access to this owner
    if keyInfo.OwnerID != ownerID {
        return &mcp.Error{
            Code:    mcp.ErrorCodeForbidden,
            Message: "Access denied to this owner's content",
        }
    }

    return nil
}
```

---

## Testing Strategy

### Unit Tests

```go
func TestUploadContentTool(t *testing.T) {
    // Create mock service
    mockService := &MockService{
        UploadContentFunc: func(ctx context.Context, req simplecontent.UploadContentRequest) (*simplecontent.Content, error) {
            return &simplecontent.Content{
                ID:     uuid.New(),
                Status: "uploaded",
            }, nil
        },
    }

    // Create server
    server, err := New(Config{
        Service: mockService,
        Name:    "test-server",
        Version: "0.1.0",
    })
    require.NoError(t, err)

    // Prepare request
    req := mcp.CallToolRequest{
        Params: mcp.CallToolParams{
            Name: "upload_content",
            Arguments: map[string]interface{}{
                "owner_id": "550e8400-e29b-41d4-a716-446655440000",
                "name":     "Test File",
                "data":     base64.StdEncoding.EncodeToString([]byte("test data")),
            },
        },
    }

    // Call tool
    result, err := server.handleUploadContent(context.Background(), req)
    require.NoError(t, err)
    require.NotNil(t, result)

    // Verify result
    assert.Contains(t, result.Content[0].Text, "content_id")
}
```

### Integration Tests

```go
func TestMCPServerIntegration(t *testing.T) {
    // Setup real service with memory backend
    repo := memory.NewRepository()
    blobStore := memorystorage.New()

    service, err := simplecontent.New(
        simplecontent.WithRepository(repo),
        simplecontent.WithBlobStore("default", blobStore),
    )
    require.NoError(t, err)

    // Create MCP server
    mcpServer, err := New(Config{
        Service: service,
        Name:    "integration-test",
        Version: "0.1.0",
    })
    require.NoError(t, err)

    // Test upload workflow
    t.Run("upload and download", func(t *testing.T) {
        // Upload
        uploadReq := mcp.CallToolRequest{...}
        uploadResult, err := mcpServer.handleUploadContent(ctx, uploadReq)
        require.NoError(t, err)

        // Extract content ID from result
        contentID := extractContentID(uploadResult)

        // Download
        downloadReq := mcp.CallToolRequest{
            Params: mcp.CallToolParams{
                Name: "download_content",
                Arguments: map[string]interface{}{
                    "content_id": contentID,
                    "format":     "base64",
                },
            },
        }
        downloadResult, err := mcpServer.handleDownloadContent(ctx, downloadReq)
        require.NoError(t, err)

        // Verify data matches
        assert.Equal(t, originalData, decodedData)
    })
}
```

### Example Agent Scripts

Create example scripts that agents can use:

**examples/agent-upload/main.go:**
```go
// Example: Agent uploading content via MCP
package main

import (
    "context"
    "encoding/base64"
    "fmt"
    "os"
    "os/exec"

    "github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
    // Create MCP client
    client := mcp.NewClient(&mcp.Implementation{
        Name: "example-agent",
    }, nil)

    // Connect to server via stdio
    transport := &mcp.CommandTransport{
        Command: exec.Command("./mcpserver", "--mode=stdio"),
    }

    session, err := client.Connect(context.Background(), transport, nil)
    if err != nil {
        panic(err)
    }
    defer session.Close()

    // Read file to upload
    data, err := os.ReadFile("./test.txt")
    if err != nil {
        panic(err)
    }

    // Upload via MCP tool
    result, err := session.CallTool(context.Background(), &mcp.CallToolRequest{
        Params: mcp.CallToolParams{
            Name: "upload_content",
            Arguments: map[string]interface{}{
                "owner_id": "550e8400-e29b-41d4-a716-446655440000",
                "name":     "Test Document",
                "data":     base64.StdEncoding.EncodeToString(data),
                "file_name": "test.txt",
                "tags":     []string{"example", "test"},
            },
        },
    })

    if err != nil {
        panic(err)
    }

    fmt.Printf("Upload successful: %s\n", result.Content[0].Text)
}
```

---

## Deployment

### Docker Image

**Dockerfile:**
```dockerfile
FROM golang:1.24-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o mcpserver ./cmd/mcpserver

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /build/mcpserver .

# Default to stdio mode
ENV MCP_MODE=stdio

ENTRYPOINT ["./mcpserver"]
CMD ["--mode=${MCP_MODE}"]
```

### Docker Compose

**docker-compose.yml:**
```yaml
version: '3.8'

services:
  mcpserver:
    build: .
    environment:
      - MCP_MODE=sse
      - MCP_PORT=8080
      - MCP_BASE_URL=http://localhost:8080
      - DATABASE_URL=postgresql://user:pass@postgres:5432/content
      - STORAGE_BACKEND=s3
      - AWS_S3_ENDPOINT=http://minio:9000
      - AWS_ACCESS_KEY_ID=minioadmin
      - AWS_SECRET_ACCESS_KEY=minioadmin
      - AWS_S3_BUCKET=content
    ports:
      - "8080:8080"
    depends_on:
      - postgres
      - minio

  postgres:
    image: postgres:15-alpine
    environment:
      - POSTGRES_USER=content
      - POSTGRES_PASSWORD=contentpass
      - POSTGRES_DB=simple_content
    volumes:
      - postgres_data:/var/lib/postgresql/data

  minio:
    image: minio/minio
    command: server /data --console-address ":9001"
    environment:
      - MINIO_ROOT_USER=minioadmin
      - MINIO_ROOT_PASSWORD=minioadmin
    ports:
      - "9000:9000"
      - "9001:9001"
    volumes:
      - minio_data:/data

volumes:
  postgres_data:
  minio_data:
```

### Kubernetes Deployment

**k8s/deployment.yaml:**
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: simple-content-mcp
spec:
  replicas: 3
  selector:
    matchLabels:
      app: simple-content-mcp
  template:
    metadata:
      labels:
        app: simple-content-mcp
    spec:
      containers:
      - name: mcpserver
        image: simple-content-mcp:latest
        ports:
        - containerPort: 8080
          name: sse
        env:
        - name: MCP_MODE
          value: "sse"
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: simple-content-secrets
              key: database-url
        - name: AWS_ACCESS_KEY_ID
          valueFrom:
            secretKeyRef:
              name: simple-content-secrets
              key: aws-access-key-id
        - name: AWS_SECRET_ACCESS_KEY
          valueFrom:
            secretKeyRef:
              name: simple-content-secrets
              key: aws-secret-access-key
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 30
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 10
---
apiVersion: v1
kind: Service
metadata:
  name: simple-content-mcp
spec:
  selector:
    app: simple-content-mcp
  ports:
  - port: 80
    targetPort: 8080
  type: LoadBalancer
```

---

## Migration Guide

### From mark3labs/mcp-go to Official SDK

#### 1. Update Dependencies

**Before:**
```go
import "github.com/mark3labs/mcp-go/server"
```

**After:**
```go
import "github.com/modelcontextprotocol/go-sdk/mcp"
```

#### 2. Server Initialization

**Before:**
```go
s := server.NewMCPServer(
    "Content Server",
    "1.0.0",
    server.WithResourceCapabilities(true, true),
)
```

**After:**
```go
impl := &mcp.Implementation{
    Name:    "Content Server",
    Version: "1.0.0",
}
s := mcp.NewServer(impl, nil)
```

#### 3. Tool Registration

**Before:**
```go
tool := mcp.NewTool("upload_content",
    mcp.WithDescription("Upload content"),
    mcp.WithString("data", mcp.Required()),
)
s.AddTool(tool, handler)
```

**After:**
```go
tool := &mcp.Tool{
    Name:        "upload_content",
    Description: "Upload content",
    InputSchema: map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "data": map[string]interface{}{
                "type": "string",
            },
        },
        "required": []string{"data"},
    },
}
mcp.AddTool(s, tool, handler)
```

#### 4. Transport

**Before:**
```go
server.ServeStdio(s)
```

**After:**
```go
transport := &mcp.StdioTransport{}
s.Run(context.Background(), transport)
```

### Breaking Changes

1. **Schema Format**: Input schemas are now pure JSON Schema objects, not builder patterns
2. **Error Handling**: Use `mcp.Error` type with proper error codes
3. **Transport**: Must explicitly create and pass transport to `Run()`
4. **Context**: Context is now required for all operations

### Migration Checklist

- [ ] Update go.mod to use official SDK
- [ ] Update import statements
- [ ] Refactor server initialization
- [ ] Convert tool schemas to JSON Schema format
- [ ] Update error handling to use mcp.Error
- [ ] Update transport initialization
- [ ] Add context to all handlers
- [ ] Test all tools
- [ ] Update documentation
- [ ] Deploy and verify

---

## Appendix

### Example Tool Schemas

All tool input/output schemas following JSON Schema Draft 7 specification.

### Error Codes

Standard MCP error codes used:
- `-32600`: Invalid Request
- `-32601`: Method Not Found (tool not found)
- `-32602`: Invalid Params (validation error)
- `-32603`: Internal Error
- Custom: `40001` - Unauthorized
- Custom: `40003` - Forbidden
- Custom: `40004` - Not Found
- Custom: `50001` - Storage Error

### Metrics & Monitoring

Recommended metrics to track:
- Tool call count by tool name
- Tool call latency percentiles
- Error rate by error code
- Active connections (SSE/HTTP modes)
- Content upload/download throughput

### Performance Considerations

- Use streaming for large file downloads (>10MB)
- Batch operations reduce round trips by 90%
- Resource caching improves discovery performance
- Connection pooling for database access
- Presigned URLs offload storage traffic from MCP server

---

## Version History

- **v1.0.0-alpha** (2025-10-02): Initial plan
- Future versions will track implementation progress

---

## References

- [MCP Specification](https://modelcontextprotocol.io/specification)
- [Official Go SDK](https://github.com/modelcontextprotocol/go-sdk)
- [Simple Content Library](https://github.com/tendant/simple-content)
- [JSON Schema Draft 7](https://json-schema.org/draft-07/schema)

---

**End of Plan Document**
