package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/tendant/simple-content/pkg/simplecontent"
	mcperrors "github.com/tendant/simple-content-mcp/pkg/mcpserver/errors"
)

// handleGetContentStatus checks content processing status
func (s *Server) handleGetContentStatus(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Unmarshal arguments
	var params map[string]interface{}
	if err := json.Unmarshal(req.Params.Arguments, &params); err != nil {
		return nil, mcperrors.NewValidationError("arguments", err)
	}

	// Parse content_id
	contentID, err := parseUUID(params["content_id"])
	if err != nil {
		return nil, mcperrors.NewValidationError("content_id", err)
	}

	// Get content
	content, err := s.service.GetContent(ctx, contentID)
	if err != nil {
		return nil, s.mapError(err)
	}

	// Check for derived content (thumbnails, previews)
	hasThumbnails := false
	hasPreviews := false

	derivedList, err := s.service.ListDerivedContent(ctx,
		simplecontent.WithParentID(contentID),
	)
	if err == nil && len(derivedList) > 0 {
		for _, derived := range derivedList {
			if derived.DerivationType == "thumbnail" {
				hasThumbnails = true
			} else if derived.DerivationType == "preview" {
				hasPreviews = true
			}
		}
	}

	// Determine ready state
	ready := content.Status == "uploaded" || content.Status == "processed"

	// Format result
	result := map[string]interface{}{
		"id":             content.ID.String(),
		"status":         content.Status,
		"ready":          ready,
		"has_thumbnails": hasThumbnails,
		"has_previews":   hasPreviews,
		"updated_at":     content.UpdatedAt,
	}

	return newTextResult(formatJSON(result)), nil
}

// handleListByStatus lists content by lifecycle status
func (s *Server) handleListByStatus(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Unmarshal arguments
	var params map[string]interface{}
	if err := json.Unmarshal(req.Params.Arguments, &params); err != nil {
		return nil, mcperrors.NewValidationError("arguments", err)
	}

	// Parse status (required)
	statusStr, ok := params["status"].(string)
	if !ok || statusStr == "" {
		return nil, mcperrors.NewValidationError("status", fmt.Errorf("status is required"))
	}

	// Parse and validate status
	status, err := simplecontent.ParseContentStatus(statusStr)
	if err != nil {
		return nil, mcperrors.NewValidationError("status", err)
	}

	// Call service
	contentList, err := s.service.GetContentByStatus(ctx, status)
	if err != nil {
		return nil, s.mapError(err)
	}

	// Optional: filter by owner_id
	if ownerIDRaw, ok := params["owner_id"]; ok {
		if ownerID, err := parseUUID(ownerIDRaw); err == nil {
			filtered := make([]*simplecontent.Content, 0)
			for _, c := range contentList {
				if c.OwnerID == ownerID {
					filtered = append(filtered, c)
				}
			}
			contentList = filtered
		}
	}

	// Apply limit
	limit := getIntOr(params, "limit", 100)
	if len(contentList) > limit {
		contentList = contentList[:limit]
	}

	// Format result
	items := make([]map[string]interface{}, len(contentList))
	for i, content := range contentList {
		items[i] = map[string]interface{}{
			"id":         content.ID.String(),
			"owner_id":   content.OwnerID.String(),
			"name":       content.Name,
			"status":     content.Status,
			"created_at": content.CreatedAt,
			"updated_at": content.UpdatedAt,
		}
	}

	result := map[string]interface{}{
		"status": statusStr,
		"items":  items,
		"count":  len(items),
	}

	return newTextResult(formatJSON(result)), nil
}
