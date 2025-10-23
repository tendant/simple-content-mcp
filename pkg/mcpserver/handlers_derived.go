package mcpserver

import (
	"context"
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/tendant/simple-content/pkg/simplecontent"
	mcperrors "github.com/tendant/simple-content-mcp/pkg/mcpserver/errors"
)

// handleListDerivedContent lists derived content (thumbnails, previews) for a parent content
func (s *Server) handleListDerivedContent(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Unmarshal arguments
	var params map[string]interface{}
	if err := json.Unmarshal(req.Params.Arguments, &params); err != nil {
		return nil, mcperrors.NewValidationError("arguments", err)
	}

	// Parse parent_id (required)
	parentID, err := parseUUID(params["parent_id"])
	if err != nil {
		return nil, mcperrors.NewValidationError("parent_id", err)
	}

	// Build list options
	var options []simplecontent.ListDerivedContentOption

	// Always include parent_id
	options = append(options, simplecontent.WithParentID(parentID))

	// Add URLs to results
	options = append(options, simplecontent.WithURLs())

	// Optional: derivation_type filter
	if derivationType, ok := params["derivation_type"].(string); ok && derivationType != "" {
		options = append(options, simplecontent.WithDerivationType(derivationType))
	}

	// Optional: variant filter
	if variant, ok := params["variant"].(string); ok && variant != "" {
		options = append(options, simplecontent.WithVariant(variant))
	}

	// Optional: variants filter (array)
	if variantsRaw, ok := params["variants"]; ok {
		if variantsArray, ok := variantsRaw.([]interface{}); ok {
			variants := make([]string, 0, len(variantsArray))
			for _, v := range variantsArray {
				if vStr, ok := v.(string); ok {
					variants = append(variants, vStr)
				}
			}
			if len(variants) > 0 {
				options = append(options, simplecontent.WithVariants(variants...))
			}
		}
	}

	// Pagination
	limit := getIntOr(params, "limit", s.config.DefaultPageSize)
	offset := getIntOr(params, "offset", 0)
	options = append(options, simplecontent.WithPagination(limit, offset))

	// Call service
	derivedList, err := s.service.ListDerivedContent(ctx, options...)
	if err != nil {
		return nil, s.mapError(err)
	}

	// Count total (if needed for pagination)
	// Note: CountDerivedContent is a helper function, not a service method
	// For now, we'll just return what we have

	// Format result
	items := make([]map[string]interface{}, len(derivedList))
	for i, derived := range derivedList {
		items[i] = map[string]interface{}{
			"parent_id":       derived.ParentID.String(),
			"content_id":      derived.ContentID.String(),
			"derivation_type": derived.DerivationType,
			"variant":         derived.Variant,
			"status":          derived.Status,
			"download_url":    derived.DownloadURL,
			"thumbnail_url":   derived.ThumbnailURL,
			"preview_url":     derived.PreviewURL,
			"document_type":   derived.DocumentType,
			"created_at":      derived.CreatedAt,
			"updated_at":      derived.UpdatedAt,
		}
	}

	result := map[string]interface{}{
		"items":  items,
		"count":  len(items),
		"limit":  limit,
		"offset": offset,
	}

	return newTextResult(formatJSON(result)), nil
}

// handleGetThumbnails gets thumbnails by size (convenience wrapper)
func (s *Server) handleGetThumbnails(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Unmarshal arguments
	var params map[string]interface{}
	if err := json.Unmarshal(req.Params.Arguments, &params); err != nil {
		return nil, mcperrors.NewValidationError("arguments", err)
	}

	// Parse parent_id (required)
	parentID, err := parseUUID(params["parent_id"])
	if err != nil {
		return nil, mcperrors.NewValidationError("parent_id", err)
	}

	// Parse sizes array (optional)
	var sizes []string
	if sizesRaw, ok := params["sizes"]; ok {
		if sizesArray, ok := sizesRaw.([]interface{}); ok {
			for _, s := range sizesArray {
				if sStr, ok := s.(string); ok {
					sizes = append(sizes, sStr)
				}
			}
		}
	}

	// If no sizes specified, use common defaults
	if len(sizes) == 0 {
		sizes = []string{"256", "512", "720", "1024"}
	}

	// Call helper function
	thumbnails, err := simplecontent.GetThumbnailsBySize(ctx, s.service, parentID, sizes)
	if err != nil {
		return nil, s.mapError(err)
	}

	// Format result as map of size -> thumbnail data
	result := map[string]interface{}{
		"parent_id":  parentID.String(),
		"thumbnails": make(map[string]interface{}),
	}

	thumbnailsMap := result["thumbnails"].(map[string]interface{})
	for _, thumb := range thumbnails {
		// Extract size from variant (e.g., "thumbnail_256" -> "256")
		size := thumb.Variant
		if len(thumb.Variant) > 10 && thumb.Variant[:10] == "thumbnail_" {
			size = thumb.Variant[10:]
		}

		thumbnailsMap[size] = map[string]interface{}{
			"content_id":    thumb.ContentID.String(),
			"variant":       thumb.Variant,
			"download_url":  thumb.DownloadURL,
			"thumbnail_url": thumb.ThumbnailURL,
			"status":        thumb.Status,
			"document_type": thumb.DocumentType,
			"created_at":    thumb.CreatedAt,
		}
	}

	return newTextResult(formatJSON(result)), nil
}
