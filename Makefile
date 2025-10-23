# Simple Content MCP Server Makefile

.PHONY: build test clean run run-sse run-stdio help install lint fmt vet

# Binary name
BINARY_NAME=mcpserver

# Build directory
BUILD_DIR=.

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/mcpserver

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -f $(BUILD_DIR)/$(BINARY_NAME)
	rm -f coverage.out coverage.html
	rm -f /tmp/mcpserver.log

# Run in stdio mode (default)
run-stdio: build
	@echo "Running in stdio mode..."
	./$(BINARY_NAME) --mode=stdio

# Run in HTTP Streamable transport mode
run-stream: build
	@echo "Running in HTTP Streamable transport mode..."
	./$(BINARY_NAME) --mode=sse --port=3030

# Run in HTTP mode
run-http: build
	@echo "Running in HTTP mode..."
	./$(BINARY_NAME) --mode=http --port=3030

# Run with .env file
run-env: build
	@echo "Running with .env configuration..."
	./$(BINARY_NAME)

# Run with custom env file
run-test-env: build
	@echo "Running with .env.test configuration..."
	./$(BINARY_NAME) --env=.env.test

# Install dependencies
install:
	@echo "Installing dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# Format code
fmt:
	@echo "Formatting code..."
	$(GOFMT) ./...

# Run go vet
vet:
	@echo "Running go vet..."
	$(GOVET) ./...

# Run linter (requires golangci-lint)
lint:
	@echo "Running golangci-lint..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed. Install from https://golangci-lint.run/usage/install/" && exit 1)
	golangci-lint run ./...

# Run compliance tests
test-compliance: build
	@echo "Running MCP 2025-06-18 compliance tests..."
	./test_mcp_2025_spec.sh

# Test HTTP Streamable transport interactively
test-streamable:
	@echo "Running interactive HTTP Streamable transport test..."
	@echo "Make sure server is running with: make run-stream"
	@echo ""
	./test_streamable_interactive.sh

# Run all checks (fmt, vet, test)
check: fmt vet test
	@echo "All checks passed!"

# Build and run example
run-example:
	@echo "Building and running basic example..."
	cd examples/basic && go build -o example main.go && ./example

# Show version
version:
	./$(BINARY_NAME) --version

# Help target
help:
	@echo "Simple Content MCP Server - Available targets:"
	@echo ""
	@echo "  make build              - Build the mcpserver binary"
	@echo "  make test               - Run all tests"
	@echo "  make test-coverage      - Run tests with coverage report"
	@echo "  make clean              - Clean build artifacts"
	@echo ""
	@echo "  make run-stdio          - Run server in stdio mode"
	@echo "  make run-stream         - Run server in HTTP Streamable transport mode (port 3030)"
	@echo "  make run-http           - Run server in HTTP mode (port 3030)"
	@echo "  make run-env            - Run server with .env configuration"
	@echo "  make run-test-env       - Run server with .env.test configuration"
	@echo ""
	@echo "  make install            - Install/update dependencies"
	@echo "  make fmt                - Format code"
	@echo "  make vet                - Run go vet"
	@echo "  make lint               - Run golangci-lint (requires installation)"
	@echo "  make check              - Run fmt, vet, and tests"
	@echo ""
	@echo "  make test-compliance    - Run MCP 2025-06-18 compliance tests"
	@echo "  make test-streamable    - Test HTTP Streamable transport interactively"
	@echo "  make run-example        - Build and run basic example"
	@echo "  make version            - Show server version"
	@echo ""
	@echo "  make help               - Show this help message"

# Default target
.DEFAULT_GOAL := help
