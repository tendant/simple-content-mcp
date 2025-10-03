package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/tendant/simple-content-mcp/pkg/mcpserver"
	"github.com/tendant/simple-content/pkg/simplecontent"
	memoryrepo "github.com/tendant/simple-content/pkg/simplecontent/repo/memory"
	memorystorage "github.com/tendant/simple-content/pkg/simplecontent/storage/memory"
)

func main() {
	// Parse command-line flags
	var (
		mode    = flag.String("mode", "stdio", "Transport mode: stdio, sse, http")
		port    = flag.Int("port", 8080, "Port for SSE/HTTP mode")
		version = flag.Bool("version", false, "Print version and exit")
	)
	flag.Parse()

	if *version {
		fmt.Println("simple-content-mcp v0.1.0")
		os.Exit(0)
	}

	// Create context with signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Received shutdown signal, shutting down gracefully...")
		cancel()
	}()

	// Initialize simple-content service
	// For Phase 1, we use in-memory implementations for testing
	// In production, you would configure PostgreSQL and S3/filesystem storage
	service, err := createService()
	if err != nil {
		log.Fatalf("Failed to create service: %v", err)
	}

	// Create MCP server configuration
	config := mcpserver.DefaultConfig(service)
	config.Mode = mcpserver.TransportMode(*mode)
	config.Port = *port

	// Create and start MCP server
	server, err := mcpserver.New(config)
	if err != nil {
		log.Fatalf("Failed to create MCP server: %v", err)
	}

	log.Printf("Starting MCP server in %s mode...", config.Mode)
	if err := server.Serve(ctx); err != nil {
		if err == context.Canceled {
			log.Println("Server stopped")
		} else {
			log.Fatalf("Server error: %v", err)
		}
	}
}

// createService creates a simple-content service with in-memory storage
// This is suitable for development and testing
func createService() (simplecontent.Service, error) {
	// Create in-memory repository
	repo := memoryrepo.New()

	// Create in-memory blob storage
	blobStore := memorystorage.New()

	// Create service with memory backends
	service, err := simplecontent.New(
		simplecontent.WithRepository(repo),
		simplecontent.WithBlobStore("default", blobStore),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create service: %w", err)
	}

	return service, nil
}
