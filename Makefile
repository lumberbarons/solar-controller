.PHONY: help build test clean build-frontend build-backend build-with-cgo build-linux-arm64 docker deploy

.DEFAULT_GOAL := help

# Version information
VERSION := $(shell git describe --tags --always --dirty)
BUILD_TIME := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
GIT_COMMIT := $(shell git rev-parse --short HEAD)

# Build flags
LDFLAGS := -X 'main.Version=$(VERSION)' -X 'main.BuildTime=$(BUILD_TIME)' -X 'main.GitCommit=$(GIT_COMMIT)'

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*##"; printf ""} /^[a-zA-Z_-]+:.*?##/ { printf "  %-20s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

build: build-frontend build-backend ## Build everything (frontend + backend)

build-frontend: ## Build only frontend (React app)
	cd site && npm install && npm run build
	rm -rf internal/static/build
	cp -r site/build internal/static/build

build-backend: ## Build only backend (Go binary)
	go build -ldflags="$(LDFLAGS)" -o bin/solar-controller ./cmd/controller

build-with-cgo: build-frontend ## Build backend with CGO enabled (for Solace support)
	CGO_ENABLED=1 go build -ldflags="$(LDFLAGS)" -o bin/solar-controller ./cmd/controller

build-linux-arm64: build-frontend ## Build backend for Linux ARM64 (requires native ARM64 or cross-compiler)
	CGO_ENABLED=1 GOOS=linux GOARCH=arm64 go build -ldflags="$(LDFLAGS) -s -w" -trimpath -o bin/solar-controller-linux-arm64 ./cmd/controller

test: ## Run tests
	go test ./...

clean: ## Clean build artifacts
	rm -f bin/solar-controller
	rm -f bin/solar-controller-linux-arm64
	rm -rf site/build
	rm -rf internal/static/build

docker: ## Build Docker image
	docker build -t solar-controller:$(VERSION) -t solar-controller:latest .

deploy: build-linux-arm64 ## Deploy to remote server (requires REMOTE_HOST=user@host)
	@if [ -z "$(REMOTE_HOST)" ]; then \
		echo "Error: REMOTE_HOST is required. Usage: make deploy REMOTE_HOST=user@host"; \
		exit 1; \
	fi
	@echo "Copying binary to $(REMOTE_HOST)..."
	scp bin/solar-controller-linux-arm64 $(REMOTE_HOST):/home/$$(echo $(REMOTE_HOST) | cut -d@ -f1)/solar-controller
	@echo "Installing and restarting service on remote server..."
	ssh $(REMOTE_HOST) 'sudo chown root:root solar-controller && sudo mv solar-controller /usr/bin && sudo service solar-controller restart'
	@echo "Deployment complete!"
