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
- **Tooltips**: Never use the native HTML `title` attribute for tooltips. Use the stylized Tailwind pattern: wrap the trigger with `relative group/<name>` and render an absolutely-positioned `<span>` child with `pointer-events-none absolute ... px-2 py-1 text-xs text-white bg-gray-900 dark:bg-gray-700 rounded whitespace-nowrap opacity-0 group-hover/<name>:opacity-100 transition-opacity`. See `WorkItemDetailPage.tsx` (pencil edit button) and `AppSidebar.tsx` for canonical examples.

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

## MCP agent operations

Guidance for Cursor (and other) agents using the Taskwondo MCP tools (`mcp_taskwondo_*`).

### Where notes live (how documentation is organized)

| Layer | Location | What goes here | Version control |
|-------|----------|----------------|-----------------|
| **Repo agent guide** | `taskwondo/AGENTS.md` (this file) | Cross-cutting rules: MCP workflow transitions, codebase conventions, ports | Git — source of truth for agents editing Taskwondo |
| **Other repo guides** | e.g. `~/work/watchtower/AGENTS.md` | Deploy/workflow for that repo | Git |
| **Cursor rules** | `~/.cursor/rules/*.mdc` | Machine-wide layout (`DIRECTORY.md`), always-on constraints | Git or local dotfiles |
| **Project description** | Taskwondo → Project settings → Description | One-paragraph scope + pointers to deeper docs (e.g. WEAVE → this section) | Taskwondo DB |
| **Epic / task body** | Work item description in Taskwondo | Specs, acceptance criteria, architecture decisions for that work | Taskwondo DB |
| **Comments** | Work item comments | Session notes, spike results, “decision recorded here” | Taskwondo DB |
| **Meta platform work** | TASK project tickets | Changes to Taskwondo itself (MCP, workflows, agent runtime) | Taskwondo DB + usually a git change |

**Rule of thumb:** Durable agent behaviour → **git** (`AGENTS.md`). Task-specific intent → **work item description**. Pointers from a project → **project description** (keep short).

Tracked in **TASK-58**.

### Workflow status changes via MCP

`update_work_item` with `status` enforces the project’s **workflow graph**. Invalid jumps return **HTTP 409**:

```text
transition from "backlog" to "done" is not allowed in this workflow: invalid transition
```

That is expected — not an MCP bug or missing permission. Other fields (description, labels, milestone) still save even when status fails.

**Before changing status:**

1. Call `list_statuses` for the project to see valid status names (case-sensitive: `in_progress`, not `In Progress`).
2. If closing an item, plan a **path** through allowed transitions — do not assume `done` / `resolved` / `closed` are reachable from the current status in one step.

### Default “Task Workflow” (WEAVE, TASK, most dev projects)

Used for types: task, bug, epic, story, feedback. Seeded in `api/internal/service/workflow.go`; extra transitions in migrations `000038`, `000064`.

**Statuses (in order):** `backlog` → `open` → `in_progress` → `in_review` → `done` | `cancelled`

**Transitions that reach `done`:**

| From | To | Transition name |
|------|-----|-----------------|
| `open` | `done` | Complete |
| `in_progress` | `done` | Complete |
| `in_review` | `done` | Approve |

**There is no direct `backlog` → `done`.** From `backlog`, use one of:

```text
backlog → in_progress → done          # fastest (2 steps)
backlog → open → done                 # Prioritize, then Complete
backlog → open → in_progress → in_review → done   # full review path
```

**Example — close a research task from `backlog` (MCP):**

```text
update_work_item(display_id, status: "in_progress")   # or "open" first
update_work_item(display_id, status: "done")            # if currently open or in_progress
# OR formal review path:
update_work_item(..., status: "in_review")
update_work_item(..., status: "done")
```

**Statuses that are not interchangeable:** `done`, `resolved`, and `closed` are different names; only use values returned by `list_statuses`. On Task Workflow, terminal completion is usually **`done`**.

**Ticket Workflow** (support/ticket types) may differ — always `list_statuses` for that project.

### MCP tool reminders

- `list_statuses` — valid status names for filters and transitions.
- `update_work_item` — only provided fields change; invalid `status` rejects the whole update (409). Supports `queue_id` (`'none'` to clear).
- Work item descriptions support markdown; use for decisions when git docs are overkill.
- Display IDs: `WEAVE-1`, `TASK-58` — use `display_id` param, not UUID.

### Agent queues and labels (TASK-16 / TASK-14)

**Routing model:** `assignee` is the source of truth for agent workers (`list_work_items(assignee="me")`). **Queues** and **`agent:*` labels** are triage/intake signals — they tell humans and the triage worker where work belongs before assignment.

#### Queues (per project)

Each project has three agent queues (see **TASK-16** for UUIDs):

| Queue | `queue_type` | Default priority | Label |
|---|---|---|---|
| Code Review Agent | `general` | `medium` | `agent:code-review` |
| Bug Triage Agent | `alerts` | `high` | `agent:triage` |
| Customer Support Agent | `support` | `medium` | `agent:support` |

