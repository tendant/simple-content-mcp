package mcpserver

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// registerTools registers all MCP tools with the server
func (s *Server) registerTools() error {
	// Define all tools with their schemas
	tools := []*mcp.Tool{
		{
			Name:        "upload_content",
			Description: "Upload content with data in a single operation",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"owner_id": map[string]interface{}{
						"type":        "string",
						"format":      "uuid",
						"description": "Owner UUID",
					},
					"tenant_id": map[string]interface{}{
						"type":        "string",
						"format":      "uuid",
						"description": "Tenant UUID (optional)",
					},
					"name": map[string]interface{}{
						"type":        "string",
						"description": "Content name",
					},
					"description": map[string]interface{}{
						"type":        "string",
						"description": "Content description",
					},
					"document_type": map[string]interface{}{
						"type":        "string",
						"description": "MIME type of the content",
					},
					"storage_backend": map[string]interface{}{
						"type":        "string",
						"description": "Storage backend name (default if empty)",
						"default":     "default",
					},
					"data": map[string]interface{}{
						"type":        "string",
						"description": "Base64 encoded data or URL to download from",
					},
					"file_name": map[string]interface{}{
						"type":        "string",
						"description": "Original file name",
					},
					"tags": map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{
							"type": "string",
						},
						"description": "Tags for categorization",
					},
					"metadata": map[string]interface{}{
						"type":        "object",
						"description": "Custom metadata",
					},
				},
				"required": []string{"owner_id", "name", "data"},
			},
		},
		{
			Name:        "get_content",
			Description: "Retrieve content metadata by ID",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"content_id": map[string]interface{}{
						"type":        "string",
						"format":      "uuid",
						"description": "Content ID",
					},
				},
				"required": []string{"content_id"},
			},
		},
		{
			Name:        "get_content_details",
			Description: "Get complete content information including URLs and metadata",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"content_id": map[string]interface{}{
						"type":        "string",
						"format":      "uuid",
						"description": "Content ID",
					},
					"include_upload_url": map[string]interface{}{
						"type":        "boolean",
						"description": "Include presigned upload URL",
						"default":     false,
					},
				},
				"required": []string{"content_id"},
			},
		},
		{
			Name:        "list_content",
			Description: "List content with filtering and pagination",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"owner_id": map[string]interface{}{
						"type":        "string",
						"format":      "uuid",
						"description": "Filter by owner ID",
					},
					"tenant_id": map[string]interface{}{
						"type":        "string",
						"format":      "uuid",
						"description": "Filter by tenant ID",
					},
					"status": map[string]interface{}{
						"type":        "string",
						"description": "Filter by status",
					},
					"tags": map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{
							"type": "string",
						},
						"description": "Filter by tags",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum number of results",
						"default":     50,
						"maximum":     1000,
					},
					"offset": map[string]interface{}{
						"type":        "integer",
						"description": "Offset for pagination",
						"default":     0,
					},
				},
			},
		},
		{
			Name:        "download_content",
			Description: "Download content data (returns download URL or base64)",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"content_id": map[string]interface{}{
						"type":        "string",
						"format":      "uuid",
						"description": "Content ID",
					},
					"format": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"url", "base64"},
						"description": "Return format",
						"default":     "url",
					},
				},
				"required": []string{"content_id"},
			},
		},
		{
			Name:        "update_content",
			Description: "Update content metadata",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"content_id": map[string]interface{}{
						"type":        "string",
						"format":      "uuid",
						"description": "Content ID",
					},
					"name": map[string]interface{}{
						"type":        "string",
						"description": "New content name",
					},
					"description": map[string]interface{}{
						"type":        "string",
						"description": "New content description",
					},
					"tags": map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{
							"type": "string",
						},
						"description": "New tags",
					},
					"metadata": map[string]interface{}{
						"type":        "object",
						"description": "New custom metadata",
					},
				},
				"required": []string{"content_id"},
			},
		},
		{
			Name:        "delete_content",
			Description: "Soft delete content",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"content_id": map[string]interface{}{
						"type":        "string",
						"format":      "uuid",
						"description": "Content ID",
					},
				},
				"required": []string{"content_id"},
			},
		},
		{
			Name:        "search_content",
			Description: "Search content by metadata, tags, or full-text",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Search query",
					},
					"owner_id": map[string]interface{}{
						"type":        "string",
						"format":      "uuid",
						"description": "Filter by owner ID",
					},
					"tenant_id": map[string]interface{}{
						"type":        "string",
						"format":      "uuid",
						"description": "Filter by tenant ID",
					},
					"tags": map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{
							"type": "string",
						},
						"description": "Filter by tags",
					},
					"status": map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{
							"type": "string",
						},
						"description": "Filter by status values",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum number of results",
						"default":     50,
					},
					"offset": map[string]interface{}{
						"type":        "integer",
						"description": "Offset for pagination",
						"default":     0,
					},
				},
			},
		},
	}

	// Register each tool with its handler
	for _, tool := range tools {
		handler := s.getToolHandler(tool.Name)
		if handler == nil {
			return &ConfigError{Field: "tools", Message: "no handler for tool: " + tool.Name}
		}
		s.mcpServer.AddTool(tool, handler)
	}

	return nil
}

// getToolHandler returns the handler function for a tool by name
func (s *Server) getToolHandler(name string) mcp.ToolHandler {
	switch name {
	case "upload_content":
		return s.handleUploadContent
	case "get_content":
		return s.handleGetContent
	case "get_content_details":
		return s.handleGetContentDetails
	case "list_content":
		return s.handleListContent
	case "download_content":
		return s.handleDownloadContent
	case "update_content":
		return s.handleUpdateContent
	case "delete_content":
		return s.handleDeleteContent
	case "search_content":
		return s.handleSearchContent
	default:
		return nil
	}
}
