# Makefile for Chat Application

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=chat
BINARY_UNIX=$(BINARY_NAME)_unix

# Build settings
BUILD_DIR=build
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)"
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date +%Y-%m-%dT%H:%M:%S)

# Directories
PKG_DIR=pkg
CMD_DIR=.

# Docker settings
DOCKER_IMAGE=chat-app
DOCKER_TAG=latest

.PHONY: all build clean test coverage deps run docker-build docker-run help install format lint vet check

# Default target
all: clean deps lint vet test build

# Build the application
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) -v ./$(CMD_DIR)

# Build for Linux
build-linux:
	@echo "Building $(BINARY_UNIX) for Linux..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_UNIX) -v ./$(CMD_DIR)

# Build for multiple platforms
build-all:
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)

	# Linux AMD64
	@echo "Building for linux/amd64..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 -v ./$(CMD_DIR)

	# Linux ARM64
	@echo "Building for linux/arm64..."
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 -v ./$(CMD_DIR)

	# macOS AMD64
	@echo "Building for darwin/amd64..."
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 -v ./$(CMD_DIR)

	# macOS ARM64 (Apple Silicon)
	@echo "Building for darwin/arm64..."
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 -v ./$(CMD_DIR)

	# Windows AMD64
	@echo "Building for windows/amd64..."
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe -v ./$(CMD_DIR)

# Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	@rm -rf $(BUILD_DIR)
	@rm -f $(BINARY_NAME)
	@rm -f $(BINARY_UNIX)

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

# Run tests with coverage
coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# Run the application
run:
	@echo "Running $(BINARY_NAME)..."
	$(GOBUILD) -o $(BINARY_NAME) -v ./$(CMD_DIR)
	./$(BINARY_NAME)

# Run with environment variables
run-dev:
	@echo "Running $(BINARY_NAME) in development mode..."
	PORT=8080 SESSION_DIR=./sessions MAX_HISTORY_SIZE=50 SKILLS_DIR=../../testdata/skills MCP_CONFIG_PATH=../../testdata/mcp/mcp.json $(GOBUILD) -o $(BINARY_NAME) -v ./$(CMD_DIR) && ./$(BINARY_NAME)

# Install the application
install: build
	@echo "Installing $(BINARY_NAME)..."
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/

# Format code
format:
	@echo "Formatting code..."
	$(GOCMD) fmt ./...

# Lint code
lint:
	@echo "Linting code..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed. Install with: curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b \$$(go env GOPATH)/bin v1.54.2" && exit 1)
	golangci-lint run

# Vet code
vet:
	@echo "Vetting code..."
	$(GOCMD) vet ./...

# Run all checks (format, vet, lint, test)
check: format vet lint test

# Generate documentation
docs:
	@echo "Generating documentation..."
	@mkdir -p docs
	@which godoc > /dev/null || (echo "godoc not installed. Install with: go install golang.org/x/tools/cmd/godoc@latest" && exit 1)
	godoc -http=:6060 &
	@echo "Documentation available at http://localhost:6060"

# Build Docker image
docker-build:
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

# Run Docker container
docker-run:
	@echo "Running Docker container..."
	docker run -p 8080:8080 -e PORT=8080 -e SESSION_DIR=/app/sessions -v $(PWD)/sessions:/app/sessions $(DOCKER_IMAGE):$(DOCKER_TAG)

# Docker build and run
docker-up: docker-build docker-run

# Stop running containers
docker-stop:
	@echo "Stopping Docker containers..."
	docker stop $$(docker ps -q --filter ancestor=$(DOCKER_IMAGE)) || true
	docker rm $$(docker ps -aq --filter ancestor=$(DOCKER_IMAGE)) || true

# Development setup (install tools)
setup-dev:
	@echo "Setting up development environment..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install golang.org/x/tools/cmd/godoc@latest
	@go install github.com/air-verse/air@latest
	@echo "Development tools installed. Use 'make dev' for hot reload."

# Development with hot reload
dev:
	@echo "Starting development server with hot reload..."
	@which air > /dev/null || (echo "air not installed. Run 'make setup-dev' first." && exit 1)
	air

# Create release package
release: clean test build-all
	@echo "Creating release package..."
	@mkdir -p $(BUILD_DIR)/release
	@cd $(BUILD_DIR) && tar -czf release/$(BINARY_NAME)-$(VERSION)-linux-amd64.tar.gz $(BINARY_NAME)-linux-amd64
	@cd $(BUILD_DIR) && tar -czf release/$(BINARY_NAME)-$(VERSION)-linux-arm64.tar.gz $(BINARY_NAME)-linux-arm64
	@cd $(BUILD_DIR) && tar -czf release/$(BINARY_NAME)-$(VERSION)-darwin-amd64.tar.gz $(BINARY_NAME)-darwin-amd64
	@cd $(BUILD_DIR) && tar -czf release/$(BINARY_NAME)-$(VERSION)-darwin-arm64.tar.gz $(BINARY_NAME)-darwin-arm64
	@cd $(BUILD_DIR) && zip -q release/$(BINARY_NAME)-$(VERSION)-windows-amd64.zip $(BINARY_NAME)-windows-amd64.exe
	@echo "Release packages created in $(BUILD_DIR)/release/"

# Show help
help:
	@echo "Available targets:"
	@echo "  all          - Clean, deps, lint, vet, test, and build"
	@echo "  build        - Build the application"
	@echo "  build-linux  - Build for Linux"
	@echo "  build-all    - Build for multiple platforms"
	@echo "  clean        - Clean build artifacts"
	@echo "  test         - Run tests"
	@echo "  coverage     - Run tests with coverage report"
	@echo "  deps         - Download dependencies"
	@echo "  run          - Build and run the application"
	@echo "  run-dev      - Run with development environment variables"
	@echo "  install      - Install the application"
	@echo "  format       - Format code"
	@echo "  lint         - Lint code"
	@echo "  vet          - Vet code"
	@echo "  check        - Run all checks (format, vet, lint, test)"
	@echo "  docs         - Generate and serve documentation"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-run   - Run Docker container"
	@echo "  docker-up    - Build and run Docker"
	@echo "  docker-stop  - Stop Docker containers"
	@echo "  setup-dev    - Install development tools"
	@echo "  dev          - Run with hot reload"
	@echo "  release      - Create release packages"
	@echo "  help         - Show this help message"