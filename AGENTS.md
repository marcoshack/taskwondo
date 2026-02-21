# AGENTS.md — AI Agent Implementation Guide

## Project Overview

Taskwondo is a self-hosted task and ticket management system built with Go (backend API), PostgreSQL (database), and React/TypeScript (frontend).

**Design docs** (consult as needed, not all at once):
- [docs/data-model.md](docs/data-model.md) — Database schema and entity relationships
- [docs/api-design.md](docs/api-design.md) — REST API specification
- [docs/access-control.md](docs/access-control.md) — User permissions
- [docs/workflows.md](docs/workflows.md) — Work item lifecycles and automation
- [docs/public-portal.md](docs/public-portal.md) — Public portal specification
- [docs/integrations.md](docs/integrations.md) — Webhooks, Discord, email
- [docs/future-work.md](docs/implementation.md) — Future work

**Environment:** See [.env.template](.env.template) for all configuration variables.

## Code Conventions

### Go

- **Logging:** `zerolog` (NOT slog). Use `log.Ctx(ctx)` for contextual logging with structured fields.
- **Context:** `context.Context` as first parameter everywhere (`_ context.Context` if unused).
- **Interfaces:** Define in the consumer package, not the provider. `service` defines repo interfaces; `repository` implements them.
- **IDs:** `google/uuid`. UUIDv7 for time-ordered entities (work items, events), UUIDv4 elsewhere.
- **Errors:** Wrap with context: `fmt.Errorf("creating work item: %w", err)`
- **No global state.** Dependency injection via constructors. No `init()` except in `main`.

### SQL / Database

- **Migrations:** Append-only sequential numbered files: `000001_create_users.up.sql` / `.down.sql`
- **Soft deletes:** Always filter `WHERE deleted_at IS NULL` in list/get queries.

### React / TypeScript

- **i18n:** `react-i18next`. All UI strings in `web/src/i18n/en.json`. Use `const { t } = useTranslation()`. Any key added to `en.json` must also be added to all other language files.
- **API client:** Centralized in `web/src/api/`. TanStack Query hooks for all data fetching.
- **State:** TanStack Query for server state. Zustand only for client state (auth, UI prefs).
- **Confirmations:** Use `<Modal>` for destructive actions. Never `window.confirm()`.
- **Success feedback:** Never use banner messages that shift layout. Instead, show a temporary green checkmark (`<Check>` from lucide-react) inline on the affected card/row, to the left of the action icons. Use a `savedId` state + `setTimeout` (~2s) to auto-clear.

## Important Rules

1. **Migrations are append-only** — never modify an existing migration file.
2. **Dependency direction:** handler → service → repository. Never import backwards.
3. **Use `chi.URLParam()`** for URL parameters, not `mux.Vars()` or manual parsing.
4. **Project key is the URL identifier**, not UUID. Routes: `/projects/:projectKey/items/:itemNumber`.
5. **Work item numbers** are per-project sequential integers, incremented in the same transaction as insert.
6. **Comments and events have visibility** — always filter by auth context (internal user vs portal contact).
7. **All times UTC** in the database. Convert to user timezone only in the frontend.
8. **Cursor-based pagination** — use last item's ID as cursor, not page numbers.
