.PHONY: help setup check run build lint fmt generate test test-integration migrate-up migrate-down docker-up docker-down docker-build

VERSION ?= dev
BINARY  := bin/server

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

setup: ## Install dev tools (run once after cloning)
	go install golang.org/x/tools/cmd/goimports@latest
	go install go.uber.org/mock/mockgen@latest
	@echo "Dev environment ready. Run 'make check' before opening a PR."

check: fmt lint test ## Run fmt + lint + tests (run before committing)

run: ## Run the server locally (requires .env)
	go run ./cmd/server serve

build: ## Build the server binary
	CGO_ENABLED=0 go build -ldflags="-s -w -X main.version=$(VERSION)" -o $(BINARY) ./cmd/server

lint: ## Run golangci-lint
	golangci-lint run ./...

fmt: ## Format all Go source files
	gofmt -w .
	goimports -w .

generate: ## Generate mocks (go.uber.org/mock/mockgen)
	go generate ./...

test: ## Run unit tests
	go test -count=1 ./...

test-integration: ## Run integration tests (requires Docker)
	go test -count=1 -tags integration -timeout 5m ./tests/integration/...

migrate-up: ## Apply all pending database migrations
	go run ./cmd/server migrate up

migrate-down: ## Revert all database migrations
	go run ./cmd/server migrate down

docker-up: ## Start all services in Docker (postgres + server)
	docker compose up -d

docker-down: ## Stop all Docker services and remove volumes
	docker compose down -v

docker-build: ## Build the Docker image
	docker build --build-arg VERSION=$(VERSION) -t service-template-go:$(VERSION) .

docker-observability: ## Start services with observability stack (HyperDX)
	docker compose --profile observability up -d
