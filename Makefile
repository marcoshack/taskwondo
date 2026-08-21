.PHONY: build push help setup setup-shell dev dev-stop dev-db dev-api dev-web dev-worker up down logs logs-api migrate migrate-new test test-api test-web test-e2e test-e2e-dev test-e2e-report check-env check-tools check-tools-api check-tools-web check-toolchain check-air release build-mcp build-mcp-linux build-mcp-darwin build-mcp-windows build-mcpb build-worker lint-ci

# Required environment variables (checked by sourcing .env)
REQUIRED_VARS := POSTGRES_USER POSTGRES_PASSWORD MINIO_ROOT_USER MINIO_ROOT_PASSWORD JWT_SECRET DATABASE_URL STORAGE_ACCESS_KEY STORAGE_SECRET_KEY

# mise.toml pins every runtime and CLI tool the repository builds with, and
# `make setup` installs them from it. Its bin directories go first on PATH here
# so `go`, `node`, `npm` and `air` resolve to the pinned versions in a shell
# where `mise activate` has never run — which is most of them, and every shell
# that opened before setup did. Sub-makes inherit the exported PATH, so mcp/ and
# mcpb/ build with the same toolchain.
#
# Absent mise (CI installs its runtimes with setup-go and setup-node) this is a
# no-op, and scripts/check-toolchain.sh is what holds those to the same pins.
MISE := $(shell command -v mise 2>/dev/null)
ifneq ($(MISE),)
export PATH := $(shell mise bin-paths 2>/dev/null | tr '\n' ':')$(PATH)
endif

