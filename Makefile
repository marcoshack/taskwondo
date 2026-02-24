.PHONY: build push help dev dev-db dev-api dev-web up down logs logs-api migrate migrate-new test test-e2e test-e2e-report check-env export import release

# Required environment variables (checked by sourcing .env)
REQUIRED_VARS := POSTGRES_USER POSTGRES_PASSWORD MINIO_ROOT_USER MINIO_ROOT_PASSWORD JWT_SECRET DATABASE_URL STORAGE_ACCESS_KEY STORAGE_SECRET_KEY

build: test ## Build all Docker images
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
	(set -a && . ./.env && set +a && export DISCORD_REDIRECT_URI=http://localhost:5173/auth/discord/callback && cd api && air) & \
	(cd web && npm run dev -- --host) & \
	wait

dev-services: check-env ## Start PostgreSQL and MinIO
	docker compose up postgres minio minio-init -d

dev-db: dev-services ## Alias for dev-services (legacy)

dev-api: check-env dev-services ## Start API server with hot reload (requires air: go install github.com/air-verse/air@latest)
	set -a && . ./.env && set +a && export DISCORD_REDIRECT_URI=http://localhost:5173/auth/discord/callback && cd api && air

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

export: check-env ## Export all data to backups/taskwondo-export.tar.gz
	mkdir -p backups
	docker compose run --rm export

import: check-env ## Import data from backups/ (IMPORT_FILE=filename.tar.gz)
	docker compose run --rm import

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
	@echo "==> Building release v$(VERSION)..."
	rm -rf build/release
	mkdir -p build/release/taskwondo-$(VERSION)/bin build/release/taskwondo-$(VERSION)/html
	@echo "==> Building API binary (Docker)..."
	docker build -f docker/Dockerfile.api --target builder -t taskwondo-api-builder api
	docker create --name taskwondo-api-extract taskwondo-api-builder true
	docker cp taskwondo-api-extract:/bin/taskwondo build/release/taskwondo-$(VERSION)/bin/taskwondo
	docker rm taskwondo-api-extract
	@echo "==> Building Web bundle (Docker)..."
	docker build -f docker/Dockerfile.web --target builder -t taskwondo-web-builder .
	docker create --name taskwondo-web-extract taskwondo-web-builder true
	docker cp taskwondo-web-extract:/src/dist/. build/release/taskwondo-$(VERSION)/html/
	docker rm taskwondo-web-extract
	cp .env.template build/release/taskwondo-$(VERSION)/.env.template
	cp docker/nginx.conf build/release/taskwondo-$(VERSION)/nginx.conf
	cp docs/install/manual-install.md build/release/taskwondo-$(VERSION)/README.md
	@echo "==> Packaging tarball..."
	tar -czf build/release/taskwondo-$(VERSION).tar.gz -C build/release taskwondo-$(VERSION)
	@echo ""
	@echo "Release artifact:"
	@ls -lh build/release/taskwondo-$(VERSION).tar.gz
	@echo ""
	@echo "Contents:"
	@tar -tzf build/release/taskwondo-$(VERSION).tar.gz | head -20

# --- Testing ---

test: ## Run all tests
	cd api && go test ./... -v -race

test-e2e: ## Run end-to-end tests (headless)
	cd e2e && npx playwright test

test-e2e-report: ## Open the last e2e test HTML report
	cd e2e && npx playwright show-report
