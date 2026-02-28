.PHONY: all build test lint clean dev testsuite install help

# Default target
all: dev

# Development workflow (equivalent to ./dev.sh)
dev:
	@FORCE_COLOR=1 ./dev.sh

# Build only
build:
	@FORCE_COLOR=1 ./dev.sh --skip-tests

# Run tests
test:
	@go test ./...

# Run linter
lint:
	@golangci-lint run

# Run full testsuite
testsuite:
	@FORCE_COLOR=1 ./dev.sh --testsuite

# Install binary to ~/.local/bin
install: build
	@cp ./bin/hpf ~/.local/bin/hpf

# Clean build artifacts
clean:
	@rm -rf ./bin/

# Show help
help:
	@echo "Available targets:"
	@echo "  all       - Run full development workflow (default)"
	@echo "  dev       - Run development workflow (./dev.sh)"
	@echo "  build     - Build only (skip tests)"
	@echo "  test      - Run unit tests only"
	@echo "  lint      - Run linter only"
	@echo "  testsuite - Run full testsuite"
	@echo "  install   - Build and install hpf to ~/.local/bin"
	@echo "  clean     - Clean build artifacts"
	@echo "  help      - Show this help"
	@echo ""
	@echo "For more options, use: ./dev.sh --help" 