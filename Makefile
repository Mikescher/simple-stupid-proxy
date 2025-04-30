.PHONY: all build run docker-build docker-run tidy fmt clean help

# Variables
APP_NAME=go-proxy
BINARY_NAME=proxy
DOCKER_IMAGE_NAME=go-proxy
GO_FILES=$(wildcard *.go)
# Default auth key for local run - CHANGE THIS OR USE ENV VAR
DEFAULT_AUTH_KEY=changeme_in_makefile_or_env

# Default target
all: tidy fmt build

# Build the Go application
build: $(GO_FILES) go.mod
	@echo "Building $(APP_NAME)..."
	@go build -o $(BINARY_NAME) main.go
	@echo "$(APP_NAME) built successfully."

# Run the Go application locally
# Requires PROXY_AUTH_KEY to be set in the environment or uses DEFAULT_AUTH_KEY
run: build
	@echo "Running $(APP_NAME)... (Using auth key: $(or $(PROXY_AUTH_KEY), $(DEFAULT_AUTH_KEY)))"
	@PROXY_AUTH_KEY=$(or $(PROXY_AUTH_KEY), $(DEFAULT_AUTH_KEY)) ./$(BINARY_NAME)

# Build the Docker image
docker-build:
	@echo "Building Docker image $(DOCKER_IMAGE_NAME)..."
	@docker build -t $(DOCKER_IMAGE_NAME) .
	@echo "Docker image $(DOCKER_IMAGE_NAME) built."

# Run the application inside a Docker container
# Requires PROXY_AUTH_KEY to be set in the environment or uses DEFAULT_AUTH_KEY
docker-run: docker-build
	@echo "Running $(APP_NAME) in Docker... (Using auth key: $(or $(PROXY_AUTH_KEY), $(DEFAULT_AUTH_KEY)))"
	@docker run --rm -p 8080:8080 -e PORT=8080 -e PROXY_AUTH_KEY=$(or $(PROXY_AUTH_KEY), $(DEFAULT_AUTH_KEY)) $(DOCKER_IMAGE_NAME)

# Tidy Go modules
tidy:
	@echo "Running go mod tidy..."
	@go mod tidy

# Format Go code
fmt:
	@echo "Running go fmt..."
	@go fmt ./...

# Clean build artifacts
clean:
	@echo "Cleaning up..."
	@rm -f $(BINARY_NAME)
	@echo "Cleanup complete."

# Display help message
help:
	@echo "Available targets:"
	@echo "  all          : Format, tidy modules, and build the application (default)"
	@echo "  build        : Build the Go application"
	@echo "  run          : Run the application locally (set PROXY_AUTH_KEY env var or modify Makefile)"
	@echo "  docker-build : Build the Docker image"
	@echo "  docker-run   : Run the application in a Docker container (set PROXY_AUTH_KEY env var or modify Makefile)"
	@echo "  tidy         : Tidy Go modules"
	@echo "  fmt          : Format Go code"
	@echo "  clean        : Remove build artifacts"
	@echo "  help         : Show this help message"

# Prevent Make from thinking files named 'build', 'run', etc. are actual targets
.PHONY: all build run docker-build docker-run tidy fmt clean help
