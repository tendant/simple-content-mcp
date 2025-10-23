package auth

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Middleware wraps a tool handler with authentication
func Middleware(authenticator Authenticator, handler mcp.ToolHandler) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Extract API key from request metadata
		// For stdio mode, we could use environment variables
		// For SSE/HTTP modes, we'd use headers
		apiKey := extractAPIKey(ctx, req)

		if apiKey == "" {
			return nil, fmt.Errorf("authentication required: API key missing")
		}

		// Validate API key
		keyInfo, err := authenticator.Validate(ctx, apiKey)
		if err != nil {
			return nil, fmt.Errorf("authentication failed: %w", err)
		}

		// Add key info to context
		ctx = WithKeyInfo(ctx, keyInfo)

		// Call original handler with authenticated context
		return handler(ctx, req)
	}
}

// extractAPIKey extracts the API key from the request
// This implementation can be extended based on transport mode
func extractAPIKey(ctx context.Context, req *mcp.CallToolRequest) string {
	// Try to get from context metadata (for HTTP/SSE transports)
	if apiKey, ok := ctx.Value("api_key").(string); ok {
		return apiKey
	}

	// Could also check environment variable for stdio mode
	// apiKey := os.Getenv("MCP_API_KEY")

	return ""
}
