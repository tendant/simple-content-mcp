package mcpserver

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/tendant/simple-content/pkg/simplecontent"
	memoryrepo "github.com/tendant/simple-content/pkg/simplecontent/repo/memory"
	memorystorage "github.com/tendant/simple-content/pkg/simplecontent/storage/memory"
)

// createTestService creates a service with in-memory backends for testing
func createTestService(t *testing.T) simplecontent.Service {
	repo := memoryrepo.New()
	blobStore := memorystorage.New()

	service, err := simplecontent.New(
		simplecontent.WithRepository(repo),
		simplecontent.WithBlobStore("default", blobStore),
	)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	return service
}

// createTestServer creates a server for testing
func createTestServer(t *testing.T) *Server {
	service := createTestService(t)
	config := DefaultConfig(service)

	server, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	return server
}

func TestServerCreation(t *testing.T) {
	service := createTestService(t)
	config := DefaultConfig(service)

	server, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	if server == nil {
		t.Fatal("Server is nil")
	}

	if server.service == nil {
		t.Fatal("Service is nil")
	}

	if server.mcpServer == nil {
		t.Fatal("MCP server is nil")
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name      string
		config    Config
		wantError bool
	}{
		{
			name: "valid config",
			config: Config{
				Service:         createTestService(t),
				Name:            "test-server",
				Version:         "0.1.0",
				Mode:            TransportStdio,
				MaxBatchSize:    100,
				DefaultPageSize: 50,
				MaxPageSize:     1000,
			},
			wantError: false,
		},
		{
			name: "missing service",
			config: Config{
				Name:            "test-server",
				Version:         "0.1.0",
				MaxBatchSize:    100,
				DefaultPageSize: 50,
				MaxPageSize:     1000,
			},
			wantError: true,
		},
		{
			name: "missing name",
			config: Config{
				Service:         createTestService(t),
				Version:         "0.1.0",
				MaxBatchSize:    100,
				DefaultPageSize: 50,
				MaxPageSize:     1000,
			},
			wantError: true,
		},
		{
			name: "invalid page size",
			config: Config{
				Service:         createTestService(t),
				Name:            "test-server",
				Version:         "0.1.0",
				MaxBatchSize:    100,
				DefaultPageSize: 100,
				MaxPageSize:     50,
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantError {
				t.Errorf("Validate() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestUploadContentTool(t *testing.T) {
	server := createTestServer(t)
	ctx := context.Background()

	ownerID := uuid.New()
	testData := "Hello, World!"
	encodedData := base64.StdEncoding.EncodeToString([]byte(testData))

	args := map[string]interface{}{
		"owner_id":  ownerID.String(),
		"name":      "test.txt",
		"data":      encodedData,
		"file_name": "test.txt",
	}

	argsJSON, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("Failed to marshal args: %v", err)
	}

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "upload_content",
			Arguments: argsJSON,
		},
	}

	result, err := server.handleUploadContent(ctx, req)
	if err != nil {
		t.Fatalf("handleUploadContent failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result is nil")
	}

	if len(result.Content) == 0 {
		t.Fatal("Result content is empty")
	}

	// Verify the result contains an ID
	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("Result content is not TextContent")
	}

	var resultData map[string]interface{}
	if err := json.Unmarshal([]byte(textContent.Text), &resultData); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	if resultData["id"] == nil {
		t.Fatal("Result does not contain ID")
	}

	if resultData["status"] != "uploaded" {
		t.Errorf("Expected status 'uploaded', got %v", resultData["status"])
	}
}

func TestGetContentTool(t *testing.T) {
	server := createTestServer(t)
	ctx := context.Background()

	// First upload a content
	ownerID := uuid.New()
	uploadArgs := map[string]interface{}{
		"owner_id": ownerID.String(),
		"name":     "test.txt",
		"data":     base64.StdEncoding.EncodeToString([]byte("test data")),
	}

	uploadArgsJSON, _ := json.Marshal(uploadArgs)
	uploadReq := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "upload_content",
			Arguments: uploadArgsJSON,
		},
	}

	uploadResult, err := server.handleUploadContent(ctx, uploadReq)
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}

	// Extract content ID
	textContent := uploadResult.Content[0].(*mcp.TextContent)
	var uploadData map[string]interface{}
	json.Unmarshal([]byte(textContent.Text), &uploadData)
	contentID := uploadData["id"].(string)

	// Now get the content
	getArgs := map[string]interface{}{
		"content_id": contentID,
	}

	getArgsJSON, _ := json.Marshal(getArgs)
	getReq := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "get_content",
			Arguments: getArgsJSON,
		},
	}

	result, err := server.handleGetContent(ctx, getReq)
	if err != nil {
		t.Fatalf("handleGetContent failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result is nil")
	}

	// Verify the result
	textContent = result.Content[0].(*mcp.TextContent)
	var resultData map[string]interface{}
	json.Unmarshal([]byte(textContent.Text), &resultData)

	if resultData["id"] != contentID {
		t.Errorf("Expected ID %s, got %v", contentID, resultData["id"])
	}

	if resultData["name"] != "test.txt" {
		t.Errorf("Expected name 'test.txt', got %v", resultData["name"])
	}
}

