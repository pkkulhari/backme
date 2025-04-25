.PHONY: build clean release

# Version from git tag
VERSION := $(shell git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.1")
VERSION_NUM := $(shell echo $(VERSION) | sed 's/^v//')

BINARY_NAME=backme
MAIN_PACKAGE=./cmd
BUILD_DIR=bin

build:
	@echo "Building $(BINARY_NAME) $(VERSION) for linux/amd64..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PACKAGE)/*.go
	@echo "Done!"

release: clean build
	@echo "Creating release artifacts..."
	@cd $(BUILD_DIR) && tar -czf $(BINARY_NAME)-linux-amd64.tar.gz $(BINARY_NAME)-linux-amd64
	@echo "Release artifacts created in $(BUILD_DIR)/"
	@echo "You can now run: gh release create $(VERSION) $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64.tar.gz"

clean:
	@echo "Cleaning build directory..."
	@rm -rf $(BUILD_DIR)
	@echo "Done!"

help:
	@echo "Available targets:"
	@echo "  build   - Build binary for linux/amd64"
	@echo "  release - Create release artifacts"
	@echo "  clean   - Clean build directory"
	@echo "  help    - Show this help message" 