# AGENTS.md

This file provides guidance to AI coding agents when working with code in this repository.

## Project

Taskwondo — a self-hosted task and ticket management system. Monorepo with a Go REST API (`api/`), React frontend (`web/`), MCP server (`mcp/`), and Playwright E2E tests (`test/e2e/`).

## Commands

Run `make help` for the full list of Make targets (dev, test, build, migrate, release, etc.).
Requires `.env` — copy from `.env.template`.

Commands not covered by `make help`:

```bash
# Go tests — single package / single test
cd api && go test ./internal/handler/... -v -race
cd api && go test ./internal/service/... -v -run TestName

# Frontend — lint / typecheck only (no Make target)
cd web && npm run lint
cd web && npm run typecheck

# Migrations also run automatically on API startup.
# Use `./taskwondo --migrate-only` to run migrations and exit (useful for init containers / CI).
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
- **Errors**: Wrap with context: `fmt.Errorf("creating work item: %w", err)`. For user-facing validation errors that need localization, use `model.NewKeyedError(sentinel, "error_key", "english message", params)` — the handler layer automatically extracts the key via `writeErrorFromService`.
- **Error keys**: Stable, snake_case identifiers (e.g. `namespace_slug_reserved`, `project_key_in_use`). Never rename once released. Add the corresponding `errors.<key>` i18n entry to all language files.
- **No global state.** Dependency injection via constructors. No `init()` except in `main`.
- **All times UTC** in the database. Convert to user timezone only in the frontend.
- **Commit messages**: Prefix with `[DISPLAY_ID]` (e.g. `[TF-141]`, `[PROJ-23]`) when a work item display ID is provided. The display ID format is `<PROJECT_KEY>-<NUMBER>`. No Co-Authored-By.

### React/TypeScript
- **i18n**: All UI strings in `web/src/i18n/en.json`. Use `const { t } = useTranslation()` in every component. `<Trans>` for JSX with embedded HTML. Module-level arrays with display strings must be inside component body. Interpolation: `{{var}}`. Pluralization: `_one`/`_other` suffixes. Any key added to `en.json` must also be added to all other language files.
- **Adding a new locale**: four places must stay in sync — (1) create `web/src/i18n/<code>.json` with a full translation of `en.json`, (2) import it in `web/src/i18n/index.ts` and add it to the `resources` map, (3) add the code to the `Language` type union in `web/src/contexts/LanguageContext.tsx`, (4) add an entry to the `SUPPORTED_LANGUAGES` array in the same file. The i18n Vitest suite will fail if keys drift between locales.
- **API errors**: Use `getLocalizedError(err, t, 'fallback.key')` from `@/utils/apiError` to display API errors. Never extract `error.message` manually. The helper resolves `error_key` → i18n translation with params, falling back to the raw message then the fallback key.
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

**Coverage target:** 80%+ per package. Skip only when the remaining paths would require disproportionate complexity (mocking transactional boundaries, platform-specific code, etc.) — document the reason in the test file if so.

**Test at the same entry point the real client hits.** A service-level test is not a substitute for a handler-level test: handlers contain their own input validation, auth checks, and response shaping that service tests will silently skip. If a bug can be triggered by an HTTP request, there must be a test that sends the same HTTP request. The same rule applies in the frontend: if a bug is visible in the UI, an E2E test should exercise the UI flow — not just the hook or API wrapper underneath.

### Go (`api/`)
In-package mocks (mock structs implementing repository interfaces) and `httptest` for handler tests. Chi router is wired up in tests when URL params are needed. Tests live alongside source files.

### Frontend (`web/`)
Vitest for unit tests. Tests use `*.test.ts` naming and live alongside source files. Currently covers i18n validation (missing keys, extra keys, placeholder consistency, untranslated values). No component or hook tests — functional coverage comes from E2E.

### E2E (`test/e2e/`)
Playwright with `*.spec.ts` naming. Tests organized by domain under `test/e2e/tests/` (auth, admin, workitems, projects, milestones, navigation, preferences).

Key infrastructure:
- **Fixtures** (`test/e2e/lib/fixtures.ts`): extends Playwright's base test with `testUser` and `testProject` fixtures that auto-create isolated users and projects per test
- **API helpers** (`test/e2e/lib/api.ts`): 60+ typed functions for setting up test data via API (work items, comments, relations, milestones, etc.)
- **Multi-project setup**: auth.setup.ts → admin tests → chromium.setup.ts → main suite → cleanup.teardown.ts
- **Fully containerized**: `make test-e2e` runs the entire stack in Docker (Postgres, MinIO, Mailpit, API, Web, Playwright)