# Colors
CYAN := \033[36m
GREEN := \033[32m
RESET := \033[0m

build: test build-worker build-mcp ## Build all Docker images, worker, MCP server, and MCPB bundle
	@echo ""
	@printf "$(CYAN)## Building Docker images...$(RESET)\n"
	docker compose build
	@printf "$(GREEN)## Docker images built successfully$(RESET)\n"

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# mise is the only thing this asks you to have installed. Go, Node, npm and air
# all come from mise.toml, at the versions pinned there, so a fresh checkout
# builds with the same toolchain everywhere instead of with whatever each
# machine happens to carry.
setup: ## Set up local dev environment (install the toolchain with mise, generate .env, configure git hooks)
	@printf "$(CYAN)## Checking for mise...$(RESET)\n"
	@if ! command -v mise >/dev/null 2>&1; then \
		printf "\033[31m## Error: mise is not installed.\033[0m\n"; \
		echo "mise installs every runtime and tool this repository needs (see mise.toml)."; \
		echo "  curl https://mise.run | sh"; \
		echo "  https://mise.jdx.dev/getting-started.html for other install methods"; \
		echo "Then re-run 'make setup'."; \
		exit 1; \
	fi
	@printf "$(GREEN)## mise %s$(RESET)\n" "$$(mise version)"
	@printf "$(CYAN)## Installing the toolchain from mise.toml...$(RESET)\n"
	@mise trust --quiet
	@mise install
	@mise ls --current
	@printf "$(CYAN)## Checking toolchain consistency...$(RESET)\n"
	@mise exec -- ./scripts/check-toolchain.sh
	@printf "$(GREEN)## Toolchain OK$(RESET)\n"
	@missing=""; \
	for cmd in docker openssl; do \
		if ! command -v $$cmd >/dev/null 2>&1; then missing="$$missing $$cmd"; fi; \
	done; \
	if [ -n "$$missing" ]; then \
		printf "\033[33m## mise cannot provide these, and they are not installed:%s\033[0m\n" "$$missing"; \
		printf "\033[33m##   docker  — 'make dev', 'make up' and 'make test-e2e' need it: https://docs.docker.com/get-docker/\033[0m\n"; \
		printf "\033[33m##   openssl — ./install.sh generates .env secrets with it (usually preinstalled)\033[0m\n"; \
	fi
	@if [ ! -f .env ]; then \
		printf "$(CYAN)## Generating .env (running ./install.sh --manual-setup -y)...$(RESET)\n"; \
		./install.sh --manual-setup -y; \
		printf "$(GREEN)## .env generated$(RESET)\n"; \
	else \
		printf "$(GREEN)## .env already exists, leaving it as-is$(RESET)\n"; \
	fi
	@missing=""; \
	set -a && . ./.env && set +a; \
	for var in $(REQUIRED_VARS); do \
		val=$$(eval echo "\$$$$var"); \
		if [ -z "$$val" ]; then missing="$$missing $$var"; fi; \
	done; \
	if [ -n "$$missing" ]; then \
		printf "\033[31m## Error: .env is missing required variables:%s\033[0m\n" "$$missing"; \
		echo "Edit .env and set the missing values (see .env.template), or delete .env and re-run 'make setup' to regenerate."; \
		exit 1; \
	fi
	@printf "$(GREEN)## .env has all required variables$(RESET)\n"
	@git config core.hooksPath .githooks
	@printf "$(GREEN)## Git hooks configured (.githooks/)$(RESET)\n"
	@printf "$(GREEN)## Setup complete — run 'make dev' to start the local stack$(RESET)\n"
	@echo ""
	@$(MAKE) --no-print-directory setup-shell

setup-shell: ## Offer to activate mise in your shell, so the pins apply outside make too
	@./scripts/setup-shell.sh

# Runs with or without mise: the pins are compared against what the go.mod
# files, the CI workflow and the Dockerfiles declare either way, and the running
# Go, Node and npm are checked too wherever mise is what installed them.
check-toolchain: ## Verify mise.toml and every version the components declare agree
	@./scripts/check-toolchain.sh

check-tools-api: check-toolchain ## Verify tools required for Go API tests/builds (go)
	@if ! command -v go >/dev/null 2>&1; then \
		printf "\033[31mError: missing required tool: go\033[0m\n"; \
		echo "Run: make setup  (go comes from mise.toml)"; \
		exit 1; \
	fi

check-tools-web: check-toolchain ## Verify tools required for frontend tests/builds (node, npm >= 11.10)
	@missing=""; \
	for cmd in node npm; do \
		if ! command -v $$cmd >/dev/null 2>&1; then missing="$$missing $$cmd"; fi; \
	done; \
	if [ -n "$$missing" ]; then \
		printf "\033[31mError: missing required tools:%s\033[0m\n" "$$missing"; \
		echo "Run: make setup  (node and npm come from mise.toml)"; \
		exit 1; \
	fi
	@npm_ver=$$(npm --version); \
	npm_major=$$(echo $$npm_ver | cut -d. -f1); \
	npm_minor=$$(echo $$npm_ver | cut -d. -f2); \
	if [ "$$npm_major" -lt 11 ] || { [ "$$npm_major" -eq 11 ] && [ "$$npm_minor" -lt 10 ]; }; then \
		printf "\033[31mError: npm %s is too old. Need >= 11.10 for the supply-chain cooldown in web/.npmrc (min-release-age).\033[0m\n" "$$npm_ver"; \
		echo "Run: make setup  (mise.toml pins the npm version)"; \
		exit 1; \
	fi

check-tools: check-tools-api check-tools-web ## Verify all required tools are installed (go, node, npm >= 11.10, docker)
	@if ! command -v docker >/dev/null 2>&1; then \
		printf "\033[31mError: missing required tool: docker\033[0m\n"; \
		echo "Install Docker: https://docs.docker.com/get-docker/"; \
		exit 1; \
	fi

check-air: check-toolchain ## Verify air is installed (required for Go hot-reload in dev)
	@if ! command -v air >/dev/null 2>&1; then \
		printf "\033[31mError: air is not installed (required for hot-reload).\033[0m\n"; \
		echo "Run: make setup  (it comes from mise.toml)"; \
		exit 1; \
	fi

check-env: check-tools ## Verify .env exists and has all required variables
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

dev: check-env check-air dev-services ## Start all services for development (API + Web + Worker)
	@trap 'kill 0' EXIT; \
	(set -a && . ./.env && set +a && export DISCORD_REDIRECT_URI=http://localhost:5173/auth/discord/callback && cd api && air) & \
	(cd web && npm run dev -- --host) & \
	(set -a && . ./.env && set +a && cd api && air -c .air-worker.toml) & \
	wait

dev-stop: ## Stop dev services (including semantic-search profile)
	@echo ""
	@printf "$(CYAN)## Stopping dev services...$(RESET)\n"
	docker compose --profile semantic-search down
	@printf "$(GREEN)## Dev services stopped$(RESET)\n"

dev-services: check-env ## Start PostgreSQL, MinIO, NATS, Mailpit and Ollama
	@echo ""
	@printf "$(CYAN)## Starting dev services (PostgreSQL + MinIO + NATS + Mailpit + Ollama)...$(RESET)\n"
	docker compose --profile semantic-search up postgres minio minio-init nats mailpit ollama ollama-init -d
	@printf "$(GREEN)## Dev services started$(RESET)\n"

dev-db: dev-services ## Alias for dev-services (legacy)

dev-api: check-env check-air dev-services ## Start API server with hot reload (air is installed by 'make setup')
	set -a && . ./.env && set +a && export DISCORD_REDIRECT_URI=http://localhost:5173/auth/discord/callback && cd api && air

dev-web: check-tools-web ## Start frontend dev server (Vite on :5173, proxies /api to :8080)
	cd web && npm run dev

dev-worker: check-env check-air dev-services ## Start worker with hot reload (air is installed by 'make setup')
	set -a && . ./.env && set +a && cd api && air -c .air-worker.toml

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

GHCR_REPO := ghcr.io/marcoshack/taskwondo
PUSH_IMAGES := api web worker

push: check-toolchain ## Push images to GHCR (usage: RELEASE_VERSION=0.2.0 make push)
	@echo ""
	@if [ -z "$(RELEASE_VERSION)" ]; then \
		printf "$(CYAN)## Pushing images as latest...$(RESET)\n"; \
		IMAGE_TAG=latest docker compose build api web worker; \
		IMAGE_TAG=latest docker compose push api web worker; \
	else \
		printf "$(CYAN)## Pushing images as $(RELEASE_VERSION) + latest...$(RESET)\n"; \
		IMAGE_TAG=latest docker compose build api web worker; \
		for img in $(PUSH_IMAGES); do \
			docker tag $(GHCR_REPO)/$$img:latest $(GHCR_REPO)/$$img:$(RELEASE_VERSION); \
			docker push $(GHCR_REPO)/$$img:$(RELEASE_VERSION); \
			docker push $(GHCR_REPO)/$$img:latest; \
		done; \
	fi
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

# --- Release ---

RELEASE_VERSION ?=

release: check-toolchain ## Build release tarballs (usage: make release or RELEASE_VERSION=1.0.0 make release)
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
	mkdir -p build/release/taskwondo-server-$(VERSION)/bin build/release/taskwondo-server-$(VERSION)/html
	@printf "$(CYAN)## Building API binary (Docker)...$(RESET)\n"
	docker build -f docker/Dockerfile.api --target builder --build-arg GOPROXY -t taskwondo-api-builder api
	docker create --name taskwondo-api-extract taskwondo-api-builder true
	docker cp taskwondo-api-extract:/bin/taskwondo build/release/taskwondo-server-$(VERSION)/bin/taskwondo-api
	docker rm taskwondo-api-extract
	@printf "$(CYAN)## Building Worker binary (Docker)...$(RESET)\n"
	docker build -f docker/Dockerfile.worker --target builder --build-arg GOPROXY -t taskwondo-worker-builder api
	docker create --name taskwondo-worker-extract taskwondo-worker-builder true
	docker cp taskwondo-worker-extract:/bin/taskwondo-worker build/release/taskwondo-server-$(VERSION)/bin/taskwondo-worker
	docker rm taskwondo-worker-extract
	@printf "$(CYAN)## Building Web bundle (Docker)...$(RESET)\n"
	docker build -f docker/Dockerfile.web --target builder -t taskwondo-web-builder .
	docker create --name taskwondo-web-extract taskwondo-web-builder true
	docker cp taskwondo-web-extract:/src/dist/. build/release/taskwondo-server-$(VERSION)/html/
	docker rm taskwondo-web-extract
	cp .env.template build/release/taskwondo-server-$(VERSION)/.env.template
	cp docker/nginx.conf build/release/taskwondo-server-$(VERSION)/nginx.conf
	cp MANUAL_INSTALL.md build/release/taskwondo-server-$(VERSION)/README.md
	@printf "$(CYAN)## Packaging tarball...$(RESET)\n"
	tar -czf build/release/taskwondo-server-$(VERSION).tar.gz -C build/release taskwondo-server-$(VERSION)
	@echo ""
	@echo "Release artifact:"
	@ls -lh build/release/taskwondo-server-$(VERSION).tar.gz
	@echo ""
	@echo "Contents:"
	@tar -tzf build/release/taskwondo-server-$(VERSION).tar.gz | head -20
	@printf "$(GREEN)## Release v$(VERSION) built successfully$(RESET)\n"

# --- Worker ---

build-worker: check-tools-api ## Build the worker binary
	@echo ""
	@printf "$(CYAN)## Building worker...$(RESET)\n"
	cd api && go build -o ../build/taskwondo-worker ./cmd/worker
	@printf "$(GREEN)## Worker built successfully$(RESET)\n"

# --- MCP Server ---

build-mcp: build-mcp-linux build-mcp-darwin build-mcp-windows build-mcpb ## Build all MCP artifacts (Linux + macOS + Windows + MCPB bundle)

build-mcp-linux: check-tools-api ## Build the MCP server binary for Linux/amd64
	@echo ""
	@printf "$(CYAN)## Building MCP server for Linux/amd64...$(RESET)\n"
	$(MAKE) -C mcp build-linux
	@printf "$(GREEN)## MCP server (Linux/amd64) built successfully$(RESET)\n"

build-mcp-darwin: check-tools-api ## Build the MCP server binary for macOS/arm64
	@echo ""
	@printf "$(CYAN)## Building MCP server for macOS/arm64...$(RESET)\n"
	$(MAKE) -C mcp build-darwin
	@printf "$(GREEN)## MCP server (macOS/arm64) built successfully$(RESET)\n"

build-mcp-windows: check-tools-api ## Build the MCP server binary for Windows/amd64
	@echo ""
	@printf "$(CYAN)## Building MCP server for Windows/amd64...$(RESET)\n"
	$(MAKE) -C mcp build-windows
	@printf "$(GREEN)## MCP server (Windows/amd64) built successfully$(RESET)\n"

# --- MCPB Bundle ---

build-mcpb: build-mcp-windows build-mcp-darwin ## Build the MCPB bundle for Claude Desktop, Windows + macOS (usage: RELEASE_VERSION=0.3.0 make build-mcpb)
	@echo ""
	@printf "$(CYAN)## Building MCPB bundle...$(RESET)\n"
	$(MAKE) -C mcpb build VERSION=$(RELEASE_VERSION)
	@printf "$(GREEN)## MCPB bundle built successfully$(RESET)\n"

# --- CI Linting ---

lint-ci: check-tools ## Lint GitHub Actions workflows and check for deprecated Node.js runtimes (non-blocking)
	@echo ""
	@printf "$(CYAN)## Linting GitHub Actions workflows...$(RESET)\n"
	@docker run --rm -v "$$(pwd)/.github:/repo/.github:ro" --entrypoint sh \
		rhysd/actionlint:latest -c "actionlint -color /repo/.github/workflows/*.yml" \
		|| printf "\033[33m## actionlint found issues (see above)\033[0m\n"
	@echo ""
	@printf "$(CYAN)## Checking action Node.js runtimes...$(RESET)\n"
	@docker run --rm -v "$$(pwd)/.github:/repo/.github:ro" alpine:3 sh -c ' \
		apk add --quiet --no-cache curl >/dev/null 2>&1; \
		failed=0; \
		for wf in /repo/.github/workflows/*.yml; do \
			for action in $$(grep "uses:" "$$wf" | sed "s/.*uses: *//" | sed "s/ .*//" | grep -v "\\./" | sort -u); do \
				repo=$$(echo "$$action" | cut -d@ -f1); \
				tag=$$(echo "$$action" | cut -d@ -f2); \
				runtime=$$(curl -sf "https://raw.githubusercontent.com/$$repo/$$tag/action.yml" \
					| grep "using:" | head -1 | sed "s/.*using: *//" | sed "s/[^a-z0-9]//g"); \
				case "$$runtime" in \
					node16|node20) \
						printf "   \033[33m⚠ %-45s %s (deprecated)\033[0m\n" "$$action" "$$runtime"; \
						failed=1 ;; \
					*) \
						printf "   \033[32m✓ %-45s %s\033[0m\n" "$$action" "$$runtime" ;; \
				esac; \
			done; \
		done; \
		if [ "$$failed" = "1" ]; then \
			echo ""; \
			printf "\033[33m## Some actions use deprecated Node.js runtimes\033[0m\n"; \
		fi \
	'
	@printf "$(GREEN)## CI lint check complete$(RESET)\n"

# --- Testing ---

LIGHT_BLUE := \033[94m

test: test-api test-web lint-ci ## Run all tests (API + frontend + CI lint)

test-api: check-tools-api ## Run Go API tests
	@echo ""
	@printf "$(CYAN)## Running Go tests...$(RESET)\n"
	cd api && go test ./... -v -race -cover 2>&1 | tee /tmp/taskwondo-test-output.txt
	@echo ""
	@printf "$(LIGHT_BLUE)## Coverage by package:$(RESET)\n"
	@grep -E '^ok\s' /tmp/taskwondo-test-output.txt | sed 's|github.com/marcoshack/taskwondo/||' | awk '{pkg=$$2; for(i=1;i<=NF;i++) if($$i ~ /^coverage:/) {pct=$$(i+1); gsub(/%/,"",pct); printf "$(LIGHT_BLUE)   %-40s %s%%$(RESET)\n", pkg, pct}}' | sort
	@total=$$(sed -n 's/.*coverage: \([0-9.]*\)%.*/\1/p' /tmp/taskwondo-test-output.txt | awk '{s+=$$1; n++} END {if(n>0) printf "%.1f", s/n; else print "0"}'); \
	printf "$(LIGHT_BLUE)   %-40s %s%%$(RESET)\n" "TOTAL (avg)" "$$total"
	@printf "$(GREEN)## Go tests passed$(RESET)\n"

test-web: check-tools-web ## Run frontend tests and build (install, Vitest, tsc + vite build)
	@echo ""
	@printf "$(CYAN)## Installing frontend dependencies...$(RESET)\n"
	cd web && npm ci
	@printf "$(CYAN)## Running frontend tests...$(RESET)\n"
	cd web && npm test
	@printf "$(CYAN)## Building frontend...$(RESET)\n"
	cd web && npm run build
	@printf "$(GREEN)## Frontend tests and build passed$(RESET)\n"

test-e2e: check-toolchain ## Run E2E tests in isolated Docker stack (no host deps)
	@echo ""
	@printf "$(CYAN)## Running E2E tests (Docker)...$(RESET)\n"
	bash scripts/e2e-docker.sh
	@printf "$(GREEN)## E2E tests passed$(RESET)\n"

test-e2e-dev: check-tools-web ## Run E2E tests against local dev server (localhost:5173)
	@echo ""
	@printf "$(CYAN)## Running E2E tests (dev)...$(RESET)\n"
	cd test/e2e && npx playwright test
	@printf "$(GREEN)## E2E tests passed$(RESET)\n"

test-e2e-report: ## Serve the last E2E HTML report at http://localhost:9323
	@echo "Serving report at http://localhost:9323"
	@echo "Press Ctrl+C to stop"
	docker run --rm -p 9323:80 -v "$$(pwd)/test/e2e/playwright-report:/usr/share/nginx/html:ro" nginx:alpine
