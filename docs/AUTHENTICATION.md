# Authentication Guide

## Overview

Phase 5 adds authentication support to the MCP server, allowing you to secure your content management operations with API keys.

## Features

- **API Key Authentication** - Simple key-based authentication
- **Owner/Tenant Scoping** - Keys are associated with specific owners and optionally tenants
- **Key Expiration** - Optional expiration dates for keys
- **Transport Integration** - Works with all transport modes (stdio, SSE, HTTP)

## Configuration

### Enabling Authentication

You can configure authentication using environment variables or a `.env` file.

#### Using .env File (Recommended)

```bash
# Copy example configuration
cp .env.example .env

# Edit .env file
# nano .env
```

Add to your `.env` file:
```bash
# Enable authentication
MCP_AUTH_ENABLED=true

# Add API keys (format: key:owner_id:tenant_id:expires_at)
MCP_API_KEY_1=mykey123:550e8400-e29b-41d4-a716-446655440000::
MCP_API_KEY_2=anotherkey:660e8400-e29b-41d4-a716-446655440001::
```

#### Using Environment Variables

```bash
# Enable authentication
export MCP_AUTH_ENABLED=true

# Add API keys (format: key:owner_id:tenant_id:expires_at)
export MCP_API_KEY_1="mykey123:550e8400-e29b-41d4-a716-446655440000::"
export MCP_API_KEY_2="anotherkey:660e8400-e29b-41d4-a716-446655440001::"
```

### API Key Format

API keys are defined using the format:
```
KEY:OWNER_ID:TENANT_ID:EXPIRES_AT
```

- **KEY** (required) - The API key string
- **OWNER_ID** (required) - UUID of the content owner
- **TENANT_ID** (optional) - UUID of the tenant (empty for none)
- **EXPIRES_AT** (optional) - RFC3339 timestamp (empty for no expiration)

### Examples

```bash
# Simple key with owner
export MCP_API_KEY_1="prod-key-123:550e8400-e29b-41d4-a716-446655440000::"

# Key with owner and tenant
export MCP_API_KEY_2="tenant-key:550e8400-e29b-41d4-a716-446655440000:660e8400-e29b-41d4-a716-446655440001:"

# Key with expiration (expires 2026-01-01)
export MCP_API_KEY_3="temp-key:550e8400-e29b-41d4-a716-446655440000::2026-01-01T00:00:00Z"
```

## Usage by Transport

### stdio Mode

For stdio mode, authentication is disabled by default as the process is typically running locally. You can enable it for testing:

```bash
export MCP_AUTH_ENABLED=true
export MCP_API_KEY_1="testkey:550e8400-e29b-41d4-a716-446655440000::"
./mcpserver --mode=stdio
```

The API key would need to be injected via context (implementation-specific).

### SSE Mode

For SSE transport, pass the API key in headers:

```bash
# Start server
export MCP_AUTH_ENABLED=true
export MCP_API_KEY_1="mykey:550e8400-e29b-41d4-a716-446655440000::"
./mcpserver --mode=sse --port=8080
```

Client request:
```bash
curl -H "X-API-Key: mykey" http://localhost:8080/sse
# OR
curl -H "Authorization: Bearer mykey" http://localhost:8080/sse
```

### HTTP Mode

For HTTP transport, pass the API key in headers:

```bash
# Start server
export MCP_AUTH_ENABLED=true
export MCP_API_KEY_1="mykey:550e8400-e29b-41d4-a716-446655440000::"
./mcpserver --mode=http --port=8080
```

Client request:
```bash
curl -X POST \
  -H "X-API-Key: mykey" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"tools/call","params":{...}}' \
  http://localhost:8080/mcp
```

## Programmatic Configuration

```go
package main

import (
    "github.com/google/uuid"
    "github.com/tendant/simple-content-mcp/pkg/mcpserver"
    "github.com/tendant/simple-content-mcp/pkg/mcpserver/auth"
)

func main() {
    // Create authenticator
    authenticator := auth.NewAPIKeyAuthenticator()

    // Add API keys
    authenticator.AddKey(&auth.KeyInfo{
        Key:     "mykey123",
        OwnerID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
    })

    // Configure server with authentication
    config := mcpserver.DefaultConfig(service)
    config.AuthEnabled = true
    config.Authenticator = authenticator

    server, _ := mcpserver.New(config)
    server.Serve(ctx)
}
```

## Security Best Practices

1. **Use HTTPS/TLS** - Always use TLS in production for SSE/HTTP modes
2. **Rotate Keys** - Regularly rotate API keys
3. **Use Expiration** - Set expiration dates for temporary access
4. **Scope Appropriately** - Use tenant IDs to limit access scope
5. **Monitor Usage** - Log and monitor API key usage
6. **Secure Storage** - Store API keys securely (use secrets management)

## Error Handling

When authentication fails, you'll receive errors:

- `authentication required: API key missing` - No API key provided
- `authentication failed: invalid or missing API key` - Invalid key
- `authentication failed: API key has expired` - Expired key
- `access denied` - Valid key but insufficient permissions

## Future Enhancements

Planned authentication features:

- [ ] OAuth 2.0 integration
- [ ] JWT token support
- [ ] Role-based access control (RBAC)
- [ ] Rate limiting per API key
- [ ] Key rotation without downtime
- [ ] Audit logging
