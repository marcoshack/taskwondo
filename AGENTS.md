# AGENTS.md — AI Agent Implementation Guide

## Project Overview

TrackForge is a self-hosted task and ticket management system built with Go (backend API), PostgreSQL (database), and React/TypeScript (frontend). It combines project management with ticketing, incident tracking, and a public-facing customer portal.

**Read the design docs before implementing anything:**
- [README.md](README.md) — Project overview and structure
- [docs/architecture.md](docs/architecture.md) — System design and deployment
- [docs/data-model.md](docs/data-model.md) — Database schema and entity relationships
- [docs/api-design.md](docs/api-design.md) — REST API specification
- [docs/access-control.md](docs/access-control.md) - How user permissions are structured
- [docs/workflows.md](docs/workflows.md) — Work item lifecycles and automation
- [docs/public-portal.md](docs/public-portal.md) — Public portal specification
- [docs/integrations.md](docs/integrations.md) — Webhooks, Discord, email

## Tech Stack

| Component | Technology | Version |
|-----------|-----------|---------|
| Language | Go | 1.25+ |
| Database | PostgreSQL | 16+ |
| HTTP Router | chi (go-chi/chi/v5) | v5 |
| SQL | sqlc | latest |
| Migrations | golang-migrate/migrate | v4 |
| Frontend | React + TypeScript + Vite | React 18, Vite 5 |
| Styling | Tailwind CSS | v3 |
| State | TanStack Query + Zustand | latest |
| Forms | React Hook Form + Zod | latest |
| Testing (Go) | stdlib testing + testify | latest |
| Testing (Frontend) | Vitest + Testing Library | latest |
| Containerization | Docker + Docker Compose | latest |

## Code Conventions

### Go

- **Package naming:** lowercase, single-word when possible (`handler`, `service`, `repository`, `model`)
- **Error handling:** Always wrap errors with context: `fmt.Errorf("creating work item: %w", err)`
- **Logging:** Use `zerolog` (github.com/rs/zerolog). Pass context everywhere; use `log.Ctx(ctx)` for contextual logging. Always include structured fields: `log.Ctx(ctx).Error().Err(err).Str("project_id", projectID).Msg("failed to create item")`
- **Context:** Pass `context.Context` as first parameter to all functions and methods (use `_ context.Context` if unused). This enables contextual logging via `log.Ctx(ctx)` and future tracing.
- **Interfaces:** Define interfaces in the consumer package, not the provider. The `service` package defines repository interfaces; the `repository` package implements them.
- **Configuration:** All config via environment variables, loaded once at startup into a `Config` struct
- **No global state:** Dependencies are injected via constructors. No `init()` functions except for the `main` package.
- **HTTP handlers:** Accept `http.ResponseWriter` and `*http.Request`. Use helper functions for JSON response writing and error responses.
- **IDs:** Use `google/uuid` package. Generate UUIDv7 where time-ordering matters (work items, events). Standard UUIDv4 elsewhere.

### SQL / Database

- **Migrations:** Sequential numbered SQL files: `000001_create_users.up.sql`, `000001_create_users.down.sql`
- **Naming:** Snake_case for tables and columns. Plural table names (`users`, `work_items`, `comments`).
- **sqlc:** Write SQL queries in `.sql` files under `api/internal/database/queries/`. Run `sqlc generate` to produce Go code.
- **Transactions:** Use a transaction helper that accepts a function: `repo.WithTx(ctx, func(tx *sql.Tx) error { ... })`
- **Soft deletes:** Filter `WHERE deleted_at IS NULL` in all list/get queries. Provide separate methods for including deleted items when needed.

### React / TypeScript

