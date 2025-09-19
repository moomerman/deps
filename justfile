# Development and build tasks for deps

# Default task - show available commands
default:
    @just --list

# Development build (fast, includes debug info)
build:
    go build -o deps *.go

# Production build (optimized, small binary)
build-prod:
    CGO_ENABLED=0 go build -ldflags="-s -w" -trimpath -o deps *.go

# Build for all major platforms
build-all: clean
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -trimpath -o dist/deps-linux-amd64 *.go
    CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -trimpath -o dist/deps-windows-amd64.exe *.go
    CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -trimpath -o dist/deps-darwin-amd64 *.go
    CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -trimpath -o dist/deps-darwin-arm64 *.go
    CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -trimpath -o dist/deps-linux-arm64 *.go

# Build with version info
build-version VERSION:
    CGO_ENABLED=0 go build -ldflags="-s -w -X main.version={{VERSION}}" -trimpath -o deps *.go

# Run tests
test:
    go test ./...

# Run with go run (for development)
run *ARGS:
    go run *.go {{ARGS}}

# Format code
fmt:
    go fmt ./...

# Run linter (requires golangci-lint)
lint:
    golangci-lint run

# Clean build artifacts
clean:
    rm -f deps
    rm -rf dist/

# Create dist directory for releases
dist:
    mkdir -p dist

# Install locally (to GOPATH/bin or GOBIN)
install:
    CGO_ENABLED=0 go install -ldflags="-s -w" -trimpath .

# Show binary info
info: build-prod
    @echo "Binary size:"
    @ls -lh deps
    @echo "\nBinary info:"
    @file deps

# Quick development workflow - format, build, test
dev: fmt build test

# Release workflow - format, lint, build all platforms
release VERSION: fmt lint clean dist
    just build-version {{VERSION}}
    just build-all
    @echo "Release {{VERSION}} built successfully in dist/"
