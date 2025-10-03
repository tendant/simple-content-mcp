package mcpserver

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/tendant/simple-content/pkg/simplecontent"

	mcperrors "github.com/tendant/simple-content-mcp/pkg/mcpserver/errors"
)

// Server wraps a simple-content Service and exposes it via MCP
type Server struct {
	service   simplecontent.Service
	mcpServer *mcp.Server
	config    Config
}

// New creates a new MCP server
func New(config Config) (*Server, error) {
	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, err
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

	// TODO: Phase 3 - Register resources and prompts
	// if config.EnableResources {
	//     if err := s.registerResources(); err != nil {
	//         return nil, err
	//     }
	// }
	// if config.EnablePrompts {
	//     if err := s.registerPrompts(); err != nil {
	//         return nil, err
	//     }
	// }

	return s, nil
}

// Serve starts the MCP server with the configured transport
func (s *Server) Serve(ctx context.Context) error {
	switch s.config.Mode {
	case TransportStdio:
		return s.serveStdio(ctx)
	case TransportSSE:
		return fmt.Errorf("SSE transport not implemented yet (Phase 5)")
	case TransportHTTP:
		return fmt.Errorf("HTTP transport not implemented yet (Phase 5)")
	default:
		return fmt.Errorf("unknown transport mode: %s", s.config.Mode)
	}
}

func (s *Server) serveStdio(ctx context.Context) error {
	transport := &mcp.StdioTransport{}
	return s.mcpServer.Run(ctx, transport)
}

// Helper functions

// decodeData handles both base64 and URL data sources
func (s *Server) decodeData(data interface{}) (io.Reader, error) {
	dataStr, ok := data.(string)
	if !ok {
		return nil, mcperrors.NewValidationError("data", fmt.Errorf("must be a string"))
	}

	// Check if it's a URL
	if strings.HasPrefix(dataStr, "http://") || strings.HasPrefix(dataStr, "https://") {
		return s.downloadFromURL(dataStr)
	}

	// Otherwise treat as base64
	decoded, err := base64.StdEncoding.DecodeString(dataStr)
	if err != nil {
		return nil, mcperrors.NewValidationError("data", fmt.Errorf("invalid base64: %w", err))
	}

	return bytes.NewReader(decoded), nil
}

// downloadFromURL fetches data from a URL
func (s *Server) downloadFromURL(url string) (io.Reader, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, mcperrors.NewInternalError(fmt.Errorf("failed to download from URL: %w", err))
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, mcperrors.NewInternalError(fmt.Errorf("failed to download from URL: status %d", resp.StatusCode))
	}

	// Read the entire response into memory
	// For very large files, we might want to use a different approach
	data, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, mcperrors.NewInternalError(fmt.Errorf("failed to read from URL: %w", err))
	}

	return bytes.NewReader(data), nil
}

// parseUUID safely parses UUID from interface{}
func parseUUID(v interface{}) (uuid.UUID, error) {
	s, ok := v.(string)
	if !ok {
		return uuid.Nil, fmt.Errorf("expected string, got %T", v)
	}
	return uuid.Parse(s)
}

// parseTenantID safely parses optional tenant ID
func parseTenantID(v interface{}) uuid.UUID {
	if v == nil {
		return uuid.Nil
	}
	id, err := parseUUID(v)
	if err != nil {
		return uuid.Nil
	}
	return id
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

// getIntOr returns int value or default
func getIntOr(params map[string]interface{}, key string, defaultVal int) int {
	if v, ok := params[key]; ok {
		switch val := v.(type) {
		case int:
			return val
		case float64:
			return int(val)
		case int64:
			return int(val)
		}
	}
	return defaultVal
}

// getBoolOr returns bool value or default
func getBoolOr(params map[string]interface{}, key string, defaultVal bool) bool {
	if v, ok := params[key]; ok {
		if b, ok := v.(bool); ok {
			return b
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

// getMap safely extracts a map
func getMap(params map[string]interface{}, key string) map[string]interface{} {
	v, ok := params[key]
	if !ok {
		return nil
	}

	m, ok := v.(map[string]interface{})
	if !ok {
		return nil
	}

	return m
}

// formatJSON formats a value as JSON string
func formatJSON(v interface{}) string {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("%+v", v)
	}
	return string(data)
}

// newTextResult creates a CallToolResult with text content
func newTextResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: text,
			},
		},
	}
}

// mapError maps simple-content errors to MCP errors
func (s *Server) mapError(err error) error {
	return mcperrors.MapError(err)
}
