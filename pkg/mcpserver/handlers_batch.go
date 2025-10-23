package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/tendant/simple-content/pkg/simplecontent"
	mcperrors "github.com/tendant/simple-content-mcp/pkg/mcpserver/errors"
)

// BatchUploadItem represents a single item in a batch upload request
type BatchUploadItem struct {
	Name           string                 `json:"name"`
	Data           string                 `json:"data"`
	FileName       string                 `json:"file_name,omitempty"`
	Description    string                 `json:"description,omitempty"`
	DocumentType   string                 `json:"document_type,omitempty"`
	Tags           []string               `json:"tags,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	StorageBackend string                 `json:"storage_backend,omitempty"`
}

// BatchUploadResult represents the result of a single upload in a batch
type BatchUploadResult struct {
	Index     int    `json:"index"`
	Success   bool   `json:"success"`
	ContentID string `json:"content_id,omitempty"`
	Error     string `json:"error,omitempty"`
}

// handleBatchUpload handles batch upload of multiple content items
func (s *Server) handleBatchUpload(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Unmarshal arguments
	var params map[string]interface{}
	if err := json.Unmarshal(req.Params.Arguments, &params); err != nil {
		return nil, mcperrors.NewValidationError("arguments", err)
	}

	// Parse owner_id (required)
	ownerID, err := parseUUID(params["owner_id"])
	if err != nil {
		return nil, mcperrors.NewValidationError("owner_id", err)
	}

	// Parse optional tenant_id
	var tenantID uuid.UUID
	if tenantIDRaw, ok := params["tenant_id"]; ok {
		tenantID, err = parseUUID(tenantIDRaw)
		if err != nil {
			return nil, mcperrors.NewValidationError("tenant_id", err)
		}
	}

	// Parse items array
	itemsRaw, ok := params["items"]
	if !ok {
		return nil, mcperrors.NewValidationError("items", fmt.Errorf("items array is required"))
	}

	itemsArray, ok := itemsRaw.([]interface{})
	if !ok {
		return nil, mcperrors.NewValidationError("items", fmt.Errorf("items must be an array"))
	}

	if len(itemsArray) == 0 {
		return nil, mcperrors.NewValidationError("items", fmt.Errorf("items array cannot be empty"))
	}

	if len(itemsArray) > s.config.MaxBatchSize {
		return nil, mcperrors.NewValidationError("items", fmt.Errorf("batch size %d exceeds maximum %d", len(itemsArray), s.config.MaxBatchSize))
	}

	// Parse each item
	items := make([]BatchUploadItem, len(itemsArray))
	for i, itemRaw := range itemsArray {
		itemMap, ok := itemRaw.(map[string]interface{})
		if !ok {
			return nil, mcperrors.NewValidationError(fmt.Sprintf("items[%d]", i), fmt.Errorf("must be an object"))
		}

		// Marshal and unmarshal to convert to BatchUploadItem
		itemJSON, _ := json.Marshal(itemMap)
		if err := json.Unmarshal(itemJSON, &items[i]); err != nil {
			return nil, mcperrors.NewValidationError(fmt.Sprintf("items[%d]", i), err)
		}

		// Validate required fields
		if items[i].Name == "" {
			return nil, mcperrors.NewValidationError(fmt.Sprintf("items[%d].name", i), fmt.Errorf("name is required"))
		}
		if items[i].Data == "" {
			return nil, mcperrors.NewValidationError(fmt.Sprintf("items[%d].data", i), fmt.Errorf("data is required"))
		}
	}

	// Process uploads in parallel
	results := make([]BatchUploadResult, len(items))
	var wg sync.WaitGroup
	var mu sync.Mutex

	for i, item := range items {
		wg.Add(1)
		go func(index int, uploadItem BatchUploadItem) {
			defer wg.Done()

			// Decode data
			reader, err := s.decodeData(uploadItem.Data)
			if err != nil {
				mu.Lock()
				results[index] = BatchUploadResult{
					Index:   index,
					Success: false,
					Error:   fmt.Sprintf("failed to decode data: %v", err),
				}
				mu.Unlock()
				return
			}

			// Build upload request
			uploadReq := simplecontent.UploadContentRequest{
				OwnerID:        ownerID,
				TenantID:       tenantID,
				Name:           uploadItem.Name,
				Description:    uploadItem.Description,
				DocumentType:   uploadItem.DocumentType,
				Reader:         reader,
				FileName:       uploadItem.FileName,
				Tags:           uploadItem.Tags,
				CustomMetadata: uploadItem.Metadata,
			}

			if uploadItem.StorageBackend != "" {
				uploadReq.StorageBackendName = uploadItem.StorageBackend
			} else {
				uploadReq.StorageBackendName = "default"
			}

			// Upload content
			content, err := s.service.UploadContent(ctx, uploadReq)
			if err != nil {
				mu.Lock()
				results[index] = BatchUploadResult{
					Index:   index,
					Success: false,
					Error:   fmt.Sprintf("upload failed: %v", err),
				}
				mu.Unlock()
				return
			}

			mu.Lock()
			results[index] = BatchUploadResult{
				Index:     index,
				Success:   true,
				ContentID: content.ID.String(),
			}
			mu.Unlock()
		}(i, item)
	}

	// Wait for all uploads to complete
	wg.Wait()

	// Count successes and failures
	successful := 0
	failed := 0
	for _, result := range results {
		if result.Success {
			successful++
		} else {
			failed++
		}
	}

	// Format result
	responseData := map[string]interface{}{
		"results":    results,
		"total":      len(results),
		"successful": successful,
		"failed":     failed,
	}

	return newTextResult(formatJSON(responseData)), nil
}

// handleBatchGetDetails handles batch retrieval of content details
func (s *Server) handleBatchGetDetails(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Unmarshal arguments
	var params map[string]interface{}
	if err := json.Unmarshal(req.Params.Arguments, &params); err != nil {
		return nil, mcperrors.NewValidationError("arguments", err)
	}

	// Parse content_ids array
	idsRaw, ok := params["content_ids"]
	if !ok {
		return nil, mcperrors.NewValidationError("content_ids", fmt.Errorf("content_ids array is required"))
	}

	idsArray, ok := idsRaw.([]interface{})
	if !ok {
		return nil, mcperrors.NewValidationError("content_ids", fmt.Errorf("content_ids must be an array"))
	}

	if len(idsArray) == 0 {
		return nil, mcperrors.NewValidationError("content_ids", fmt.Errorf("content_ids array cannot be empty"))
	}

	if len(idsArray) > s.config.MaxBatchSize {
		return nil, mcperrors.NewValidationError("content_ids", fmt.Errorf("batch size %d exceeds maximum %d", len(idsArray), s.config.MaxBatchSize))
	}

	// Parse each ID
	contentIDs := make([]uuid.UUID, len(idsArray))
	for i, idRaw := range idsArray {
		contentID, err := parseUUID(idRaw)
		if err != nil {
			return nil, mcperrors.NewValidationError(fmt.Sprintf("content_ids[%d]", i), err)
		}
		contentIDs[i] = contentID
	}

	// Fetch details in parallel
	type detailResult struct {
		Index   int
		Details map[string]interface{}
		Error   string
	}

	results := make([]detailResult, len(contentIDs))
	var wg sync.WaitGroup
	var mu sync.Mutex

	for i, contentID := range contentIDs {
		wg.Add(1)
		go func(index int, id uuid.UUID) {
			defer wg.Done()

			// Get content details
			details, err := s.service.GetContentDetails(ctx, id)
			if err != nil {
				mu.Lock()
				results[index] = detailResult{
					Index: index,
					Error: fmt.Sprintf("failed to get details: %v", err),
				}
				mu.Unlock()
				return
			}

			// Format details
			detailsMap := map[string]interface{}{
				"id":           details.ID,
				"download_url": details.Download,
				"file_name":    details.FileName,
				"file_size":    details.FileSize,
				"mime_type":    details.MimeType,
				"ready":        details.Ready,
				"created_at":   details.CreatedAt,
				"updated_at":   details.UpdatedAt,
			}

			mu.Lock()
			results[index] = detailResult{
				Index:   index,
				Details: detailsMap,
			}
			mu.Unlock()
		}(i, contentID)
	}

	// Wait for all fetches to complete
	wg.Wait()

	// Count successes and failures
	successful := 0
	failed := 0
	for _, result := range results {
		if result.Error == "" {
			successful++
		} else {
			failed++
		}
	}

	// Format result
	responseData := map[string]interface{}{
		"results":    results,
		"total":      len(results),
		"successful": successful,
		"failed":     failed,
	}

	return newTextResult(formatJSON(responseData)), nil
}