- **Components:** Functional components with hooks. No class components.
- **File naming:** PascalCase for components (`WorkItemCard.tsx`), camelCase for utilities (`formatDate.ts`)
- **API client:** Centralized in `web/src/api/`. Use TanStack Query hooks for all data fetching.
- **Types:** Define API response types in `web/src/types/`. Mirror the API response structure.
- **Styling:** Tailwind utility classes only. No CSS modules or styled-components. Extract repeated patterns into component variants.
- **Forms:** React Hook Form with Zod schemas for validation. Define schemas in the component file or a shared schemas file.
- **State management:** TanStack Query for server state (the primary state). Zustand only for pure client state (auth token, UI preferences, sidebar collapsed).
- **Error boundaries:** Wrap major sections in error boundaries with fallback UI.

## Implementation Order

Build the system in this order. Each phase produces a working, testable increment.

### Phase 1: Foundation

**Goal:** API server boots, connects to database, runs migrations, serves health checks.

1. Initialize Go module (`go mod init github.com/marcoshack/trackforge`) in `api/`
2. Create `api/cmd/server/main.go` — config loading, DB connection, HTTP server with graceful shutdown
3. Create `api/internal/config/config.go` — environment variable loading with defaults
4. Create `api/internal/database/database.go` — PostgreSQL connection pool setup
5. Create initial migration: `users` table
6. Create `api/internal/middleware/` — logging, recovery, CORS, request ID
7. Wire up chi router with `/healthz` and `/readyz` endpoints
8. Create `Dockerfile.api` and `docker-compose.yml`
9. Verify: `docker compose up` starts API + Postgres, health checks return 200

**Key files:**
```
api/cmd/server/main.go
api/internal/config/config.go
api/internal/database/database.go
api/internal/database/migrations/000001_create_users.up.sql
api/internal/database/migrations/000001_create_users.down.sql
api/internal/middleware/logging.go
api/internal/middleware/recovery.go
api/internal/middleware/cors.go
api/internal/middleware/requestid.go
api/internal/handler/health.go
docker/Dockerfile.api
docker-compose.yml
.env.template
```

### Phase 2: Authentication & Users

**Goal:** Users can register, login, and receive JWT tokens. API key authentication works.

1. Create migration: `api_keys` table
2. Implement `api/internal/model/user.go` — User struct, role constants
3. Implement `api/internal/repository/user.go` — CRUD operations via sqlc
4. Implement `api/internal/service/auth.go` — password hashing (bcrypt), JWT generation/validation, API key validation
5. Implement `api/internal/handler/auth.go` — login, refresh, me, logout endpoints
6. Implement `api/internal/middleware/auth.go` — JWT extraction and validation middleware
7. Create initial admin user via environment variable or CLI flag
8. Test: Login returns JWT, authenticated requests work, unauthenticated requests return 401

### Phase 3: Projects & Members

**Goal:** CRUD for projects and project membership.

1. Create migration: `projects`, `project_members` tables
2. Implement model, repository, service, handler for projects
3. Implement project member management
4. Add authorization middleware (check project membership and role)
5. Test: Create project, add members, verify role-based access

### Phase 4: Work Items (Core)

**Goal:** Full CRUD for work items with filtering, search, and pagination.

1. Create migration: `work_items`, `work_item_events` tables
2. Implement model with all work item types and fields
3. Implement repository with cursor-based pagination and filtering
4. Implement service with business logic (item number generation, event recording)
5. Implement handler with all query parameter support
6. Add full-text search using the generated `tsvector` column
7. Test: Create items of different types, filter, search, paginate

### Phase 5: Comments, Relations, Activity

**Goal:** Comments with visibility, work item relations, activity timeline.

1. Create migration: `comments`, `work_item_relations` tables
2. Implement comments with internal/public visibility
3. Implement relations (blocks, relates_to, caused_by, etc.)
4. Implement activity timeline (merge events + comments, ordered by time)
5. Test: Add internal and public comments, create relations, verify timeline

### Phase 6: Workflows

**Goal:** Configurable workflows with status transitions.

1. Create migration: `workflows`, `workflow_statuses`, `workflow_transitions` tables
2. Seed default workflows (task workflow, ticket workflow)
3. Implement workflow validation on status changes
4. Implement available transitions endpoint (for UI to show valid actions)
5. Test: Status transitions follow workflow rules, invalid transitions rejected

