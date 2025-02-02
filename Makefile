# Project variables
APP_NAME := ghostshell
BUILD_DIR := build
GO_FILES := $(shell find . -name '*.go' -not -path "./vendor/*")
DOCKER_IMAGE := ghostshell:latest

# Default target
.PHONY: all
all: build

# Build the application
.PHONY: build
build:
	@echo "Building $(APP_NAME)..."
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o $(BUILD_DIR)/$(APP_NAME) ./cmd/ghostshell/main.go
	@echo "Build completed: $(BUILD_DIR)/$(APP_NAME)"

# Run the application
.PHONY: run
run: build
	@echo "Running $(APP_NAME)..."
	./$(BUILD_DIR)/$(APP_NAME)

# Test the application
.PHONY: test
test:
	@echo "Running tests..."
	go test ./... -cover
	@echo "Tests completed."

# Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Lint code (requires golangci-lint)
.PHONY: lint
lint:
	@echo "Linting code..."
	golangci-lint run
	@echo "Linting completed."

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning up build artifacts..."
	rm -rf $(BUILD_DIR)
	@echo "Cleanup completed."

# Build Docker image
.PHONY: docker-build
docker-build:
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE) .
	@echo "Docker image built: $(DOCKER_IMAGE)"

# Run Docker container
.PHONY: docker-run
docker-run:
	@echo "Running Docker container..."
	docker run --rm -it $(DOCKER_IMAGE)

# Install dependencies
.PHONY: deps
deps:
	@echo "Downloading dependencies..."
	go mod tidy
	@echo "Dependencies installed."

# Help command to list all targets
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  make build         - Build the application"
	@echo "  make run           - Run the application"
	@echo "  make test          - Run tests"
	@echo "  make fmt           - Format code"
	@echo "  make lint          - Lint code (requires golangci-lint)"
	@echo "  make clean         - Clean build artifacts"
	@echo "  make docker-build  - Build Docker image"
	@echo "  make docker-run    - Run Docker container"
	@echo "  make deps          - Install dependencies"
	@echo "  make help          - Show this help message"