Set `queue_id` on create (`create_work_item`) or update (`update_work_item` with `queue_id`).

#### Label convention

| Label | Meaning | Paired queue |
|---|---|---|
| `agent:code-review` | Route to Code Review Agent | Code Review Agent |
| `agent:triage` | Route to Bug Triage Agent | Bug Triage Agent |
| `agent:support` | Route to Customer Support Agent | Customer Support Agent |
| `agent:research` | Research/summarization work (no dedicated queue yet) | — |
| `agent:summarizer` | Summarization-only output | — |

**Approval overrides** (TASK-19 / TASK-17): `agent:auto-done`, `agent:must-review`, `agent:draft-only`.

#### Status handoff protocol (TASK-17)

Canonical lifecycle for agent-driven work items:

| Status | Meaning |
|---|---|
| `backlog` / `open` / `new` / `triaged` | Waiting — workers poll `assignees=me` in these statuses |
| `in_progress` | Agent claimed and is working |
| `in_review` | Agent finished; human review required (default terminal) |
| `done` | Completed and approved |
| `cancelled` | Rejected or no longer applicable |

**Claim:** worker sets `in_progress` after pickup. **Terminal:** worker sets status after execute per role + labels below.

| Role | Default terminal | Notes |
|---|---|---|
| `triage` | `in_review` | Comment-only + `agent:auto-done` → `done` |
| `implementer` / `codereview` | `in_review` | Always; human promotes to `done` |
| `support` | `in_review` | `agent:auto-done` → `done` |

**Label overrides** (apply before claim):

| Label | Effect |
|---|---|
| `agent:auto-done` | Triage/support may set `done` when work is non-mutating (triage) or always (support) |
| `agent:must-review` | Force `in_review` even with `agent:auto-done` |
| `agent:draft-only` | Implementer produces branch/diff only; terminal `in_review` |

**Internal comment on `in_review`:** summary, open questions, branch/PR link if applicable. See TASK-15 template.

Human attention: items in **`in_review`** assigned to an agent or recently updated by one.

#### Comment conventions (TASK-15)

**Public agent output:**

```markdown
**🤖 Agent: [Agent Name]**
**Result:** [Summary]
**Confidence:** High / Medium / Low
**Next Step:** [Suggested action]
```

**Internal / inter-agent (`internal: true`):**

```markdown
**🔁 Handoff from [Agent A] → [Agent B]**
**Context:** [Details for receiving agent]
**Input artifacts:** [Links, IDs, data]
```

Workers use `watchtower_agents/comments.py` helpers. Triage posts **internal** comments only.

#### Intake rules (backfill / new items)

| Work item type | Queue | Label |
|---|---|---|
| `bug` | Bug Triage Agent | `agent:triage` |
| `ticket`, `feedback` | Customer Support Agent | `agent:support` |
| `task` / `epic` / `story` with code-review keywords in title | Code Review Agent | `agent:code-review` |

When updating labels via MCP, `labels` **replaces** the full set — merge existing labels in your tool call.

#### Triage worker filtering

- **By queue:** API `GET .../items?queue_id=<uuid>` (queue-scoped board in UI).
- **By label:** API `GET .../items?label=agent:triage` (MCP `list_work_items` does not expose `label` yet — use API or assignee after triage).
- **By assignee:** `list_work_items(assignee="me")` — what running agent workers poll.

Tracked in **TASK-16** (queues), **TASK-14** (labels).

#### Agent identities (TASK-13)

Each queue-backed agent has a dedicated Taskwondo **user account** (one MCP process = one identity = one `twk_` key per TASK-18):

| Agent | Email | User ID |
|---|---|---|
| Bug Triage Agent | `agent-triage@agents.watchtower.lan` | `3f2000fe-64de-4eff-870c-71e45018e534` |
| Code Review Agent | `agent-codereview@agents.watchtower.lan` | `76a372e3-3c3b-495d-82e9-3e31f7fb021f` |
| Customer Support Agent | `agent-support@agents.watchtower.lan` | `f055e2dd-0155-4299-a3b2-027eb29fc0e5` |

All three are `member` on TASK, WATCH, WEAVE, DAUNT, SYMON, TEST. Assign work with `update_work_item(assignee=<user_id>)` after triage. Full table in **TASK-13**.

#### API keys (TASK-18)

Each agent user has a dedicated `twk_` key (permissions `read`+`write` at current API granularity). Keys live on the beast at `/data/watchtower/agents/` — per-agent env files + encrypted registry. **Never commit or log raw keys.**

| Agent slug | Key prefix | Env file |
|---|---|---|
| `agent-triage` | `twk_fd3d` | `/data/watchtower/agents/env/agent-triage.env` |
| `agent-codereview` | `twk_0dfc` | `/data/watchtower/agents/env/agent-codereview.env` |
| `agent-support` | `twk_6ba6` | `/data/watchtower/agents/env/agent-support.env` |

Provision/rotate: `~/work/watchtower/watchtower/agents/provision-api-keys.py` (see README there). Workers inject `TASKWONDO_API_KEY` via env only.
