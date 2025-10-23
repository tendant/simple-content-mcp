package mcpserver

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// registerPrompts registers all MCP prompts with the server
func (s *Server) registerPrompts() error {
	if !s.config.EnablePrompts {
		return nil
	}

	prompts := []*mcp.Prompt{
		{
			Name:        "upload-workflow",
			Description: "Step-by-step guide for uploading content",
			Arguments: []*mcp.PromptArgument{
				{
					Name:        "content_type",
					Description: "Type of content being uploaded (optional)",
					Required:    false,
				},
			},
		},
		{
			Name:        "search-workflow",
			Description: "Guide for searching and filtering content",
			Arguments: []*mcp.PromptArgument{
				{
					Name:        "search_type",
					Description: "Type of search: tags, metadata, status (optional)",
					Required:    false,
				},
			},
		},
		{
			Name:        "derived-content-workflow",
			Description: "Guide for working with derived content (thumbnails, previews)",
		},
		{
			Name:        "status-monitoring",
			Description: "Guide for monitoring content status and processing",
		},
	}

	for _, prompt := range prompts {
		handler := s.getPromptHandler(prompt.Name)
		if handler == nil {
			return &ConfigError{Field: "prompts", Message: "no handler for prompt: " + prompt.Name}
		}
		s.mcpServer.AddPrompt(prompt, handler)
	}

	return nil
}

// getPromptHandler returns the handler function for a prompt by name
func (s *Server) getPromptHandler(name string) mcp.PromptHandler {
	switch name {
	case "upload-workflow":
		return s.handleUploadWorkflowPrompt
	case "search-workflow":
		return s.handleSearchWorkflowPrompt
	case "derived-content-workflow":
		return s.handleDerivedContentWorkflowPrompt
	case "status-monitoring":
		return s.handleStatusMonitoringPrompt
	default:
		return nil
	}
}

// handleUploadWorkflowPrompt returns guidance for uploading content
func (s *Server) handleUploadWorkflowPrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	contentType := getArgumentValue(req.Params.Arguments, "content_type", "any")

	message := fmt.Sprintf(`To upload %s content to the system:

1. Prepare your content data
   - For binary content: Encode as base64
   - For URLs: Use the URL directly

2. Call the upload_content tool with:
   - owner_id: Your user/organization UUID
   - name: Descriptive name for the content
   - data: Base64 encoded content or URL
   - file_name: Original filename (optional but recommended)
   - tags: Array of tags for categorization (optional)

3. The tool returns:
   - content_id: Use this for all future operations
   - download_url: Direct download link
   - status: Should be "uploaded" for successful uploads

Example:
{
  "owner_id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "Company Logo",
  "data": "iVBORw0KGgoAAAANS...",
  "file_name": "logo.png",
  "tags": ["branding", "images"]
}

You can verify the upload with get_content_status or get_content_details.`, contentType)

	return &mcp.GetPromptResult{
		Description: "Guide for uploading content",
		Messages: []*mcp.PromptMessage{
			{
				Role: "user",
				Content: &mcp.TextContent{
					Text: message,
				},
			},
		},
	}, nil
}

// handleSearchWorkflowPrompt returns guidance for searching content
func (s *Server) handleSearchWorkflowPrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	searchType := getArgumentValue(req.Params.Arguments, "search_type", "general")

	var message string
	switch searchType {
	case "tags":
		message = `To search content by tags:

Use the search_content tool with:
- owner_id: Filter by owner (optional)
- tags: Array of tags to match

Example:
{
  "owner_id": "550e8400-e29b-41d4-a716-446655440000",
  "tags": ["invoice", "2024"]
}

This returns all content tagged with both "invoice" and "2024".`

	case "status":
		message = `To search content by status:

Use the list_by_status tool with:
- status: One of: created, uploading, uploaded, processing, processed, failed, archived
- owner_id: Filter by owner (optional)
- limit: Maximum results (default 100)

Example:
{
  "status": "failed",
  "limit": 50
}

This is useful for monitoring and finding content that needs attention.`

	default:
		message = `To search for content, you have several options:

1. By Tags:
   - Use search_content with tags array
   - Example: {"tags": ["invoice", "2024"]}

2. By Status:
   - Use list_by_status with desired status
   - Useful for finding failed uploads or pending processing
   - Example: {"status": "uploaded"}

3. By Query:
   - Use search_content with query string
   - Searches in name, description, and metadata fields
   - Example: {"query": "logo", "owner_id": "..."}

4. List All (with filters):
   - Use list_content with owner_id, tenant_id, etc.
   - Supports pagination with limit/offset

Combine tools with get_content_details to get full information including URLs.`
	}

	return &mcp.GetPromptResult{
		Description: "Guide for searching content",
		Messages: []*mcp.PromptMessage{
			{
				Role: "user",
				Content: &mcp.TextContent{
					Text: message,
				},
			},
		},
	}, nil
}

