.PHONY: build test clean install lint coverage help vet tidy security-scan ci check snapshot release build-all deps fmt run

# Variables
BINARY_NAME=tfskel
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-s -w -X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildTime=$(BUILD_TIME)"
BUILD_DIR=dist
GO_FILES=$(shell find . -name '*.go' -not -path "./vendor/*" -not -path "./.history/*" -not -path "./dist/*")

# Build the application
build:
	@echo "Building $(BINARY_NAME) $(VERSION)..."
	@go build $(LDFLAGS) -o $(BINARY_NAME) .

# Run tests with race detector
test:
	@echo "Running tests..."
	@go test -v -race -coverprofile=coverage.out ./...

# Run tests with coverage report
coverage: test
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run linter
lint:
	@echo "Running linter..."
	@golangci-lint run

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -f $(BINARY_NAME)
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html

# Install dependencies
deps:
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy

# Install the binary
install: build
	@echo "Installing $(BINARY_NAME)..."
	@go install

# Run the application
run: build
	@./$(BINARY_NAME)

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt $(shell go list ./... | grep -v '.history')
	@which goimports > /dev/null 2>&1 && goimports -w -local . $(GO_FILES) || echo "Tip: Install goimports with: go install golang.org/x/tools/cmd/goimports@latest"

# Run go vet
vet:
	@echo "Running go vet..."
	@go vet $(shell go list ./... | grep -v '.history')

# Tidy dependencies
tidy:
	@echo "Tidying dependencies..."
	@go mod tidy

# Security scanning
security-scan:
	@echo "Running security scans..."
	@which gosec > /dev/null 2>&1 && gosec ./... || echo "Tip: Install gosec with: go install github.com/securego/gosec/v2/cmd/gosec@latest"
	@which trivy > /dev/null 2>&1 && trivy fs --severity HIGH,CRITICAL . || echo "Tip: Install trivy from https://github.com/aquasecurity/trivy"

# Run all checks (used in CI and local development)
check: tidy fmt vet lint test
	@echo "All checks passed!"

# Full CI pipeline
ci: check security-scan
	@echo "CI pipeline completed successfully!"

# Build for multiple platforms (use GoReleaser for releases)
build-all:
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 .
	@GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 .
	@GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 .
	@echo "Built binaries in $(BUILD_DIR)/"

# Create snapshot release with GoReleaser
snapshot:
	@echo "Creating snapshot release..."
	@which goreleaser > /dev/null 2>&1 && goreleaser release --snapshot --skip-publish --clean || echo "Tip: Install GoReleaser from https://goreleaser.com/install/"

# Create production release with GoReleaser (requires git tag)
release:
	@if [ -z "$$(git tag --points-at HEAD)" ]; then \
		echo "Error: No git tag found at HEAD. Create a tag first: git tag v1.0.0"; \
		exit 1; \
	fi
	@which goreleaser > /dev/null 2>&1 && goreleaser release --clean || echo "Error: Install GoReleaser from https://goreleaser.com/install/"

# Help target
help:
	@echo "Available targets:"
	@echo "  build          - Build the application with version info"
	@echo "  test           - Run tests with race detector"
	@echo "  coverage       - Run tests with coverage report"
	@echo "  lint           - Run golangci-lint"
	@echo "  fmt            - Format code (go fmt + goimports)"
	@echo "  vet            - Run go vet"
	@echo "  tidy           - Tidy go modules"
	@echo "  security-scan  - Run security scanners (gosec, trivy)"
	@echo "  check          - Run all checks (tidy, fmt, vet, lint, test)"
	@echo "  ci             - Full CI pipeline (check + security)"
	@echo "  clean          - Clean build artifacts"
	@echo "  deps           - Install dependencies"
	@echo "  install        - Install the binary to GOPATH/bin"
	@echo "  run            - Build and run the application"
	@echo "  build-all      - Build for multiple platforms"
	@echo "  snapshot       - Create local snapshot release (GoReleaser)"
	@echo "  release        - Create production release (GoReleaser, requires tag)"
	@echo ""
	@echo "Common workflows:"
	@echo "  make check     - Run all checks before committing"
	@echo "  make ci        - Run full CI pipeline locally"
	@echo "  make build     - Quick build for local testing"