func TestListContentTool(t *testing.T) {
	server := createTestServer(t)
	ctx := context.Background()

	ownerID := uuid.New()

	// Upload some test content
	for i := 0; i < 3; i++ {
		args := map[string]interface{}{
			"owner_id": ownerID.String(),
			"name":     "test" + string(rune('0'+i)) + ".txt",
			"data":     base64.StdEncoding.EncodeToString([]byte("test data")),
		}

		argsJSON, _ := json.Marshal(args)
		req := &mcp.CallToolRequest{
			Params: &mcp.CallToolParamsRaw{
				Name:      "upload_content",
				Arguments: argsJSON,
			},
		}

		_, err := server.handleUploadContent(ctx, req)
		if err != nil {
			t.Fatalf("Upload failed: %v", err)
		}
	}

	// Now list content
	listArgs := map[string]interface{}{
		"owner_id": ownerID.String(),
		"limit":    10,
		"offset":   0,
	}

	listArgsJSON, _ := json.Marshal(listArgs)
	listReq := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "list_content",
			Arguments: listArgsJSON,
		},
	}

	result, err := server.handleListContent(ctx, listReq)
	if err != nil {
		t.Fatalf("handleListContent failed: %v", err)
	}

	// Verify the result
	textContent := result.Content[0].(*mcp.TextContent)
	var resultData map[string]interface{}
	json.Unmarshal([]byte(textContent.Text), &resultData)

	items := resultData["items"].([]interface{})
	if len(items) != 3 {
		t.Errorf("Expected 3 items, got %d", len(items))
	}

	total := int(resultData["total"].(float64))
	if total != 3 {
		t.Errorf("Expected total 3, got %d", total)
	}
}

func TestListDerivedContentTool(t *testing.T) {
	server := createTestServer(t)
	ctx := context.Background()

	ownerID := uuid.New()

	// Upload a parent content (image)
	parentArgs := map[string]interface{}{
		"owner_id":  ownerID.String(),
		"name":      "parent-image.jpg",
		"data":      base64.StdEncoding.EncodeToString([]byte("parent image data")),
		"file_name": "parent-image.jpg",
	}

	parentArgsJSON, _ := json.Marshal(parentArgs)
	parentReq := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "upload_content",
			Arguments: parentArgsJSON,
		},
	}

	parentResult, err := server.handleUploadContent(ctx, parentReq)
	if err != nil {
		t.Fatalf("Parent upload failed: %v", err)
	}

	// Extract parent ID
	textContent := parentResult.Content[0].(*mcp.TextContent)
	var parentData map[string]interface{}
	json.Unmarshal([]byte(textContent.Text), &parentData)
	parentID := parentData["id"].(string)

	// Upload some derived content (thumbnails)
	for i, size := range []string{"256", "512"} {
		derivedArgs := map[string]interface{}{
			"parent_id":       parentID,
			"name":            "thumbnail_" + size,
			"data":            base64.StdEncoding.EncodeToString([]byte("thumbnail " + size)),
			"derivation_type": "thumbnail",
			"variant":         "thumbnail_" + size,
		}

		derivedArgsJSON, _ := json.Marshal(derivedArgs)
		derivedReq := &mcp.CallToolRequest{
			Params: &mcp.CallToolParamsRaw{
				Name:      "upload_content",
				Arguments: derivedArgsJSON,
			},
		}

		// Note: We're using upload_content, but need to create derived relationship
		// For proper testing, we'd need access to UploadDerivedContent or CreateDerivedContent
		// For now, we'll just test that list_derived_content doesn't error with no results
		_, _ = server.handleUploadContent(ctx, derivedReq)
		_ = i // avoid unused warning
	}

	// List derived content
	listArgs := map[string]interface{}{
		"parent_id": parentID,
	}

	listArgsJSON, _ := json.Marshal(listArgs)
	listReq := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "list_derived_content",
			Arguments: listArgsJSON,
		},
	}

	result, err := server.handleListDerivedContent(ctx, listReq)
	if err != nil {
		t.Fatalf("handleListDerivedContent failed: %v", err)
	}

	// Verify result structure
	textContent = result.Content[0].(*mcp.TextContent)
	var resultData map[string]interface{}
	json.Unmarshal([]byte(textContent.Text), &resultData)

	if resultData["items"] == nil {
		t.Error("Result does not contain items")
	}
}

