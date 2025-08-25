.PHONY: build test coverage lint fmt vet tidy clean install-tools check help

# Default target
.DEFAULT_GOAL := test

# Build the library (verify it compiles)
build:
	@echo "Building..."
	go build ./...

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Run tests with coverage
coverage:
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	go tool cover -func=coverage.out

# Run linting
lint:
	@echo "Running linters..."
	golangci-lint run

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Run go vet
vet:
	@echo "Running go vet..."
	go vet ./...

# Tidy dependencies
tidy:
	@echo "Tidying dependencies..."
	go mod tidy

# Clean build artifacts
clean:
	@echo "Cleaning..."
	go clean
	rm -f coverage.out coverage.html

# Install development tools
install-tools:
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run all checks (useful for CI)
check: fmt vet lint test

# Run benchmarks
bench:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./...

# Display help
help:
	@echo "Available targets:"
	@echo "  build         - Build the library"
	@echo "  test          - Run tests"
	@echo "  coverage      - Run tests with coverage report"
	@echo "  lint          - Run golangci-lint"
	@echo "  fmt           - Format code with go fmt"
	@echo "  vet           - Run go vet"
	@echo "  tidy          - Tidy go modules"
	@echo "  clean         - Clean build artifacts"
	@echo "  install-tools - Install development tools"
	@echo "  check         - Run fmt, vet, lint, and test"
	@echo "  bench         - Run benchmarks"
	@echo "  help          - Display this help message"