### Phase 7: Queues & Milestones

**Goal:** Queues for inbound work, milestones for tracking progress.

1. Create migration: `queues`, `milestones` tables, add `queue_id` and `milestone_id` to work items
2. Implement queue CRUD with default assignment and workflow override
3. Implement milestone CRUD with progress tracking (count of open/closed items)
4. Test: Create queues, assign items to queues, milestone progress updates

### Phase 8: React Frontend (Shell)

**Goal:** React app with auth, project list, and basic navigation.

1. Initialize Vite + React + TypeScript project in `web/`
2. Set up Tailwind CSS
3. Create base components (Button, Input, Select, Modal, Badge, Avatar, DataTable)
4. Implement API client with TanStack Query
5. Implement auth flow (login page, token storage, protected routes)
6. Implement project list and project detail layout
7. Create `Dockerfile.web` with Nginx config
8. Test: Login, see projects, navigate between projects

### Phase 9: Frontend — Work Item Views

**Goal:** Full work item management UI.

1. Work item list view with filters, search, and pagination
2. Work item detail view with description, comments, activity timeline, relations
3. Work item creation/edit forms
4. Board view (kanban columns by status category)
5. Bulk actions (multi-select, bulk status change, bulk assign)
6. Test: Full CRUD flow through the UI, board drag-and-drop

### Phase 10: Portal

**Goal:** Public-facing portal for ticket submission and tracking.

1. Create migration: `portal_contacts`, `portal_sessions` tables
2. Implement portal auth (email verification code flow)
3. Implement portal ticket submission endpoint
4. Implement portal ticket list/detail (scoped to own tickets, public visibility only)
5. Build portal React pages (separate route tree under `/portal/`)
6. Test: Submit ticket as anonymous user, verify email, track ticket, see public comments

### Phase 11: Automation Engine

**Goal:** Automation rules with triggers and actions.

1. Create migration: `automation_rules` table
2. Implement rule evaluation engine (match trigger conditions against events)
3. Implement action executors (set_field, add_comment, send_webhook, etc.)
4. Implement background runner for async action execution
5. Wire automation into work item service (evaluate rules after creates/updates)
6. Build automation rule management UI
7. Test: Create rules, trigger them via work item actions, verify actions execute

### Phase 12: Inbound Webhooks

**Goal:** Prometheus and Grafana alerts automatically create tickets.

1. Create migration: `webhook_endpoints`, `webhook_deliveries` tables
2. Implement webhook receiver handler with auth validation
3. Implement Prometheus Alertmanager payload parser
4. Implement Grafana payload parser
5. Implement generic payload parser with JSONPath field mapping
6. Implement deduplication logic (fingerprint matching)
7. Wire into automation engine
8. Build webhook management UI
9. Test: POST Alertmanager payload, verify ticket created with correct fields

### Phase 13: Outbound Notifications

**Goal:** Discord and email notifications for events.

1. Implement outbound webhook delivery with retry logic
2. Implement SMTP email sender
3. Implement notification preference management
4. Build notification settings UI
5. Test: Trigger automation rule with webhook action, verify delivery and retry

### Phase 14: Observability

**Goal:** OpenTelemetry traces and Prometheus metrics.

1. Add OpenTelemetry SDK initialization
2. Instrument HTTP middleware (traces + metrics)
3. Instrument database layer (query spans)
4. Instrument automation engine (rule execution spans)
5. Add Prometheus `/metrics` endpoint
6. Add custom business metrics (work items created, open count, etc.)
7. Test: Verify traces appear in collector, metrics scrapeable

### Phase 15: Polish & Hardening

1. SLA tracking implementation (timers, breach detection)
2. Knowledge base view in portal
3. API key management UI
4. Rate limiting implementation
5. Comprehensive error handling audit
6. Performance testing and query optimization
7. Documentation updates

## Testing Strategy

### Go Tests

