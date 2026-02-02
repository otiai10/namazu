# Namazu Makefile

REGISTRY := us-west1-docker.pkg.dev/namazu-live/namazu
IMAGE := $(REGISTRY)/namazu
TAG := latest
CONTAINER_RUNTIME := $(shell command -v docker 2>/dev/null || command -v podman 2>/dev/null)

.PHONY: help build push test test-e2e ship

help: ## Show this help
	@grep -E '^[a-zA-Z0-9_-]+:.*## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*## "}; {printf "\033[36m%-10s\033[0m %s\n", $$1, $$2}'

build: ## Build Docker image
	$(CONTAINER_RUNTIME) build --platform linux/amd64 -t $(IMAGE):$(TAG) .

push: ## Push image to registry
	$(CONTAINER_RUNTIME) push $(IMAGE):$(TAG)

test: ## Run Go tests
	go test ./...

test-e2e: ## Run E2E tests
	./scripts/e2e-test.sh

ship: build push ## Build and push image
