# Makefile for RSS Aggregator
# This shows examples of standard build commands

# Auto-detect version info (optional)
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME ?= $(shell date -u '+%Y-%m-%d %H:%M:%S UTC')
GIT_COMMIT ?= $(shell git rev-parse HEAD 2>/dev/null || echo "unknown")

# Standard Go build (works without version info)
.PHONY: build
build:
	@echo "Building with standard go build..."
	go build -o gorssag .

# Standard Docker build (works without version info)
.PHONY: docker
docker:
	@echo "Building with standard docker build..."
	docker build -t gorssag:latest .

# Docker build with version info (optional enhancement)
.PHONY: docker-version
docker-version:
	@echo "Building with version info..."
	@echo "Version: $(VERSION)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Git Commit: $(GIT_COMMIT)"
	docker build \
		--build-arg VERSION="$(VERSION)" \
		--build-arg BUILD_TIME="$(BUILD_TIME)" \
		--build-arg GIT_COMMIT="$(GIT_COMMIT)" \
		-t gorssag:$(VERSION) \
		-t gorssag:latest \
		.

# Standard docker-compose (works without version info)
.PHONY: compose
compose:
	@echo "Building with docker-compose..."
	docker-compose up -d --build

# Docker-compose with version info (optional)
.PHONY: compose-version
compose-version:
	@echo "Building with docker-compose and version info..."
	VERSION=$(VERSION) BUILD_TIME="$(BUILD_TIME)" GIT_COMMIT=$(GIT_COMMIT) \
		docker-compose up -d --build

# Development
.PHONY: dev
dev:
	@echo "Running in development mode..."
	go run .

# Clean
.PHONY: clean
clean:
	@echo "Cleaning up..."
	docker-compose down
	docker rmi gorssag:latest 2>/dev/null || true
	rm -f gorssag

# Show version info
.PHONY: version
version:
	@echo "Version: $(VERSION)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Git Commit: $(GIT_COMMIT)"

# Help
.PHONY: help
help:
	@echo "Available commands:"
	@echo "  build          - Standard Go build"
	@echo "  docker         - Standard Docker build" 
	@echo "  docker-version - Docker build with version info"
	@echo "  compose        - Standard docker-compose build"
	@echo "  compose-version- Docker-compose with version info"
	@echo "  dev            - Run in development mode"
	@echo "  clean          - Clean up containers and images"
	@echo "  version        - Show version info"
	@echo "  help           - Show this help"
