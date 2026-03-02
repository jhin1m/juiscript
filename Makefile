# juiscript - LEMP server management TUI
# Build produces a single static binary

APP_NAME := juiscript
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT)"

# Default: build for current platform
.PHONY: build
build:
	go build $(LDFLAGS) -o bin/$(APP_NAME) ./cmd/juiscript

# Build for Ubuntu server (Linux AMD64)
.PHONY: build-linux
build-linux:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/$(APP_NAME)-linux-amd64 ./cmd/juiscript

# Install to /usr/local/bin (requires sudo on most systems)
.PHONY: install
install: build
	cp bin/$(APP_NAME) /usr/local/bin/$(APP_NAME)

# Run all tests
.PHONY: test
test:
	go test ./... -v -count=1

# Run tests with coverage report
.PHONY: cover
cover:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

# Format and lint
.PHONY: fmt
fmt:
	go fmt ./...
	go vet ./...

# Clean build artifacts
.PHONY: clean
clean:
	rm -rf bin/ coverage.out coverage.html

# Development: build and run
.PHONY: dev
dev: build
	./bin/$(APP_NAME)
