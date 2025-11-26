# gomarkdown Makefile - Professional Go build system
.PHONY: help clean install dev debug watch watch-air watch-nodemon quality-check test build-all check-deps vendor

# Build variables
BINARY_NAME=gomtb-manifest
BUILD_FLAGS=-ldflags="-s -w" -mod=vendor
VENFORFILE=vendor/.timestamp
CMD_DIR=cmd/gomtb-manifest


# OS-aware binary naming
GOOS ?= $(shell go env GOOS)
BINARY_FULL=bin/$(BINARY_NAME)

# Default target
all: dev

# Build the main binary
$(BINARY_FULL): vendor
	@mkdir -p bin
	@echo "🏗️ Building $(BINARY_FULL)..."
	go build $(BUILD_FLAGS) -o $(BINARY_FULL) ./$(CMD_DIR)
	@echo "✅ Build complete"

# Quality checks
quality-check:
	@echo "🔍 Running quality checks..."
	go fmt ./...
	go vet ./...
	@command -v staticcheck >/dev/null 2>&1 && staticcheck ./... || echo "⚠️  staticcheck not installed (go install honnef.co/go/tools/cmd/staticcheck@latest)"
	@command -v errcheck >/dev/null 2>&1 && errcheck ./... || echo "⚠️  errcheck not installed (go install github.com/kisielk/errcheck@latest)"
	@echo "✅ Quality checks passed"

# Run tests
test:
	@echo "🧪 Running tests..."
	go test ./...

# Clean build artifacts
clean:
	@echo "🧹 Cleaning..."
	rm -rf bin/
	go clean

# Install dependencies
deps:
	@echo "📦 Installing dependencies..."
	go mod download
	go mod tidy
	go mod vendor
	@echo "📦 Installing development tools..."
	go install honnef.co/go/tools/cmd/staticcheck@latest
	go install github.com/kisielk/errcheck@latest
	go install github.com/air-verse/air@latest
	@echo "✅ Dependencies installed and vendored"

# Check for required tools
check-deps:
	@echo "🔍 Checking for required tools..."
	@echo ""
	@echo "Core tools:"
	@command -v go >/dev/null 2>&1 && echo "  ✅ go        $$(go version | cut -d' ' -f3)" || echo "  ❌ go        NOT FOUND"
	@echo ""
	@echo "Quality tools:"
	@command -v gofmt >/dev/null 2>&1 && echo "  ✅ gofmt     (built-in)" || echo "  ❌ gofmt     NOT FOUND"
	@go help vet >/dev/null 2>&1 && echo "  ✅ go vet    (built-in)" || echo "  ❌ go vet    NOT FOUND"
	@command -v staticcheck >/dev/null 2>&1 && echo "  ✅ staticcheck $$(staticcheck -version 2>&1 | head -n1)" || echo "  ⚠️  staticcheck NOT FOUND - run: go install honnef.co/go/tools/cmd/staticcheck@latest"
	@command -v errcheck >/dev/null 2>&1 && echo "  ✅ errcheck  installed" || echo "  ⚠️  errcheck NOT FOUND - run: go install github.com/kisielk/errcheck@latest"
	@echo ""
	@echo "Development tools:"
	@command -v air >/dev/null 2>&1 && echo "  ✅ air       $$(air -v 2>&1 | head -n1)" || echo "  ⚠️  air      NOT FOUND - run: go install github.com/air-verse/air@latest"
	@echo ""
	@echo "To install all missing tools, run: make deps"
	@echo ""

vendor: $(VENFORFILE)
	@if [ ! -d "vendor" ]; then \
		echo "⚠️  Vendor directory not found. Running 'go mod vendor'..."; \
		go mod vendor; \
	fi

# Vendor with timestamp tracking - auto-updates when go.mod/go.sum change
$(VENFORFILE): go.mod go.sum
	@echo "📦 Updating vendored dependencies..."
	go mod vendor
	@echo $(shell date) > $@

# Development build (quick, minimal checks)
dev: debug
	go fmt ./...
	go vet ./...

# Debug build (with debug symbols, no optimization)
debug: vendor
	@mkdir -p bin
	@echo "🐛 Building debug binary..."
	go build -mod=vendor -gcflags="all=-N -l" -o $(BINARY_FULL) ./$(CMD_DIR)
	@echo "✅ Debug build complete (ready for delve)"
ifeq ($(GOOS),windows)
	cp $(BINARY_FULL) $(BINARY_FULL).exe
endif