func TestGetContentStatusTool(t *testing.T) {
	server := createTestServer(t)
	ctx := context.Background()

	ownerID := uuid.New()

	// Upload content
	uploadArgs := map[string]interface{}{
		"owner_id": ownerID.String(),
		"name":     "test.txt",
		"data":     base64.StdEncoding.EncodeToString([]byte("test data")),
	}

	uploadArgsJSON, _ := json.Marshal(uploadArgs)
	uploadReq := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "upload_content",
			Arguments: uploadArgsJSON,
		},
	}

	uploadResult, err := server.handleUploadContent(ctx, uploadReq)
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}

	// Extract content ID
	textContent := uploadResult.Content[0].(*mcp.TextContent)
	var uploadData map[string]interface{}
	json.Unmarshal([]byte(textContent.Text), &uploadData)
	contentID := uploadData["id"].(string)

	// Get content status
	statusArgs := map[string]interface{}{
		"content_id": contentID,
	}

	statusArgsJSON, _ := json.Marshal(statusArgs)
	statusReq := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "get_content_status",
			Arguments: statusArgsJSON,
		},
	}

	result, err := server.handleGetContentStatus(ctx, statusReq)
	if err != nil {
		t.Fatalf("handleGetContentStatus failed: %v", err)
	}

	// Verify result
	textContent = result.Content[0].(*mcp.TextContent)
	var resultData map[string]interface{}
	json.Unmarshal([]byte(textContent.Text), &resultData)

	if resultData["id"] != contentID {
		t.Errorf("Expected ID %s, got %v", contentID, resultData["id"])
	}

	if resultData["status"] != "uploaded" {
		t.Errorf("Expected status 'uploaded', got %v", resultData["status"])
	}

	if resultData["ready"] != true {
		t.Errorf("Expected ready to be true, got %v", resultData["ready"])
	}
}

func TestListByStatusTool(t *testing.T) {
	server := createTestServer(t)
	ctx := context.Background()

	ownerID := uuid.New()

	// Upload some content
	for i := 0; i < 2; i++ {
		args := map[string]interface{}{
			"owner_id": ownerID.String(),
			"name":     "test" + string(rune('0'+i)) + ".txt",
			"data":     base64.StdEncoding.EncodeToString([]byte("test data")),
		}

		argsJSON, _ := json.Marshal(args)
		req := &mcp.CallToolRequest{
			Params: &mcp.CallToolParamsRaw{
				Name:      "upload_content",
				Arguments: argsJSON,
			},
		}

		_, err := server.handleUploadContent(ctx, req)
		if err != nil {
			t.Fatalf("Upload failed: %v", err)
		}
	}

	// List by status
	listArgs := map[string]interface{}{
		"status": "uploaded",
	}

	listArgsJSON, _ := json.Marshal(listArgs)
	listReq := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "list_by_status",
			Arguments: listArgsJSON,
		},
	}

	result, err := server.handleListByStatus(ctx, listReq)
	if err != nil {
		t.Fatalf("handleListByStatus failed: %v", err)
	}

	// Verify result
	textContent := result.Content[0].(*mcp.TextContent)
	var resultData map[string]interface{}
	json.Unmarshal([]byte(textContent.Text), &resultData)

	items := resultData["items"].([]interface{})
	if len(items) < 2 {
		t.Errorf("Expected at least 2 items, got %d", len(items))
	}

	if resultData["status"] != "uploaded" {
		t.Errorf("Expected status 'uploaded', got %v", resultData["status"])
	}
}

