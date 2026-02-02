# Namazu Makefile
# Common commands for development and deployment

# Configuration
REGISTRY := us-west1-docker.pkg.dev/namazu-live/namazu
IMAGE := $(REGISTRY)/namazu
TAG := latest
ZONE := us-west1-b
INSTANCE := namazu-dev-instance

# Detect container runtime (docker or podman)
CONTAINER_RUNTIME := $(shell command -v docker 2>/dev/null || command -v podman 2>/dev/null)

.PHONY: help build push deploy restart logs health test clean login

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

login: ## Authenticate to Artifact Registry
	gcloud auth print-access-token | $(CONTAINER_RUNTIME) login -u oauth2accesstoken --password-stdin us-west1-docker.pkg.dev

build: ## Build Docker image for linux/amd64
	$(CONTAINER_RUNTIME) build --platform linux/amd64 -t $(IMAGE):$(TAG) .

push: ## Push Docker image to Artifact Registry
	$(CONTAINER_RUNTIME) push $(IMAGE):$(TAG)

deploy: ## Deploy infrastructure with Pulumi
	pulumi -C infra up

preview: ## Preview infrastructure changes
	pulumi -C infra preview

restart: ## Restart the GCE instance
	gcloud compute instances reset $(INSTANCE) --zone=$(ZONE)

logs: ## View instance startup logs
	gcloud compute instances get-serial-port-output $(INSTANCE) --zone=$(ZONE) 2>&1 | tail -100

logs-startup: ## View startup script logs
	gcloud compute instances get-serial-port-output $(INSTANCE) --zone=$(ZONE) 2>&1 | grep -A 20 "startup-script"

ssh: ## SSH into the instance
	gcloud compute ssh $(INSTANCE) --zone=$(ZONE) --tunnel-through-iap

health: ## Check health endpoint
	curl -s http://$(shell pulumi -C infra stack output externalIp 2>/dev/null):8080/health

docker-logs: ## View container logs (via SSH)
	gcloud compute ssh $(INSTANCE) --zone=$(ZONE) --tunnel-through-iap --command="docker logs namazu"

docker-restart: ## Restart container (stop, rm, run)
	gcloud compute ssh $(INSTANCE) --zone=$(ZONE) --tunnel-through-iap --command="\
		docker stop namazu 2>/dev/null || true && \
		docker rm namazu 2>/dev/null || true && \
		docker pull $(IMAGE):$(TAG) && \
		docker run -d --name namazu --restart=always -p 8080:8080 \
			-e NAMAZU_SOURCE_TYPE=p2pquake \
			-e NAMAZU_SOURCE_ENDPOINT=wss://api.p2pquake.net/v2/ws \
			-e NAMAZU_API_ADDR=:8080 \
			-e NAMAZU_STORE_PROJECT_ID=namazu-live \
			-e NAMAZU_STORE_DATABASE=namazu-dev \
			$(IMAGE):$(TAG)"

# Combined commands
ship: build push restart ## Build, push, and restart (full deploy)

test: ## Run tests
	go test ./...

test-e2e: ## Run E2E tests
	./scripts/e2e-test.sh

clean: ## Clean build artifacts
	rm -rf ./cmd/namazu/static
	go clean

# Development
dev: ## Run locally
	go run ./cmd/namazu --test-mode

dev-web: ## Run frontend dev server
	cd web && pnpm dev
