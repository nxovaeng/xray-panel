.PHONY: all build clean test run dev install help

# Variables
VERSION ?= v1.0.0
BINARY_NAME = panel
OUTPUT_DIR = dist
MAIN_PATH = ./cmd/panel

# Go parameters
GOCMD = go
GOBUILD = $(GOCMD) build
GOCLEAN = $(GOCMD) clean
GOTEST = $(GOCMD) test
GOGET = $(GOCMD) get
GOMOD = $(GOCMD) mod

# Build flags
LDFLAGS = -s -w -X main.Version=$(VERSION)
BUILD_FLAGS = -v -trimpath -ldflags="$(LDFLAGS)"

# Default target
all: clean build

# Help target
help:
	@echo "Xray Panel Build System"
	@echo ""
	@echo "Usage:"
	@echo "  make build              Build for current platform"
	@echo "  make build-all          Build for all platforms"
	@echo "  make build-linux        Build for Linux (amd64 & arm64)"
	@echo "  make build-windows      Build for Windows (amd64 & arm64)"
	@echo "  make build-darwin       Build for macOS (amd64 & arm64)"
	@echo "  make clean              Clean build artifacts"
	@echo "  make test               Run tests"
	@echo "  make run                Run the application"
	@echo "  make dev                Run in development mode"
	@echo "  make install            Install dependencies"
	@echo ""
	@echo "Variables:"
	@echo "  VERSION=v1.0.0          Set version (default: v1.0.0)"

# Build for current platform
build:
	@echo "Building for current platform..."
	@mkdir -p $(OUTPUT_DIR)
	$(GOBUILD) $(BUILD_FLAGS) -o $(OUTPUT_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "Build complete: $(OUTPUT_DIR)/$(BINARY_NAME)"

# Build for all platforms
build-all: build-linux build-windows build-darwin
	@echo "All builds complete!"
	@ls -lh $(OUTPUT_DIR)/

# Build for Linux
build-linux:
	@echo "Building for Linux..."
	@mkdir -p $(OUTPUT_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) -o $(OUTPUT_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GOBUILD) $(BUILD_FLAGS) -o $(OUTPUT_DIR)/$(BINARY_NAME)-linux-arm64 $(MAIN_PATH)
	@echo "Linux builds complete"

# Build for Windows
build-windows:
	@echo "Building for Windows..."
	@mkdir -p $(OUTPUT_DIR)
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) -o $(OUTPUT_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PATH)
	CGO_ENABLED=0 GOOS=windows GOARCH=arm64 $(GOBUILD) $(BUILD_FLAGS) -o $(OUTPUT_DIR)/$(BINARY_NAME)-windows-arm64.exe $(MAIN_PATH)
	@echo "Windows builds complete"

# Build for macOS
build-darwin:
	@echo "Building for macOS..."
	@mkdir -p $(OUTPUT_DIR)
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) -o $(OUTPUT_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GOBUILD) $(BUILD_FLAGS) -o $(OUTPUT_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_PATH)
	@echo "macOS builds complete"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@$(GOCLEAN)
	@rm -rf $(OUTPUT_DIR)
	@rm -f $(BINARY_NAME) $(BINARY_NAME).exe
	@echo "Clean complete"

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

# Run the application
run: build
	@echo "Running application..."
	@./$(OUTPUT_DIR)/$(BINARY_NAME)

# Run in development mode
dev:
	@echo "Running in development mode..."
	$(GOCMD) run $(MAIN_PATH)

# Install dependencies
install:
	@echo "Installing dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy
	@echo "Dependencies installed"

# Update dependencies
update:
	@echo "Updating dependencies..."
	$(GOMOD) tidy
	$(GOGET) -u ./...
	@echo "Dependencies updated"

# Show version
version:
	@echo "Version: $(VERSION)"

# Build with race detector (for development)
build-race:
	@echo "Building with race detector..."
	@mkdir -p $(OUTPUT_DIR)
	$(GOBUILD) -race -o $(OUTPUT_DIR)/$(BINARY_NAME)-race $(MAIN_PATH)
	@echo "Race detector build complete"

# Create release packages
package: build-all
	@echo "Creating release packages..."
	@cd $(OUTPUT_DIR) && \
	for file in panel-*; do \
		if [[ $$file == *.exe ]]; then \
			zip "$${file%.exe}.zip" "$$file"; \
		else \
			tar czf "$$file.tar.gz" "$$file"; \
		fi; \
		sha256sum "$$file" > "$$file.sha256"; \
	done
	@echo "Packages created"
