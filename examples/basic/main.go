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
