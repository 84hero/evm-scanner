.PHONY: all build test lint clean docker-build help snapshot release run

# Project Variables
BINARY_NAME=scanner-cli
CMD_PATH=./cmd/scanner-cli
DOCKER_IMAGE=evm-scanner:latest

# Default Target
all: build

# Print Help Information
help:
	@echo "Available commands:"
	@echo "  make build         - Build the scanner-cli binary to bin/"
	@echo "  make test          - Run all unit tests with coverage"
	@echo "  make lint          - Run golangci-lint"
	@echo "  make clean         - Remove binaries, temp files, and artifacts"
	@echo "  make docker-build  - Build the Docker image"
	@echo "  make run           - Run the scanner (requires config.yaml/app.yaml)"
	@echo "  make snapshot      - Test build with goreleaser (local)"
	@echo "  make release       - Build and release with goreleaser"

# Build Binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p bin
	go build -o bin/$(BINARY_NAME) $(CMD_PATH)
	@echo "Done! Binary is at bin/$(BINARY_NAME)"

# Run Tests
test:
	@echo "Running tests..."
	go test -v -cover ./...

# Lint Code (requires golangci-lint)
lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Skipping."; \
	fi

# Clean Build Artifacts
clean:
	@echo "Cleaning up..."
	rm -rf bin/
	rm -rf data/
	rm -f coverage.out
	rm -f scanner-cli
	@echo "Cleaned."

# Docker Build
docker-build:
	@echo "Building Docker image $(DOCKER_IMAGE)..."
	docker build -t $(DOCKER_IMAGE) .

# GoReleaser Local Test
snapshot:
	@echo "Running goreleaser snapshot..."
	goreleaser release --snapshot --clean --skip=publish

# GoReleaser Release (typically run in CI)
release:
	@echo "Running goreleaser release..."
	goreleaser release --clean

# Local Run (for development)
run: build
	@echo "Starting scanner..."
	./bin/$(BINARY_NAME)