# Cross-platform production builds
build-all: clean quality-check vendor
	@echo "🌍 Building for all platforms..."
	@mkdir -p bin
	GOOS=darwin GOARCH=arm64 go build $(BUILD_FLAGS) -o bin/darwin-arm64/$(BINARY_NAME) ./$(CMD_DIR)
	GOOS=darwin GOARCH=amd64 go build $(BUILD_FLAGS) -o bin/macos-x64/$(BINARY_NAME) ./$(CMD_DIR)
	GOOS=linux GOARCH=amd64 go build $(BUILD_FLAGS) -o bin/linux-x64/$(BINARY_NAME) ./$(CMD_DIR)
	GOOS=windows GOARCH=amd64 go build $(BUILD_FLAGS) -o bin/windows-x64/$(BINARY_NAME).exe ./$(CMD_DIR)
	@echo "✅ Cross-platform builds complete"
	@ls -la bin/*

# Install to system
install: $(BINARY_FULL)
	@echo "📦 Installing to system..."
ifeq ($(GOOS),windows)
	@echo "ℹ️  On Windows, manually copy $(BINARY_FULL) to a directory in your PATH"
	@echo "   Example: copy $(BINARY_FULL) C:\\Windows\\System32\\"
else
	cp $(BINARY_FULL) /usr/local/bin/$(BINARY_NAME)
	@echo "✅ Installed to /usr/local/bin/$(BINARY_NAME)"
endif

# Watch mode for development
watch:
	@echo "👁️  Watching for changes..."
	@echo "Checking for watch tools..."
	@if command -v air >/dev/null 2>&1; then \
		echo "Using air..."; \
		air; \
	elif command -v watchexec >/dev/null 2>&1; then \
		echo "Using watchexec..."; \
		watchexec -e go -r -- make dev; \
	elif command -v nodemon >/dev/null 2>&1; then \
		echo "Using nodemon..."; \
		nodemon -e go --exec "make dev"; \
	else \
		echo "❌ No watch tool found. Install one of:"; \
		echo "  - air:       go install github.com/air-verse/air@latest"; \
		echo "  - watchexec: scoop install watchexec  OR  choco install watchexec"; \
		echo "  - nodemon:   npm install -g nodemon"; \
		exit 1; \
	fi

# Watch with air (recommended for Go)
watch-air:
	@echo "👁️  Watching with air..."
	@if command -v air >/dev/null 2>&1; then \
		air; \
	else \
		echo "❌ air not installed. Run: go install github.com/air-verse/air@latest"; \
		exit 1; \
	fi

# Watch with nodemon (if you have Node.js)
watch-nodemon:
	@echo "👁️  Watching with nodemon..."
	@if command -v nodemon >/dev/null 2>&1; then \
		nodemon -e go --exec "make dev"; \
	else \
		echo "❌ nodemon not installed. Run: npm install -g nodemon"; \
		exit 1; \
	fi

# Watch with watchexec (Windows/cross-platform)
watch-watchexec:
	@echo "👁️  Watching with watchexec..."
	@if command -v watchexec >/dev/null 2>&1; then \
		watchexec -e go -r -- make dev; \
	else \
		echo "❌ watchexec not installed."; \
		echo "Install via:"; \
		echo "  - Scoop:  scoop install watchexec"; \
		echo "  - Choco:  choco install watchexec"; \
		echo "  - Manual: https://github.com/watchexec/watchexec/releases"; \
		exit 1; \
	fi

# Help
help:
	@echo "gomtb-manifest Build System"
	@echo "========================"
	@echo ""
	@echo "Development:"
	@echo "  make dev              Quick development build"
	@echo "  make debug            Build with debug symbols (for delve)"
	@echo "  make all              Full build with quality checks and tests (default)"
	@echo "  make watch            Watch for changes and auto-rebuild (auto-detect tool)"
	@echo "  make watch-air        Watch with air (recommended for Go)"
	@echo "  make watch-nodemon    Watch with nodemon (if Node.js installed)"
	@echo "  make watch-watchexec  Watch with watchexec (Windows-friendly)"
	@echo ""
	@echo "Building:"
	@echo "  make build-all        Cross-platform builds with checks"
	@echo "  make install          Install to system"
	@echo ""
	@echo "Quality:"
	@echo "  make quality-check    Run fmt, vet, staticcheck, errcheck"
	@echo "  make test             Run tests"
	@echo ""
	@echo "Utilities:"
	@echo "  make check-deps       Check for required tools"
	@echo "  make deps             Install/update dependencies and tools"
	@echo "  make clean            Remove build artifacts"
	@echo "  make help             Show this help"
	@echo ""
	@echo "Examples:"
	@echo "  make              # Full build"
	@echo "  make dev          # Quick build for development"
	@echo "  make watch        # Auto-rebuild on changes"
	@echo "  make clean all    # Clean rebuild"
	@echo "  make build-all    # Build for all platforms"
	@echo ""
