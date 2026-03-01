# AGENTS.md

This file provides guidance to AI coding agents when working with code in this repository.

## Project

Taskwondo — a self-hosted task and ticket management system. Monorepo with a Go REST API (`api/`), React frontend (`web/`), MCP server (`mcp/`), and Playwright E2E tests (`e2e/`).

## Commands

```bash
# Development (requires .env — copy from .env.template)
make dev              # Start everything: Postgres + MinIO + API (air hot-reload) + Web (Vite)
make dev-services     # Start only Postgres and MinIO (Docker)
make dev-api          # API with hot reload (requires: go install github.com/air-verse/air@latest)
make dev-web          # Vite dev server on :5173

# Go tests
cd api && go test ./... -v -race                                # All tests
cd api && go test ./internal/handler/... -v -race               # One package
cd api && go test ./internal/service/... -v -run TestName       # Single test

# Frontend
cd web && npm run build       # tsc + vite build
cd web && npm run lint        # ESLint
cd web && npm run typecheck   # tsc only

# E2E tests
make test-e2e                 # Fully containerized — builds & runs everything in Docker
make test-e2e-dev             # Against local dev server (localhost:5173, needs running dev stack)
make test-e2e-report          # Serve HTML report at http://localhost:9323

# Database
make migrate                              # Run migrations
make migrate-new name=create_foo          # Create new migration pair

# Docker
make up / make down / make logs
make build                    # Run tests then build images
```

## Architecture

### Go API (`api/`)

Entry point: `api/cmd/server/main.go`. Internal packages follow `handler → service → repository` dependency direction (never reversed). Interfaces are defined by the consumer.

```
api/internal/
  config/       — Env-based configuration
  database/     — DB connection + migration runner
    migrations/ — Numbered SQL files (000001_*.up.sql / *.down.sql), append-only
  handler/      — HTTP handlers (chi router), DTOs, request/response parsing
  middleware/   — Auth (JWT + API key), CORS, logging, rate limit, etc.
  model/        — Domain structs + error sentinels (ErrNotFound, ErrForbidden, ErrConflict, ErrValidation, ErrInvalidTransition)
  repository/   — SQL queries implementing service interfaces
  service/      — Business logic, RBAC authorization
  storage/      — Storage interface + MinIO/S3 implementation (attachments)
```

### React Frontend (`web/src/`)

```
api/          — Axios client functions (one file per domain)
components/ui/— Reusable primitives (Button, Input, Modal, Badge, DataTable, etc.)
components/workitems/ — Domain components (BoardView, CommentList, WorkItemForm, etc.)
contexts/     — Auth, Theme, Language, Notification contexts
hooks/        — TanStack Query hooks (useWorkItems, useProjects, useWorkflows, etc.)
i18n/         — en.json (all UI strings), init config
pages/        — Page components
```

Path alias: `@/` → `src/`. Vite proxies `/api` to `:8080` in dev.

### Key Patterns

- **Routing**: chi router. URL identifiers are project keys (not UUIDs): `/projects/:projectKey/items/:itemNumber`
- **Work item numbers**: Per-project sequential integers, incremented atomically during insert
- **IDs**: UUIDv7 for time-ordered entities (work items, events), UUIDv4 elsewhere
- **Auth**: JWT + API key (`twk_<hex>`) middleware. Passwords bcrypt-hashed, API keys SHA-256 hashed.
- **Pagination**: Cursor-based (last item ID), not page numbers
- **Soft deletes**: All queries filter `WHERE deleted_at IS NULL`
- **Workflow statuses**: Categories (todo, in_progress, done, cancelled) drive resolved_at and board column logic

## Conventions

### Go
- **Logging**: zerolog only. Use `log.Ctx(ctx)` for contextual logging.
- **Context**: `context.Context` as first param everywhere (`_ context.Context` if unused)
- **Interfaces**: Define in the consumer package, not the provider. `service` defines repo interfaces; `repository` implements them.
- **Errors**: Wrap with context: `fmt.Errorf("creating work item: %w", err)`
- **No global state.** Dependency injection via constructors. No `init()` except in `main`.
- **All times UTC** in the database. Convert to user timezone only in the frontend.
- **Commit messages**: Prefix with `[TF-nnn]` when a task display_id is provided. No Co-Authored-By.

### React/TypeScript
- **i18n**: All UI strings in `web/src/i18n/en.json`. Use `const { t } = useTranslation()` in every component. `<Trans>` for JSX with embedded HTML. Module-level arrays with display strings must be inside component body. Interpolation: `{{var}}`. Pluralization: `_one`/`_other` suffixes. Any key added to `en.json` must also be added to all other language files.
- **Destructive actions**: Always `<Modal>` with cancel/confirm. Never `window.confirm()`.
- **Success feedback**: Inline green checkmark (`<Check>` from lucide-react), never layout-shifting toasts. Pattern: `savedId` state + `setTimeout(~2s)`.
- **Settings pages**: Danger Zone is always the last section.

### API Compatibility
Always ask before making breaking API changes. Deprecation pattern: keep old param working, log warning, reject requests using both old and new params (400).

## Services & Ports

| Service    | Dev Port | Prod Port |
|------------|----------|-----------|
| Web (Vite) | 5173     | 3000 (nginx) |
| API        | 8080 (local only) | internal (via nginx `/api` proxy) |
| PostgreSQL | 5432     | -         |
| MinIO      | 9000/9001| -         |

The API is not exposed directly in Docker — all API traffic goes through the nginx container's `/api` reverse proxy. Port 8080 is only used when running the API locally with `make dev-api`.

Health: `GET /healthz` (liveness), `GET /readyz` (readiness + DB ping)

## Test Patterns

### Go (`api/`)
In-package mocks (mock structs implementing repository interfaces) and `httptest` for handler tests. Chi router is wired up in tests when URL params are needed. Tests live alongside source files.

### Frontend (`web/`)
Vitest for unit tests. Tests use `*.test.ts` naming and live alongside source files. Currently covers i18n validation (missing keys, extra keys, placeholder consistency, untranslated values). No component or hook tests — functional coverage comes from E2E.

### E2E (`e2e/`)
Playwright with `*.spec.ts` naming. Tests organized by domain under `e2e/tests/` (auth, admin, workitems, projects, milestones, navigation, preferences).

Key infrastructure:
- **Fixtures** (`e2e/lib/fixtures.ts`): extends Playwright's base test with `testUser` and `testProject` fixtures that auto-create isolated users and projects per test
- **API helpers** (`e2e/lib/api.ts`): 60+ typed functions for setting up test data via API (work items, comments, relations, milestones, etc.)
- **Multi-project setup**: auth.setup.ts → admin tests → chromium.setup.ts → main suite → cleanup.teardown.ts
- **Fully containerized**: `make test-e2e` runs the entire stack in Docker (Postgres, MinIO, Mailpit, API, Web, Playwright)
