package mcpserver

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
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

	// Phase 3 - Register resources and prompts
	if config.EnableResources {
		if err := s.registerResources(); err != nil {
			return nil, err
		}
	}
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
	// Create HTTP mux
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Readiness check endpoint
	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("READY"))
	})

	// MCP SSE endpoint using official SDK SSEHandler
	// The SSEHandler manages SSE sessions per the MCP spec
	sseHandler := mcp.NewSSEHandler(
		func(r *http.Request) *mcp.Server {
			// For now, return the same server instance for all requests
			// In production, you might want per-session servers with different contexts
			return s.mcpServer
		},
		&mcp.SSEOptions{
			// Endpoint where clients POST messages
			// Defaults to the request URL
		},
	)

	// Wrap SSE handler with MCP 2025-06-18 spec compliance and auth middleware
	mux.HandleFunc("/sse", func(w http.ResponseWriter, r *http.Request) {
		// MCP 2025-06-18: Validate Origin header for security
		origin := r.Header.Get("Origin")
		if origin != "" {
			// For localhost servers, only accept localhost origins
			// In production, implement stricter origin validation
			if s.config.Host == "localhost" || s.config.Host == "127.0.0.1" {
				if !strings.HasPrefix(origin, "http://localhost") && !strings.HasPrefix(origin, "http://127.0.0.1") {
					http.Error(w, "Invalid origin", http.StatusForbidden)
					return
				}
			}
		}

		// MCP 2025-06-18: Protocol version negotiation
		protocolVersion := r.Header.Get("MCP-Protocol-Version")
		if protocolVersion == "" {
			// Default to 2025-03-26 as per spec
			protocolVersion = "2025-03-26"
		}
		// Validate protocol version (we support 2024-11-05 and newer)
		supportedVersions := []string{"2024-11-05", "2025-03-26", "2025-06-18"}
		validVersion := false
		for _, v := range supportedVersions {
			if protocolVersion == v {
				validVersion = true
				break
			}
		}
		if !validVersion {
			http.Error(w, "Unsupported MCP protocol version", http.StatusBadRequest)
			return
		}

		// MCP 2025-06-18: Session ID handling
		sessionID := r.Header.Get("Mcp-Session-Id")
		if sessionID != "" {
			// Add session ID to context for downstream handlers
			ctx := context.WithValue(r.Context(), "mcp_session_id", sessionID)
			r = r.WithContext(ctx)
		}

		// Extract and validate API key if auth is enabled
		if s.config.AuthEnabled && s.config.Authenticator != nil {
			apiKey := extractAPIKeyFromHeader(r)
			if apiKey == "" {
				http.Error(w, "API key required", http.StatusUnauthorized)
				return
			}

			// Validate API key
			_, err := s.config.Authenticator.Validate(r.Context(), apiKey)
			if err != nil {
				http.Error(w, "Invalid API key", http.StatusUnauthorized)
				return
			}

			// Add API key to context for downstream handlers
			ctx := context.WithValue(r.Context(), "api_key", apiKey)
			r = r.WithContext(ctx)
		}

		log.Printf("SSE connection from %s (protocol: %s, session: %s)", r.RemoteAddr, protocolVersion, sessionID)
		sseHandler.ServeHTTP(w, r)
	})

	// Create HTTP server
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		log.Printf("Starting SSE server on %s", addr)
		log.Printf("SSE endpoint: http://%s/sse", addr)
		errChan <- server.ListenAndServe()
	}()

	// Wait for context cancellation or server error
	select {
	case <-ctx.Done():
		log.Println("Shutting down SSE server...")
		return server.Shutdown(context.Background())
	case err := <-errChan:
		return err
	}
}

func (s *Server) serveHTTP(ctx context.Context) error {
	// Create HTTP mux
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Readiness check endpoint
	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("READY"))
	})

	// MCP-over-HTTP endpoint
	mux.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Extract API key from header if auth is enabled
		reqCtx := r.Context()
		if s.config.AuthEnabled {
			apiKey := r.Header.Get("X-API-Key")
			if apiKey == "" {
				apiKey = r.Header.Get("Authorization")
				// Remove "Bearer " prefix if present
				if strings.HasPrefix(apiKey, "Bearer ") {
					apiKey = strings.TrimPrefix(apiKey, "Bearer ")
				}
			}

			if apiKey != "" {
				reqCtx = context.WithValue(reqCtx, "api_key", apiKey)
			}
		}

		// For now, return a basic response
		// Full implementation would parse JSON-RPC MCP requests and route to tools
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte(`{"error": "HTTP transport support coming soon"}`))
	})

	// Create HTTP server
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		log.Printf("Starting HTTP server on %s", addr)
		errChan <- server.ListenAndServe()
	}()

	// Wait for context cancellation or server error
	select {
	case <-ctx.Done():
		log.Println("Shutting down HTTP server...")
		return server.Shutdown(context.Background())
	case err := <-errChan:
		return err
	}
}

// Helper functions

// extractAPIKeyFromHeader extracts API key from HTTP request headers
func extractAPIKeyFromHeader(r *http.Request) string {
	// Try X-API-Key header first
	if apiKey := r.Header.Get("X-API-Key"); apiKey != "" {
		return apiKey
	}

	// Try Authorization header
	if auth := r.Header.Get("Authorization"); auth != "" {
		// Remove "Bearer " prefix if present
		if strings.HasPrefix(auth, "Bearer ") {
			return strings.TrimPrefix(auth, "Bearer ")
		}
		return auth
	}

	return ""
}

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