// handleDerivedContentWorkflowPrompt returns guidance for working with derived content
func (s *Server) handleDerivedContentWorkflowPrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	message := `Working with derived content (thumbnails, previews):

1. List All Derived Content:
   Use list_derived_content tool:
   - parent_id: The original content UUID
   - derivation_type: Filter by "thumbnail" or "preview" (optional)
   - variant: Specific variant like "thumbnail_256" (optional)

   Example:
   {
     "parent_id": "550e8400-e29b-41d4-a716-446655440000"
   }

2. Get Thumbnails by Size (convenience):
   Use get_thumbnails tool:
   - parent_id: The original image UUID
   - sizes: Array like ["256", "512", "720"] (optional, defaults to common sizes)

   Example:
   {
     "parent_id": "550e8400-e29b-41d4-a716-446655440000",
     "sizes": ["256", "512"]
   }

   Returns a map of size → thumbnail data with download URLs.

3. Check Availability:
   Use get_content_status to check if thumbnails/previews exist:
   {
     "content_id": "550e8400-e29b-41d4-a716-446655440000"
   }

   Returns has_thumbnails and has_previews flags.

Note: Derived content is typically generated by background workers.
Agents primarily consume (read) derived content, not create it.`

	return &mcp.GetPromptResult{
		Description: "Guide for working with derived content",
		Messages: []*mcp.PromptMessage{
			{
				Role: "user",
				Content: &mcp.TextContent{
					Text: message,
				},
			},
		},
	}, nil
}

// handleStatusMonitoringPrompt returns guidance for status monitoring
func (s *Server) handleStatusMonitoringPrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	message := `Monitoring content status and processing:

1. Check Single Content Status:
   Use get_content_status:
   {
     "content_id": "550e8400-e29b-41d4-a716-446655440000"
   }

   Returns:
   - status: Current lifecycle status
   - ready: Boolean indicating if content is ready to use
   - has_thumbnails: Boolean for thumbnail availability
   - has_previews: Boolean for preview availability

2. List Content by Status:
   Use list_by_status to find all content in a specific state:
   {
     "status": "processing",
     "owner_id": "550e8400-e29b-41d4-a716-446655440000",
     "limit": 100
   }

   Useful statuses to monitor:
   - "failed": Content that failed processing
   - "processing": Currently being processed
   - "uploaded": Ready and available
   - "created": Awaiting processing

3. Polling Pattern:
   For async operations, poll get_content_status until ready=true:

   while (!ready) {
     result = get_content_status(content_id)
     if (result.ready) break
     if (result.status == "failed") handle_error()
     wait(5 seconds)
   }

4. System Overview:
   Read the stats://system resource for aggregate statistics:
   - Total content count
   - Breakdown by status
   - Identify bottlenecks or issues

Status Lifecycle:
created → uploading → uploaded → processing → processed
                               ↓
                            failed`

	return &mcp.GetPromptResult{
		Description: "Guide for monitoring content status",
		Messages: []*mcp.PromptMessage{
			{
				Role: "user",
				Content: &mcp.TextContent{
					Text: message,
				},
			},
		},
	}, nil
}

// Helper to get argument value with default
func getArgumentValue(args map[string]string, key string, defaultValue string) string {
	if val, ok := args[key]; ok {
		return val
	}
	return defaultValue
}
