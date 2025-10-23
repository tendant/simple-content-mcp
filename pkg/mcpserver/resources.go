package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/tendant/simple-content/pkg/simplecontent"
	mcperrors "github.com/tendant/simple-content-mcp/pkg/mcpserver/errors"
)

// registerResources registers all MCP resources with the server
func (s *Server) registerResources() error {
	if !s.config.EnableResources {
		return nil
	}

	// Register resource templates (with parameters)
	templates := []*mcp.ResourceTemplate{
		{
			URITemplate: "content://{id}",
			Name:        "content",
			Description: "Content metadata by ID",
			MIMEType:    "application/json",
		},
	}

	for _, template := range templates {
		handler := s.getResourceTemplateHandler(template.Name)
		if handler == nil {
			return &ConfigError{Field: "resources", Message: "no handler for resource template: " + template.Name}
		}
		s.mcpServer.AddResourceTemplate(template, handler)
	}

	// Register static resources
	resources := []*mcp.Resource{
		{
			URI:         "schema://content",
			Name:        "content-schema",
			Description: "JSON schema for Content entity",
			MIMEType:    "application/schema+json",
		},
		{
			URI:         "stats://system",
			Name:        "system-stats",
			Description: "System statistics and health",
			MIMEType:    "application/json",
		},
	}

	for _, resource := range resources {
		handler := s.getResourceHandler(resource.Name)
		if handler == nil {
			return &ConfigError{Field: "resources", Message: "no handler for resource: " + resource.Name}
		}
		s.mcpServer.AddResource(resource, handler)
	}

	return nil
}

// getResourceTemplateHandler returns the handler function for a resource template by name
func (s *Server) getResourceTemplateHandler(name string) mcp.ResourceHandler {
	switch name {
	case "content":
		return s.handleContentResource
	default:
		return nil
	}
}

// getResourceHandler returns the handler function for a static resource by name
func (s *Server) getResourceHandler(name string) mcp.ResourceHandler {
	switch name {
	case "content-schema":
		return s.handleContentSchemaResource
	case "system-stats":
		return s.handleSystemStatsResource
	default:
		return nil
	}
}

// handleContentResource handles content://{id} and content://{id}/details
func (s *Server) handleContentResource(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	uri := req.Params.URI

	// Parse URI
	if !strings.HasPrefix(uri, "content://") {
		return nil, mcp.ResourceNotFoundError(uri)
	}

	path := strings.TrimPrefix(uri, "content://")
	parts := strings.Split(path, "/")

	if len(parts) == 0 || parts[0] == "" {
		return nil, mcp.ResourceNotFoundError(uri)
	}

	// Parse content ID
	contentID, err := uuid.Parse(parts[0])
	if err != nil {
		return nil, mcperrors.NewValidationError("id", err)
	}

	// Check for /details suffix
	if len(parts) == 2 && parts[1] == "details" {
		return s.handleContentDetailsResource(ctx, contentID, uri)
	}

	// Get content metadata
	content, err := s.service.GetContent(ctx, contentID)
	if err != nil {
		return nil, s.mapError(err)
	}

	// Format as JSON
	data := map[string]interface{}{
		"id":          content.ID.String(),
		"owner_id":    content.OwnerID.String(),
		"name":        content.Name,
		"description": content.Description,
		"status":      content.Status,
		"created_at":  content.CreatedAt,
		"updated_at":  content.UpdatedAt,
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal content: %w", err)
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      uri,
				MIMEType: "application/json",
				Text:     string(jsonData),
			},
		},
	}, nil
}

// handleContentDetailsResource handles content://{id}/details
func (s *Server) handleContentDetailsResource(ctx context.Context, contentID uuid.UUID, uri string) (*mcp.ReadResourceResult, error) {
	// Get content details
	details, err := s.service.GetContentDetails(ctx, contentID)
	if err != nil {
		return nil, s.mapError(err)
	}

	// Format as JSON
	data := map[string]interface{}{
		"id":           details.ID,
		"download_url": details.Download,
		"file_name":    details.FileName,
		"file_size":    details.FileSize,
		"mime_type":    details.MimeType,
		"ready":        details.Ready,
		"created_at":   details.CreatedAt,
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal content details: %w", err)
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      uri,
				MIMEType: "application/json",
				Text:     string(jsonData),
			},
		},
	}, nil
}

// handleContentSchemaResource returns the JSON schema for Content entity
func (s *Server) handleContentSchemaResource(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	schema := map[string]interface{}{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type":    "object",
		"properties": map[string]interface{}{
			"id": map[string]interface{}{
				"type":   "string",
				"format": "uuid",
			},
			"owner_id": map[string]interface{}{
				"type":   "string",
				"format": "uuid",
			},
			"tenant_id": map[string]interface{}{
				"type":   "string",
				"format": "uuid",
			},
			"name": map[string]interface{}{
				"type": "string",
			},
			"description": map[string]interface{}{
				"type": "string",
			},
			"status": map[string]interface{}{
				"type": "string",
				"enum": []string{
					"created",
					"uploading",
					"uploaded",
					"processing",
					"processed",
					"failed",
					"archived",
				},
			},
			"created_at": map[string]interface{}{
				"type":   "string",
				"format": "date-time",
			},
			"updated_at": map[string]interface{}{
				"type":   "string",
				"format": "date-time",
			},
		},
		"required": []string{"id", "owner_id", "name", "status"},
	}

	jsonData, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal schema: %w", err)
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      req.Params.URI,
				MIMEType: "application/schema+json",
				Text:     string(jsonData),
			},
		},
	}, nil
}

// handleSystemStatsResource returns system statistics
func (s *Server) handleSystemStatsResource(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	// Get counts by status
	statusCounts := make(map[string]int)
	statuses := []simplecontent.ContentStatus{
		"created",
		"uploading",
		"uploaded",
		"processing",
		"processed",
		"failed",
		"archived",
	}

	total := 0
	for _, status := range statuses {
		contentList, err := s.service.GetContentByStatus(ctx, status)
		if err == nil {
			count := len(contentList)
			statusCounts[string(status)] = count
			total += count
		}
	}

	stats := map[string]interface{}{
		"content_count": map[string]interface{}{
			"total":     total,
			"by_status": statusCounts,
		},
		"timestamp": "now",
	}

	jsonData, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal stats: %w", err)
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     string(jsonData),
			},
		},
	}, nil
}