- **Unit tests** for service layer business logic (mock repository interfaces)
- **Integration tests** for repository layer (use a test PostgreSQL database via testcontainers or docker)
- **Handler tests** for HTTP request/response validation (use `httptest`)
- Test files live alongside the code: `api/internal/service/workitem.go` → `api/internal/service/workitem_test.go`
- Run: `cd api && go test ./...`

### Frontend Tests

- **Component tests** with Vitest + Testing Library for interactive components
- **Hook tests** for custom hooks with TanStack Query
- Run: `cd web && npm test`

### Integration / E2E

- Full stack tests using Docker Compose + API calls
- Portal submission flow end-to-end
- Webhook receipt and ticket creation end-to-end

## Environment Variables

```env
# Database
DATABASE_URL=postgres://trackforge:password@postgres:5432/trackforge?sslmode=disable

# API Server
API_PORT=8080
API_HOST=0.0.0.0
JWT_SECRET=your-jwt-secret-min-32-chars
JWT_EXPIRY=24h

# Initial Admin
ADMIN_EMAIL=admin@example.com
ADMIN_PASSWORD=changeme

# SMTP (optional)
SMTP_HOST=
SMTP_PORT=587
SMTP_USERNAME=
SMTP_PASSWORD=
SMTP_FROM=TrackForge <noreply@example.com>

# OpenTelemetry (optional)
OTEL_ENABLED=false
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317
OTEL_SERVICE_NAME=trackforge

# Portal
PORTAL_ENABLED=true
PORTAL_SESSION_EXPIRY=720h

# General
LOG_LEVEL=info
LOG_FORMAT=json
BASE_URL=http://localhost:3000
```

## Common Patterns

### Adding a New Entity

1. Write the migration SQL (up and down)
2. Define the model struct in `api/internal/model/`
3. Write SQL queries in `api/internal/database/queries/`
4. Run `sqlc generate`
5. Create the repository in `api/internal/repository/` implementing the interface defined in the service package
6. Create the service in `api/internal/service/` with business logic
7. Create the handler in `api/internal/handler/` with HTTP endpoints
8. Register routes in the router setup
9. Add frontend API client hook
10. Build the UI components

### Adding a New API Endpoint

1. Define the route in the router (with appropriate middleware group)
2. Create handler method with request parsing, validation, service call, response
3. Add the corresponding service method
4. Add repository method if database access needed
5. Write handler test and service test
6. Add frontend API hook and UI

### Response Helpers

Use consistent response helpers:

```go
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(map[string]interface{}{"data": data})
}

func writeError(w http.ResponseWriter, status int, code string, message string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(map[string]interface{}{
        "error": map[string]interface{}{
            "code":    code,
            "message": message,
        },
    })
}
```

## Important Notes for AI Agents

1. **Always run `sqlc generate` after modifying SQL queries** — The generated Go code must match the SQL.

2. **Migrations are append-only** — Never modify an existing migration file. Create a new migration for schema changes.

3. **Test database isolation** — Each test should use a clean database state. Use transactions that roll back, or truncate tables between tests.

4. **The project key is the URL identifier for projects**, not the UUID. API routes use `/projects/:projectKey/items/:itemNumber`, not UUIDs in URLs.

5. **Work item numbers are per-project sequential integers.** Use `UPDATE projects SET item_counter = item_counter + 1 WHERE id = $1 RETURNING item_counter` in the same transaction as the insert.

6. **Comments and events have visibility.** Always filter by visibility based on the auth context (internal user vs portal contact).

7. **Don't import from handler into service or repository.** Dependencies flow one way: handler → service → repository.

8. **Use the `chi.URLParam()` function** to extract URL parameters from chi routes, not `mux.Vars()` or manual parsing.

9. **All times in UTC** in the database. Convert to user timezone only in the frontend.

10. **Cursor-based pagination** uses the last item's ID (or composite key) as the cursor, not page numbers. This is more efficient and handles concurrent inserts correctly.