func TestResourceContent(t *testing.T) {
	server := createTestServer(t)
	ctx := context.Background()

	ownerID := uuid.New()

	// Upload content
	uploadArgs := map[string]interface{}{
		"owner_id": ownerID.String(),
		"name":     "test-resource.txt",
		"data":     base64.StdEncoding.EncodeToString([]byte("test data")),
	}

	uploadArgsJSON, _ := json.Marshal(uploadArgs)
	uploadReq := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "upload_content",
			Arguments: uploadArgsJSON,
		},
	}

	uploadResult, err := server.handleUploadContent(ctx, uploadReq)
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}

	textContent := uploadResult.Content[0].(*mcp.TextContent)
	var uploadData map[string]interface{}
	json.Unmarshal([]byte(textContent.Text), &uploadData)
	contentID := uploadData["id"].(string)

	// Read resource: content://{id}
	resourceReq := &mcp.ReadResourceRequest{
		Params: &mcp.ReadResourceParams{
			URI: "content://" + contentID,
		},
	}

	result, err := server.handleContentResource(ctx, resourceReq)
	if err != nil {
		t.Fatalf("Read resource failed: %v", err)
	}

	if len(result.Contents) == 0 {
		t.Fatal("No resource contents returned")
	}

	if result.Contents[0].MIMEType != "application/json" {
		t.Errorf("Expected MIME type application/json, got %s", result.Contents[0].MIMEType)
	}

	// Verify JSON content
	var resourceData map[string]interface{}
	if err := json.Unmarshal([]byte(result.Contents[0].Text), &resourceData); err != nil {
		t.Fatalf("Failed to parse resource JSON: %v", err)
	}

	if resourceData["id"] != contentID {
		t.Errorf("Expected ID %s, got %v", contentID, resourceData["id"])
	}
}

func TestResourceSchema(t *testing.T) {
	server := createTestServer(t)
	ctx := context.Background()

	resourceReq := &mcp.ReadResourceRequest{
		Params: &mcp.ReadResourceParams{
			URI: "schema://content",
		},
	}

	result, err := server.handleContentSchemaResource(ctx, resourceReq)
	if err != nil {
		t.Fatalf("Read schema resource failed: %v", err)
	}

	if len(result.Contents) == 0 {
		t.Fatal("No resource contents returned")
	}

	if result.Contents[0].MIMEType != "application/schema+json" {
		t.Errorf("Expected MIME type application/schema+json, got %s", result.Contents[0].MIMEType)
	}

	// Verify it's valid JSON Schema
	var schema map[string]interface{}
	if err := json.Unmarshal([]byte(result.Contents[0].Text), &schema); err != nil {
		t.Fatalf("Failed to parse schema JSON: %v", err)
	}

	if schema["$schema"] == nil {
		t.Error("Schema missing $schema field")
	}
}

func TestResourceStats(t *testing.T) {
	server := createTestServer(t)
	ctx := context.Background()

	resourceReq := &mcp.ReadResourceRequest{
		Params: &mcp.ReadResourceParams{
			URI: "stats://system",
		},
	}

	result, err := server.handleSystemStatsResource(ctx, resourceReq)
	if err != nil {
		t.Fatalf("Read stats resource failed: %v", err)
	}

	if len(result.Contents) == 0 {
		t.Fatal("No resource contents returned")
	}

	// Verify JSON content
	var stats map[string]interface{}
	if err := json.Unmarshal([]byte(result.Contents[0].Text), &stats); err != nil {
		t.Fatalf("Failed to parse stats JSON: %v", err)
	}

	if stats["content_count"] == nil {
		t.Error("Stats missing content_count")
	}
}

func TestPromptUploadWorkflow(t *testing.T) {
	server := createTestServer(t)
	ctx := context.Background()

	promptReq := &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{
			Name:      "upload-workflow",
			Arguments: map[string]string{},
		},
	}

	result, err := server.handleUploadWorkflowPrompt(ctx, promptReq)
	if err != nil {
		t.Fatalf("Get prompt failed: %v", err)
	}

	if len(result.Messages) == 0 {
		t.Fatal("No prompt messages returned")
	}

	textContent, ok := result.Messages[0].Content.(*mcp.TextContent)
	if !ok {
		t.Fatal("Prompt message content is not TextContent")
	}

	if textContent.Text == "" {
		t.Error("Prompt message text is empty")
	}
}

func TestPromptSearchWorkflow(t *testing.T) {
	server := createTestServer(t)
	ctx := context.Background()

	promptReq := &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{
			Name: "search-workflow",
			Arguments: map[string]string{
				"search_type": "tags",
			},
		},
	}

	result, err := server.handleSearchWorkflowPrompt(ctx, promptReq)
	if err != nil {
		t.Fatalf("Get prompt failed: %v", err)
	}

	if len(result.Messages) == 0 {
		t.Fatal("No prompt messages returned")
	}

	textContent, ok := result.Messages[0].Content.(*mcp.TextContent)
	if !ok {
		t.Fatal("Prompt message content is not TextContent")
	}

	if !strings.Contains(textContent.Text, "tags") {
		t.Error("Prompt should contain information about tags search")
	}
}

