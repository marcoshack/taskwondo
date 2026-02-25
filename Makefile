.PHONY: build push help dev dev-db dev-api dev-web up down logs logs-api migrate migrate-new test test-e2e test-e2e-dev test-e2e-report check-env export import release build-mcp

# Required environment variables (checked by sourcing .env)
REQUIRED_VARS := POSTGRES_USER POSTGRES_PASSWORD MINIO_ROOT_USER MINIO_ROOT_PASSWORD JWT_SECRET DATABASE_URL STORAGE_ACCESS_KEY STORAGE_SECRET_KEY

# Colors
CYAN := \033[36m
GREEN := \033[32m
RESET := \033[0m

build: test build-mcp ## Build all Docker images and MCP server
	@echo ""
	@printf "$(CYAN)## Building Docker images...$(RESET)\n"
	docker compose build
	@printf "$(GREEN)## Docker images built successfully$(RESET)\n"

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
	(set -a && . ./.env && set +a && export DISCORD_REDIRECT_URI=http://localhost:5173/auth/discord/callback && cd api && air) & \
	(cd web && npm run dev -- --host) & \
	wait

dev-services: check-env ## Start PostgreSQL and MinIO
	@echo ""
	@printf "$(CYAN)## Starting dev services (PostgreSQL + MinIO)...$(RESET)\n"
	docker compose up postgres minio minio-init -d
	@printf "$(GREEN)## Dev services started$(RESET)\n"

dev-db: dev-services ## Alias for dev-services (legacy)

dev-api: check-env dev-services ## Start API server with hot reload (requires air: go install github.com/air-verse/air@latest)
	set -a && . ./.env && set +a && export DISCORD_REDIRECT_URI=http://localhost:5173/auth/discord/callback && cd api && air

dev-web: ## Start frontend dev server (Vite on :5173, proxies /api to :8080)
	cd web && npm run dev

# --- Docker ---

up: check-env ## Start all services
	@echo ""
	@printf "$(CYAN)## Starting all services...$(RESET)\n"
	docker compose up -d
	@printf "$(GREEN)## All services started$(RESET)\n"

down: ## Stop all services
	@echo ""
	@printf "$(CYAN)## Stopping all services...$(RESET)\n"
	docker compose down
	@printf "$(GREEN)## All services stopped$(RESET)\n"

logs: ## Tail logs from all services
	docker compose logs -f

logs-api: ## Tail API logs
	docker compose logs -f api

push: ## Push images to GHCR (usage: make push or IMAGE_TAG=v1.0.0 make push)
	@echo ""
	@printf "$(CYAN)## Pushing images to registry...$(RESET)\n"
	docker compose push api web
	@printf "$(GREEN)## Images pushed successfully$(RESET)\n"

# --- Database ---

migrate: check-env ## Run database migrations
	@echo ""
	@printf "$(CYAN)## Running database migrations...$(RESET)\n"
	set -a && . ./.env && set +a && cd api && go run ./cmd/server -migrate-only
	@printf "$(GREEN)## Migrations completed$(RESET)\n"

migrate-new: ## Create a new migration (usage: make migrate-new name=create_users)
	@if [ -z "$(name)" ]; then echo "Usage: make migrate-new name=create_users"; exit 1; fi
	@num=$$(printf "%06d" $$(($$(ls api/internal/database/migrations/*.up.sql 2>/dev/null | wc -l) + 1))); \
	touch "api/internal/database/migrations/$${num}_$(name).up.sql"; \
	touch "api/internal/database/migrations/$${num}_$(name).down.sql"; \
	echo "Created migrations: $${num}_$(name).{up,down}.sql"

# --- Data Export/Import ---

export: check-env ## Export all data to backups/taskwondo-export.tar.gz
	@echo ""
	@printf "$(CYAN)## Exporting data...$(RESET)\n"
	mkdir -p backups
	docker compose run --rm export
	@printf "$(GREEN)## Export completed$(RESET)\n"

import: check-env ## Import data from backups/ (IMPORT_FILE=filename.tar.gz)
	@echo ""
	@printf "$(CYAN)## Importing data...$(RESET)\n"
	docker compose run --rm import
	@printf "$(GREEN)## Import completed$(RESET)\n"

# --- Release ---

RELEASE_VERSION ?=

release: ## Build release tarballs (usage: make release or RELEASE_VERSION=1.0.0 make release)
	@if [ -z "$(RELEASE_VERSION)" ]; then \
		printf "Release version (e.g. 1.0.0): "; \
		read ver; \
		if [ -z "$$ver" ]; then echo "Error: version is required"; exit 1; fi; \
		$(MAKE) _release VERSION=$$ver; \
	else \
		$(MAKE) _release VERSION=$(RELEASE_VERSION); \
	fi

_release:
	@echo ""
	@printf "$(CYAN)## Building release v$(VERSION)...$(RESET)\n"
	rm -rf build/release
	mkdir -p build/release/taskwondo-$(VERSION)/bin build/release/taskwondo-$(VERSION)/html
	@printf "$(CYAN)## Building API binary (Docker)...$(RESET)\n"
	docker build -f docker/Dockerfile.api --target builder -t taskwondo-api-builder api
	docker create --name taskwondo-api-extract taskwondo-api-builder true
	docker cp taskwondo-api-extract:/bin/taskwondo build/release/taskwondo-$(VERSION)/bin/taskwondo
	docker rm taskwondo-api-extract
	@printf "$(CYAN)## Building Web bundle (Docker)...$(RESET)\n"
	docker build -f docker/Dockerfile.web --target builder -t taskwondo-web-builder .
	docker create --name taskwondo-web-extract taskwondo-web-builder true
	docker cp taskwondo-web-extract:/src/dist/. build/release/taskwondo-$(VERSION)/html/
	docker rm taskwondo-web-extract
	cp .env.template build/release/taskwondo-$(VERSION)/.env.template
	cp docker/nginx.conf build/release/taskwondo-$(VERSION)/nginx.conf
	cp docs/install/manual-install.md build/release/taskwondo-$(VERSION)/README.md
	@printf "$(CYAN)## Packaging tarball...$(RESET)\n"
	tar -czf build/release/taskwondo-$(VERSION).tar.gz -C build/release taskwondo-$(VERSION)
	@echo ""
	@echo "Release artifact:"
	@ls -lh build/release/taskwondo-$(VERSION).tar.gz
	@echo ""
	@echo "Contents:"
	@tar -tzf build/release/taskwondo-$(VERSION).tar.gz | head -20
	@printf "$(GREEN)## Release v$(VERSION) built successfully$(RESET)\n"

# --- MCP Server ---

build-mcp: ## Build the MCP server binary
	@echo ""
	@printf "$(CYAN)## Building MCP server...$(RESET)\n"
	$(MAKE) -C mcp build
	@printf "$(GREEN)## MCP server built successfully$(RESET)\n"

# --- Testing ---

test: ## Run all tests
	@echo ""
	@printf "$(CYAN)## Running Go tests...$(RESET)\n"
	cd api && go test ./... -v -race
	@printf "$(GREEN)## All tests passed$(RESET)\n"

test-e2e: ## Run E2E tests in isolated Docker stack (no host deps)
	@echo ""
	@printf "$(CYAN)## Running E2E tests (Docker)...$(RESET)\n"
	bash scripts/e2e-docker.sh
	@printf "$(GREEN)## E2E tests passed$(RESET)\n"

test-e2e-dev: ## Run E2E tests against local dev server (localhost:5173)
	@echo ""
	@printf "$(CYAN)## Running E2E tests (dev)...$(RESET)\n"
	cd e2e && npx playwright test
	@printf "$(GREEN)## E2E tests passed$(RESET)\n"

test-e2e-report: ## Serve the last E2E HTML report at http://localhost:9323
	@echo "Serving report at http://localhost:9323"
	@echo "Press Ctrl+C to stop"
	docker run --rm -p 9323:80 -v "$$(pwd)/e2e/playwright-report:/usr/share/nginx/html:ro" nginx:alpine
