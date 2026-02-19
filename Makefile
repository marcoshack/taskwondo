.PHONY: build push help dev dev-db dev-api dev-web up down logs logs-api migrate migrate-new test check-env export import

# Required environment variables (checked by sourcing .env)
REQUIRED_VARS := POSTGRES_USER POSTGRES_PASSWORD MINIO_ROOT_USER MINIO_ROOT_PASSWORD JWT_SECRET DATABASE_URL STORAGE_ACCESS_KEY STORAGE_SECRET_KEY

build: ## Build all Docker images
	docker compose build

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

check-env: ## Verify .env exists and has all required variables
	@if [ ! -f .env ]; then \
		printf "\033[31mError: .env file not found\033[0m\n"; \
		echo "Run: cp .env.template .env"; \
		exit 1; \
	fi; \
	missing=""; \
	set -a && . ./.env && set +a; \
	for var in $(REQUIRED_VARS); do \
		val=$$(eval echo "\$$$$var"); \
		if [ -z "$$val" ]; then missing="$$missing $$var"; fi; \
	done; \
	if [ -n "$$missing" ]; then \
		printf "\033[31mError: missing required environment variables:%s\033[0m\n" "$$missing"; \
		echo "Check .env and set all required values (see .env.template)."; \
		exit 1; \
	fi

# --- Development ---

dev: check-env dev-services ## Start all services for development (API + Web)
	trap 'kill 0' EXIT; \
	(set -a && . ./.env && set +a && cd api && air) & \
	(cd web && npm run dev) & \
	wait

dev-services: check-env ## Start PostgreSQL and MinIO
	docker compose up postgres minio minio-init -d

dev-db: dev-services ## Alias for dev-services (legacy)

dev-api: check-env dev-services ## Start API server with hot reload (requires air: go install github.com/air-verse/air@latest)
	set -a && . ./.env && set +a && cd api && air

dev-web: ## Start frontend dev server (Vite on :5173, proxies /api to :8080)
	cd web && npm run dev

# --- Docker ---

up: check-env ## Start all services
	docker compose up -d

down: ## Stop all services
	docker compose down

logs: ## Tail logs from all services
	docker compose logs -f

logs-api: ## Tail API logs
	docker compose logs -f api

push: ## Push images to GHCR (usage: make push or IMAGE_TAG=v1.0.0 make push)
	docker compose push api web

# --- Database ---

migrate: check-env ## Run database migrations
	set -a && . ./.env && set +a && cd api && go run ./cmd/server -migrate-only

migrate-new: ## Create a new migration (usage: make migrate-new name=create_users)
	@if [ -z "$(name)" ]; then echo "Usage: make migrate-new name=create_users"; exit 1; fi
	@num=$$(printf "%06d" $$(($$(ls api/internal/database/migrations/*.up.sql 2>/dev/null | wc -l) + 1))); \
	touch "api/internal/database/migrations/$${num}_$(name).up.sql"; \
	touch "api/internal/database/migrations/$${num}_$(name).down.sql"; \
	echo "Created migrations: $${num}_$(name).{up,down}.sql"

# --- Data Export/Import ---

export: check-env ## Export all data to backups/trackforge-export.tar.gz
	mkdir -p backups
	docker compose run --rm export

import: check-env ## Import data from backups/ (IMPORT_FILE=filename.tar.gz)
	docker compose run --rm import

# --- Testing ---

test: ## Run all tests
	cd api && go test ./... -v -race
