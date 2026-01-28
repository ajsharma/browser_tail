# browser_tail Makefile

# Version from git tag or default
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')

# Go parameters
GOCMD := go
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOMOD := $(GOCMD) mod
GOFMT := gofmt

# Build parameters
BINARY_NAME := browser_tail
MAIN_PATH := ./cmd/browser_tail
LDFLAGS := -ldflags "-X github.com/ajsharma/browser_tail/internal/config.Version=$(VERSION)"

# Output directories
BUILD_DIR := build
DIST_DIR := dist

.PHONY: all build clean test fmt lint install uninstall help

all: build

## build: Build the binary for the current platform
build:
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) $(MAIN_PATH)

## build-all: Build for all supported platforms
build-all: clean
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_PATH)
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(MAIN_PATH)

## clean: Remove build artifacts
clean:
	rm -f $(BINARY_NAME)
	rm -rf $(BUILD_DIR)
	rm -rf $(DIST_DIR)

## test: Run tests
test:
	$(GOTEST) -v ./...

## test-coverage: Run tests with coverage
test-coverage:
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

## fmt: Format code
fmt:
	$(GOFMT) -s -w .

## lint: Run linters
lint:
	golangci-lint run

## tidy: Tidy dependencies
tidy:
	$(GOMOD) tidy

## install: Install to /usr/local/bin
install: build
	install -m 755 $(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)

## uninstall: Remove from /usr/local/bin
uninstall:
	rm -f /usr/local/bin/$(BINARY_NAME)

## version: Show version info
version:
	@echo "Version: $(VERSION)"
	@echo "Commit: $(COMMIT)"
	@echo "Build Time: $(BUILD_TIME)"

## help: Show this help
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed 's/^/ /'
