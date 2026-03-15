# ==============================================================================
# Variables & Paths
# ==============================================================================
BINARY_NAME=cmdChroma
MAIN_PATH=./cmd/chroma
BUILD_DIR=./dist
GOARCH=amd64
GOOS=linux

# Absolute paths for reliability with CGO and Shared Libraries
PROJECT_DIR      := $(CURDIR)
TOKENIZER_LIB_DIR := $(PROJECT_DIR)/tokenizerLib
ONNX_LIB_DIR      := $(PROJECT_DIR)/models/onnx_runtime/lib

# ==============================================================================
# CGO Settings (Required for Tokenizers & ONNX)
# ==============================================================================
export CGO_ENABLED=1

# Only set tokenizers linker flags if the library directory exists.
# This allows building on systems that do not have libtokenizers installed.
ifeq ($(wildcard $(TOKENIZER_LIB_DIR)/libtokenizers.*),)
	# No tokenizer library found: build without tokenizers support.
	export CGO_LDFLAGS=
else
	export CGO_LDFLAGS=-L$(TOKENIZER_LIB_DIR) -ltokenizers -lstdc++
endif

# ==============================================================================
# Development Targets
# ==============================================================================

.PHONY: all
all: clean build ## Clean and build the CLI binary

.PHONY: help
help: ## Show this help message
	@echo "Usage: make [target]"
	@echo ""
	@echo "Available Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

.PHONY: run
run: ## Run in dev mode (Handles LD_LIBRARY_PATH for ONNX)
	@echo "Running in development mode..."
	LD_LIBRARY_PATH="$(ONNX_LIB_DIR):$(LD_LIBRARY_PATH)" go run $(MAIN_PATH)

# .PHONY: build
# build: ## Build the binary for current OS
# 	@echo "Building $(BINARY_NAME)..."
# 	@mkdir -p $(BUILD_DIR)
# 	go build -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
# 	@echo "✅ Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# ==============================================================================
# Revised Build Logic with RPATH
# ==============================================================================

# $ORIGIN is a special linker variable that represents the binary's directory.
# We use single quotes and double dollar signs to prevent the Shell/Make 
# from evaluating it before it reaches the linker.
# Use this exact escaping for $ORIGIN
# Use FOUR dollar signs to survive the Make -> Shell -> Linker journey
# Use single $ for the variable definition, but we will escape it in the command
# Use single $ for the variable definition, but we will escape it in the command
# Define the RPATH relative to the binary's location
# We use '$$ORIGIN' so Make passes '$ORIGIN' to the shell
RPATH_VALUE = '$$ORIGIN/../models/onnx_runtime/lib:$$ORIGIN/../models/tokenizerLib'

.PHONY: build
build:
	@echo "Building with Go Linker RPATH..."
	@mkdir -p ./dist
	CGO_ENABLED=1 \
	go build -ldflags="-r $(RPATH_VALUE)" -o ./dist/$(BINARY_NAME) ./cmd/chroma

.PHONY: clean
clean: ## Remove build artifacts and coverage files
	@echo "Cleaning up..."
	rm -rf $(BUILD_DIR) coverage.out coverage.html
	@echo "✅ Clean complete"

# ==============================================================================
# Integration Testing (Venom)
# ==============================================================================

.PHONY: venom
venom: build ## Build and run Venom integration tests
	@echo "🚀 Preparing Venom Integration Tests..."
	@# Ensure the script is executable
	@chmod +x ./.ci/scripts/run-venom.sh
	@# Run the script with necessary Library Paths for CGO/ONNX
	LD_LIBRARY_PATH="$(ONNX_LIB_DIR):$(LD_LIBRARY_PATH)"; \
	./.ci/scripts/run-venom.sh

.PHONY: venom-clean
venom-clean: ## Remove Venom logs and reports
	@echo "🧹 Cleaning Venom artifacts..."
	rm -rf .ci/logs/*
	@echo "✅ Clean complete"

# ==============================================================================
# Cross-Compilation Targets
# ==============================================================================

.PHONY: build-linux
build-linux: ## Build for Linux (amd64)
	@echo "Building for Linux..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)

.PHONY: build-darwin
build-darwin: ## Build for macOS (amd64)
	@echo "Building for macOS..."
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME-darwin-amd64) $(MAIN_PATH)

.PHONY: build-windows
build-windows: ## Build for Windows (amd64)
	@echo "Building for Windows..."
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME).exe $(MAIN_PATH)

.PHONY: build-all
build-all: build-linux build-darwin build-windows ## Build for all platforms

# ==============================================================================
# Maintenance & Tooling
# ==============================================================================

.PHONY: deps
deps: ## Download and tidy go modules
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

.PHONY: test
test: ## Run unit tests
	@echo "Running tests..."
	go test -v ./...

.PHONY: generate
generate: ## Run go code generation (if any)
	@echo "Running go generate..."
	go generate ./...

.PHONY: fmt
fmt: ## Format source code
	go fmt ./...

.PHONY: lint
lint: ## Run linter (golangci-lint)
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run; \
	else \
		go vet ./...; \
	fi

.PHONY: install
install: build ## Install binary to /usr/local/bin
	sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/

.PHONY: uninstall
uninstall: ## Remove binary from /usr/local/bin
	sudo rm -f /usr/local/bin/$(BINARY_NAME)

