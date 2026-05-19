# Makefile for Telegram to VK Content Transfer

.PHONY: help build run test clean generate-config lint vet

# Go parameters
GO := go
GOFMT := gofmt
GOLINT := golint
GOVET := $(GO) vet
BINARY_NAME := transfer
MAIN_PACKAGE := ./cmd/transfer

help: ## Show this help
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the application
	$(GO) build -o bin/$(BINARY_NAME) $(MAIN_PACKAGE)

run: ## Run the application (requires config)
	$(GO) run $(MAIN_PACKAGE)

test: ## Run unit tests
	$(GO) test ./...

test-verbose: ## Run unit tests with verbose output
	$(GO) test -v ./...

clean: ## Clean build artifacts
	rm -rf bin/
	rm -rf temp/
	rm -rf logs/
	rm -rf checkpoints/

generate-config: ## Generate example configuration file
	@mkdir -p configs
	$(GO) run $(MAIN_PACKAGE) --generate-config configs/config.example.yaml

lint: ## Run linter (requires golint)
	$(GOLINT) ./...

vet: ## Run go vet
	$(GOVET) ./...

fmt: ## Format Go code
	$(GOFMT) -w .

fmt-check: ## Check formatting without applying
	$(GOFMT) -d .

deps: ## Download dependencies
	$(GO) mod download

tidy: ## Tidy go.mod
	$(GO) mod tidy

install-tools: ## Install development tools
	$(GO) install golang.org/x/lint/golint@latest

all: deps fmt vet lint build ## Run all checks and build

# Development targets
dev-run: build ## Build and run
	./bin/$(BINARY_NAME)

dev-test: test ## Run tests

# Docker targets (optional)
docker-build: ## Build Docker image
	docker build -t tg2vk .

docker-run: ## Run Docker container
	docker run --rm -v $(PWD)/configs:/app/configs tg2vk

# Platform-specific builds
build-linux: ## Build for Linux
	GOOS=linux GOARCH=amd64 $(GO) build -o bin/$(BINARY_NAME)-linux $(MAIN_PACKAGE)

build-windows: ## Build for Windows
	GOOS=windows GOARCH=amd64 $(GO) build -o bin/$(BINARY_NAME).exe $(MAIN_PACKAGE)

build-mac: ## Build for macOS
	GOOS=darwin GOARCH=amd64 $(GO) build -o bin/$(BINARY_NAME)-mac $(MAIN_PACKAGE)