func TestBatchUpload(t *testing.T) {
	server := createTestServer(t)
	ctx := context.Background()

	ownerID := uuid.New()

	// Prepare batch items
	items := []map[string]interface{}{
		{
			"name":      "batch-item-1.txt",
			"data":      base64.StdEncoding.EncodeToString([]byte("batch data 1")),
			"file_name": "batch-item-1.txt",
			"tags":      []string{"batch", "test"},
		},
		{
			"name":      "batch-item-2.txt",
			"data":      base64.StdEncoding.EncodeToString([]byte("batch data 2")),
			"file_name": "batch-item-2.txt",
			"tags":      []string{"batch", "test"},
		},
		{
			"name":      "batch-item-3.txt",
			"data":      base64.StdEncoding.EncodeToString([]byte("batch data 3")),
			"file_name": "batch-item-3.txt",
			"tags":      []string{"batch", "test"},
		},
	}

	batchArgs := map[string]interface{}{
		"owner_id": ownerID.String(),
		"items":    items,
	}

	batchArgsJSON, _ := json.Marshal(batchArgs)
	batchReq := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "batch_upload",
			Arguments: batchArgsJSON,
		},
	}

	result, err := server.handleBatchUpload(ctx, batchReq)
	if err != nil {
		t.Fatalf("handleBatchUpload failed: %v", err)
	}

	// Verify result
	textContent := result.Content[0].(*mcp.TextContent)
	var resultData map[string]interface{}
	json.Unmarshal([]byte(textContent.Text), &resultData)

	total := int(resultData["total"].(float64))
	if total != 3 {
		t.Errorf("Expected total 3, got %d", total)
	}

	successful := int(resultData["successful"].(float64))
	if successful != 3 {
		t.Errorf("Expected 3 successful uploads, got %d", successful)
	}

	failed := int(resultData["failed"].(float64))
	if failed != 0 {
		t.Errorf("Expected 0 failed uploads, got %d", failed)
	}

	// Verify results array
	results := resultData["results"].([]interface{})
	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}

	for _, r := range results {
		resultItem := r.(map[string]interface{})
		if !resultItem["success"].(bool) {
			t.Errorf("Upload should have succeeded, got error: %v", resultItem["error"])
		}
		if resultItem["content_id"] == nil || resultItem["content_id"].(string) == "" {
			t.Error("Expected content_id to be set")
		}
	}
}

func TestBatchGetDetails(t *testing.T) {
	server := createTestServer(t)
	ctx := context.Background()

	ownerID := uuid.New()

	// Upload some content first
	contentIDs := []string{}
	for i := 0; i < 3; i++ {
		uploadArgs := map[string]interface{}{
			"owner_id":  ownerID.String(),
			"name":      fmt.Sprintf("test%d.txt", i),
			"data":      base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("test data %d", i))),
			"file_name": fmt.Sprintf("test%d.txt", i),
		}

		uploadArgsJSON, _ := json.Marshal(uploadArgs)
		uploadReq := &mcp.CallToolRequest{
			Params: &mcp.CallToolParamsRaw{
				Name:      "upload_content",
				Arguments: uploadArgsJSON,
			},
		}

		uploadResult, err := server.handleUploadContent(ctx, uploadReq)
		if err != nil {
			t.Fatalf("Upload failed: %v", err)
		}

		textContent := uploadResult.Content[0].(*mcp.TextContent)
		var uploadData map[string]interface{}
		json.Unmarshal([]byte(textContent.Text), &uploadData)
		contentIDs = append(contentIDs, uploadData["id"].(string))
	}

	// Batch get details
	batchArgs := map[string]interface{}{
		"content_ids": contentIDs,
	}

	batchArgsJSON, _ := json.Marshal(batchArgs)
	batchReq := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "batch_get_details",
			Arguments: batchArgsJSON,
		},
	}

	result, err := server.handleBatchGetDetails(ctx, batchReq)
	if err != nil {
		t.Fatalf("handleBatchGetDetails failed: %v", err)
	}

	// Verify result
	textContent := result.Content[0].(*mcp.TextContent)
	var resultData map[string]interface{}
	json.Unmarshal([]byte(textContent.Text), &resultData)

	total := int(resultData["total"].(float64))
	if total != 3 {
		t.Errorf("Expected total 3, got %d", total)
	}

	successful := int(resultData["successful"].(float64))
	if successful != 3 {
		t.Errorf("Expected 3 successful fetches, got %d", successful)
	}

	// Verify results array
	results := resultData["results"].([]interface{})
	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}

	for _, r := range results {
		resultItem := r.(map[string]interface{})
		if resultItem["Error"] != nil && resultItem["Error"].(string) != "" {
			t.Errorf("Get details should have succeeded, got error: %v", resultItem["Error"])
		}
		if resultItem["Details"] == nil {
			t.Error("Expected Details to be set")
		}
	}
}
