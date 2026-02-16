.PHONY: help dev dev-db dev-api dev-web build up down logs migrate sqlc test test-go test-web lint

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# --- Development ---

dev: dev-db dev-api dev-web ## Start all services for development

dev-db: ## Start PostgreSQL only
	docker compose up postgres -d

dev-api: ## Start API server with hot reload (requires air: go install github.com/air-verse/air@latest)
	air

dev-web: ## Start frontend dev server
	cd web && npm run dev

# --- Docker ---

build: ## Build all Docker images
	docker compose build

up: ## Start all services
	docker compose up -d

down: ## Stop all services
	docker compose down

logs: ## Tail logs from all services
	docker compose logs -f

logs-api: ## Tail API logs
	docker compose logs -f api

# --- Database ---

migrate: ## Run database migrations
	go run ./cmd/server -migrate-only

migrate-new: ## Create a new migration (usage: make migrate-new name=create_users)
	@if [ -z "$(name)" ]; then echo "Usage: make migrate-new name=create_users"; exit 1; fi
	@num=$$(printf "%06d" $$(($$(ls internal/database/migrations/*.up.sql 2>/dev/null | wc -l) + 1))); \
	touch "internal/database/migrations/$${num}_$(name).up.sql"; \
	touch "internal/database/migrations/$${num}_$(name).down.sql"; \
	echo "Created migrations: $${num}_$(name).{up,down}.sql"

# --- Code Generation ---

sqlc: ## Generate Go code from SQL queries
	sqlc generate

# --- Testing ---

test: test-go test-web ## Run all tests

test-go: ## Run Go tests
	go test ./... -v -race

test-web: ## Run frontend tests
	cd web && npm test

# --- Linting ---

lint: lint-go lint-web ## Run all linters

lint-go: ## Run Go linter
	golangci-lint run ./...

lint-web: ## Run frontend linter
	cd web && npm run lint
