package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	fmt.Println("=== Simple Content MCP Client Example ===")
	fmt.Println()

	// Create MCP client
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "example-client",
		Version: "0.1.0",
	}, nil)

	// Start the MCP server as a subprocess
	cmd := exec.Command("../../mcpserver", "--mode=stdio")
	transport := &mcp.CommandTransport{
		Command: cmd,
	}

	fmt.Println("Connecting to MCP server...")
	session, err := client.Connect(context.Background(), transport, nil)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer session.Close()

	fmt.Println("Connected successfully!")
	fmt.Println()

	ctx := context.Background()
	ownerID := uuid.New()

	// Example 1: Upload content
	fmt.Println("Example 1: Uploading content...")
	testData := "Hello from MCP client! This is test content."
	encodedData := base64.StdEncoding.EncodeToString([]byte(testData))

	uploadResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "upload_content",
		Arguments: map[string]interface{}{
			"owner_id":  ownerID.String(),
			"name":      "example.txt",
			"data":      encodedData,
			"file_name": "example.txt",
			"tags":      []string{"example", "test"},
		},
	})
	if err != nil {
		log.Fatalf("Upload failed: %v", err)
	}

	uploadData := parseResult(uploadResult)
	contentID := uploadData["id"].(string)
	fmt.Printf("✓ Content uploaded successfully!\n")
	fmt.Printf("  ID: %s\n", contentID)
	fmt.Printf("  Status: %s\n\n", uploadData["status"])

	// Example 2: Get content details
	fmt.Println("Example 2: Getting content details...")
	detailsResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "get_content_details",
		Arguments: map[string]interface{}{
			"content_id": contentID,
		},
	})
	if err != nil {
		log.Fatalf("Get details failed: %v", err)
	}

	detailsData := parseResult(detailsResult)
	fmt.Printf("✓ Content details retrieved:\n")
	fmt.Printf("  ID: %s\n", detailsData["id"])
	fmt.Printf("  File Name: %s\n", detailsData["file_name"])
	fmt.Printf("  File Size: %v bytes\n", detailsData["file_size"])
	fmt.Printf("  Ready: %v\n\n", detailsData["ready"])

	// Example 3: Upload more content for listing
	fmt.Println("Example 3: Uploading additional content...")
	for i := 1; i <= 2; i++ {
		data := fmt.Sprintf("Additional content #%d", i)
		encoded := base64.StdEncoding.EncodeToString([]byte(data))

		_, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name: "upload_content",
			Arguments: map[string]interface{}{
				"owner_id":  ownerID.String(),
				"name":      fmt.Sprintf("file%d.txt", i),
				"data":      encoded,
				"file_name": fmt.Sprintf("file%d.txt", i),
			},
		})
		if err != nil {
			log.Fatalf("Upload %d failed: %v", i, err)
		}
		fmt.Printf("  ✓ Uploaded file%d.txt\n", i)
	}
	fmt.Println()

	// Example 4: List content
	fmt.Println("Example 4: Listing content...")
	listResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "list_content",
		Arguments: map[string]interface{}{
			"owner_id": ownerID.String(),
			"limit":    10,
			"offset":   0,
		},
	})
	if err != nil {
		log.Fatalf("List failed: %v", err)
	}

	listData := parseResult(listResult)
	items := listData["items"].([]interface{})
	fmt.Printf("✓ Found %v content items:\n", listData["total"])
	for i, item := range items {
		itemMap := item.(map[string]interface{})
		fmt.Printf("  %d. %s (ID: %s)\n", i+1, itemMap["name"], itemMap["id"])
	}
	fmt.Println()

	// Example 5: Search content
	fmt.Println("Example 5: Searching content...")
	searchResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "search_content",
		Arguments: map[string]interface{}{
			"owner_id": ownerID.String(),
			"query":    "example",
		},
	})
	if err != nil {
		log.Fatalf("Search failed: %v", err)
	}

	searchData := parseResult(searchResult)
	searchItems := searchData["items"].([]interface{})
	fmt.Printf("✓ Search results for 'example': %d items\n", len(searchItems))
	for _, item := range searchItems {
		itemMap := item.(map[string]interface{})
		fmt.Printf("  - %s\n", itemMap["name"])
	}
	fmt.Println()

	// Example 6: Download content
	fmt.Println("Example 6: Downloading content...")
	downloadResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "download_content",
		Arguments: map[string]interface{}{
			"content_id": contentID,
			"format":     "base64",
		},
	})
	if err != nil {
		log.Fatalf("Download failed: %v", err)
	}

	downloadData := parseResult(downloadResult)
	decodedData, _ := base64.StdEncoding.DecodeString(downloadData["data"].(string))
	fmt.Printf("✓ Downloaded content:\n")
	fmt.Printf("  Data: %s\n\n", string(decodedData))

	// Example 7: Update content
	fmt.Println("Example 7: Updating content metadata...")
	_, err = session.CallTool(ctx, &mcp.CallToolParams{
		Name: "update_content",
		Arguments: map[string]interface{}{
			"content_id":  contentID,
			"name":        "updated-example.txt",
			"description": "This content has been updated",
			"tags":        []string{"updated", "example"},
		},
	})
	if err != nil {
		log.Fatalf("Update failed: %v", err)
	}
	fmt.Printf("✓ Content updated successfully\n\n")

	// Example 8: Delete content
	fmt.Println("Example 8: Deleting content...")
	deleteResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "delete_content",
		Arguments: map[string]interface{}{
			"content_id": contentID,
		},
	})
	if err != nil {
		log.Fatalf("Delete failed: %v", err)
	}

	deleteData := parseResult(deleteResult)
	fmt.Printf("✓ Content deleted successfully at %s\n\n", deleteData["deleted_at"])

	// Example 9: Get content status
	fmt.Println("Example 9: Getting content status...")
	// Upload a new content for status check
	statusTestData := "Content for status check"
	statusEncodedData := base64.StdEncoding.EncodeToString([]byte(statusTestData))
	statusUploadResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "upload_content",
		Arguments: map[string]interface{}{
			"owner_id":  ownerID.String(),
			"name":      "status-test.txt",
			"data":      statusEncodedData,
			"file_name": "status-test.txt",
		},
	})
	if err != nil {
		log.Fatalf("Status test upload failed: %v", err)
	}

	statusUploadData := parseResult(statusUploadResult)
	statusContentID := statusUploadData["id"].(string)

	statusResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "get_content_status",
		Arguments: map[string]interface{}{
			"content_id": statusContentID,
		},
	})
	if err != nil {
		log.Fatalf("Get status failed: %v", err)
	}

	statusData := parseResult(statusResult)
	fmt.Printf("✓ Content status retrieved:\n")
	fmt.Printf("  Status: %s\n", statusData["status"])
	fmt.Printf("  Ready: %v\n", statusData["ready"])
	fmt.Printf("  Has Thumbnails: %v\n", statusData["has_thumbnails"])
	fmt.Printf("  Has Previews: %v\n\n", statusData["has_previews"])

	// Example 10: List by status
	fmt.Println("Example 10: Listing content by status...")
	listStatusResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "list_by_status",
		Arguments: map[string]interface{}{
			"status":   "uploaded",
			"owner_id": ownerID.String(),
			"limit":    5,
		},
	})
	if err != nil {
		log.Fatalf("List by status failed: %v", err)
	}

	listStatusData := parseResult(listStatusResult)
	statusItems := listStatusData["items"].([]interface{})
	count := int(listStatusData["count"].(float64))
	fmt.Printf("✓ Found %d uploaded items\n", count)
	for i, item := range statusItems {
		itemMap := item.(map[string]interface{})
		fmt.Printf("  %d. %s (Status: %s)\n", i+1, itemMap["name"], itemMap["status"])
	}
	fmt.Println()

	// Example 11: List derived content (empty for now)
	fmt.Println("Example 11: Listing derived content...")
	derivedResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "list_derived_content",
		Arguments: map[string]interface{}{
			"parent_id": statusContentID,
		},
	})
	if err != nil {
		log.Fatalf("List derived failed: %v", err)
	}

	derivedData := parseResult(derivedResult)
	derivedItems := derivedData["items"].([]interface{})
	fmt.Printf("✓ Found %d derived content items (thumbnails would appear here if generated)\n\n", len(derivedItems))

	// Example 12: Read a resource (Phase 3)
	fmt.Println("Example 12: Reading MCP resources...")
	// Read content as a resource
	resourceResult, err := session.ReadResource(ctx, &mcp.ReadResourceParams{
		URI: "content://" + statusContentID,
	})
	if err != nil {
		log.Fatalf("Read resource failed: %v", err)
	}
	fmt.Printf("✓ Read resource: %s\n", resourceResult.Contents[0].URI)
	fmt.Printf("  MIME Type: %s\n\n", resourceResult.Contents[0].MIMEType)

	// Read schema resource
	schemaResult, err := session.ReadResource(ctx, &mcp.ReadResourceParams{
		URI: "schema://content",
	})
	if err != nil {
		log.Fatalf("Read schema resource failed: %v", err)
	}
	fmt.Printf("✓ Read schema resource\n")
	fmt.Printf("  MIME Type: %s\n\n", schemaResult.Contents[0].MIMEType)

	// Read stats resource
	statsResult, err := session.ReadResource(ctx, &mcp.ReadResourceParams{
		URI: "stats://system",
	})
	if err != nil {
		log.Fatalf("Read stats resource failed: %v", err)
	}
	fmt.Printf("✓ Read system stats resource\n")
	fmt.Printf("  Content: %s\n\n", statsResult.Contents[0].Text[:100]+"...")

	// Example 13: Get a prompt (Phase 3)
	fmt.Println("Example 13: Getting MCP prompts...")
	promptResult, err := session.GetPrompt(ctx, &mcp.GetPromptParams{
		Name: "upload-workflow",
		Arguments: map[string]string{
			"content_type": "image",
		},
	})
	if err != nil {
		log.Fatalf("Get prompt failed: %v", err)
	}
	fmt.Printf("✓ Retrieved upload-workflow prompt\n")
	if len(promptResult.Messages) > 0 {
		textContent, ok := promptResult.Messages[0].Content.(*mcp.TextContent)
		if ok {
			// Show first 200 chars of the guidance
			guidance := textContent.Text
			if len(guidance) > 200 {
				guidance = guidance[:200] + "..."
			}
			fmt.Printf("  Guidance: %s\n\n", guidance)
		}
	}

	// List all available prompts
	promptsList, err := session.ListPrompts(ctx, &mcp.ListPromptsParams{})
	if err != nil {
		log.Fatalf("List prompts failed: %v", err)
	}
	fmt.Printf("✓ Available prompts:\n")
	for _, prompt := range promptsList.Prompts {
		fmt.Printf("  - %s: %s\n", prompt.Name, prompt.Description)
	}
	fmt.Println()

	// Example 14: Batch upload (Phase 4)
	fmt.Println("Example 14: Batch uploading content...")
	batchItems := []map[string]interface{}{
		{
			"name":      "batch-doc-1.txt",
			"data":      base64.StdEncoding.EncodeToString([]byte("Batch document 1 content")),
			"file_name": "batch-doc-1.txt",
			"tags":      []string{"batch", "document"},
		},
		{
			"name":      "batch-doc-2.txt",
			"data":      base64.StdEncoding.EncodeToString([]byte("Batch document 2 content")),
			"file_name": "batch-doc-2.txt",
			"tags":      []string{"batch", "document"},
		},
		{
			"name":      "batch-doc-3.txt",
			"data":      base64.StdEncoding.EncodeToString([]byte("Batch document 3 content")),
			"file_name": "batch-doc-3.txt",
			"tags":      []string{"batch", "document"},
		},
		{
			"name":      "batch-doc-4.txt",
			"data":      base64.StdEncoding.EncodeToString([]byte("Batch document 4 content")),
			"file_name": "batch-doc-4.txt",
			"tags":      []string{"batch", "document"},
		},
		{
			"name":      "batch-doc-5.txt",
			"data":      base64.StdEncoding.EncodeToString([]byte("Batch document 5 content")),
			"file_name": "batch-doc-5.txt",
			"tags":      []string{"batch", "document"},
		},
	}

	batchResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "batch_upload",
		Arguments: map[string]interface{}{
			"owner_id": ownerID.String(),
			"items":    batchItems,
		},
	})
	if err != nil {
		log.Fatalf("Batch upload failed: %v", err)
	}

	batchData := parseResult(batchResult)
	fmt.Printf("✓ Batch upload completed\n")
	fmt.Printf("  Total: %v\n", batchData["total"])
	fmt.Printf("  Successful: %v\n", batchData["successful"])
	fmt.Printf("  Failed: %v\n\n", batchData["failed"])

	fmt.Println("=== All examples completed successfully! ===")
}

// Helper functions

func parseResult(result *mcp.CallToolResult) map[string]interface{} {
	if result.IsError {
		log.Fatalf("Tool returned error: %v", result.Content)
	}

	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		log.Fatalf("Expected text content, got %T", result.Content[0])
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(textContent.Text), &data); err != nil {
		log.Fatalf("Failed to parse result: %v", err)
	}

	return data
}
