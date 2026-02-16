# Architecture

## Overview

TrackForge is a monolithic Go application backed by a single PostgreSQL database, with a React/TypeScript frontend served via Nginx. The system is deployed as 3 Docker Compose services.

The architecture prioritizes simplicity: one API server, one database, one frontend container. No message queues, no Redis, no separate worker processes. Background work (automation rules, webhook retries) is handled by goroutines within the API server using a simple internal task scheduler.

## Design Principles

1. **Single database, single schema** — All data lives in one PostgreSQL database. No microservice boundaries, no cross-service joins, no distributed transactions.
2. **Monolithic API** — One Go binary handles all HTTP endpoints (internal API, public portal API, webhook receivers). Different auth middleware scopes access.
3. **Convention over configuration** — Sensible defaults for workflows, statuses, and priorities. Customizable but functional out of the box.
4. **API-first** — The React frontend is a pure API client. Every action available in the UI is available via the API. This enables CLI tools, Discord bots, and automation.
5. **Observable from day one** — OpenTelemetry instrumentation baked into the API server, exporting traces and metrics to the operator's existing Prometheus/Grafana stack.

## Component Architecture

### API Server (Go)

```
cmd/server/main.go
    │
    ├── Config loading (.env, environment variables)
    ├── Database connection + migration runner
    ├── OpenTelemetry provider initialization
    ├── Service layer initialization
    ├── HTTP router setup
    │   ├── /api/v1/*          — Internal API (JWT auth)
    │   ├── /portal/api/v1/*   — Public portal API (portal token or anonymous)
    │   ├── /webhooks/*        — Inbound webhooks (HMAC or bearer token auth)
    │   └── /healthz, /readyz  — Health checks
    └── Graceful shutdown handler
```

**Layered architecture within the monolith:**

| Layer | Responsibility | Package |
|-------|---------------|---------|
| Handler | HTTP request/response, validation, serialization | `internal/handler/` |
| Service | Business logic, authorization, orchestration | `internal/service/` |
| Repository | Database queries, transactions | `internal/repository/` |
| Model | Domain types, interfaces, constants | `internal/model/` |

**Key design decisions:**

- **chi router** for HTTP routing — lightweight, stdlib-compatible, good middleware support
- **sqlc** for type-safe SQL queries — write SQL, generate Go code. No ORM overhead, full control over queries.
- **golang-migrate** for schema migrations — SQL-based, versioned, runs on startup
- **No framework** — stdlib `net/http` with chi for routing. No Gin, Echo, Fiber, etc.

### Database (PostgreSQL)

Single PostgreSQL 16 instance. Schema uses:

- **UUIDs** for all primary keys (generated server-side using UUIDv7 for time-ordering)
- **JSONB columns** for flexible metadata (custom fields, automation rule definitions)
- **Timestamps with timezone** everywhere
- **Soft deletes** via `deleted_at` column on key entities
- **Row-level indexing** on `project_id`, `status`, `assignee_id`, `type`, `created_at`

No partitioning or sharding needed at this scale. A single Postgres instance handles millions of work items comfortably.

### Frontend (React + TypeScript)

Single-page application built with Vite, served by Nginx in production.

**Two entry points:**
- **Main app** (`/app/*`) — Full project management UI, requires authentication
- **Public portal** (`/portal/*`) — Ticket submission, status tracking, public views

Both share a component library but have separate routing trees and auth contexts.

**Key libraries:**
- React 18 with hooks
- React Router v6
- TanStack Query for server state management
- Tailwind CSS for styling
- Zustand for minimal client state (auth, UI preferences)
- React Hook Form + Zod for forms/validation

**No component library** (no Material UI, no Ant Design). Tailwind utility classes keep the bundle small and the design consistent. A small set of base components (Button, Input, Select, Modal, DataTable, Badge, Avatar) are built in-house.

### Nginx

Serves the React build and proxies API requests:

```nginx
/ → React SPA (index.html fallback)
/api/* → upstream api:8080
/portal/api/* → upstream api:8080
/webhooks/* → upstream api:8080
```

## Authentication & Authorization

### Auth Model

| Context | Method | Details |
|---------|--------|---------|
| Internal UI | JWT (httpOnly cookie) | Login with email/password, optional OIDC |
| API clients | API key (Bearer token) | Per-user or per-service keys with scoped permissions |
| Public portal | Portal session token | Anonymous or email-verified, scoped to own tickets |
| Webhooks | HMAC signature or shared secret | Per-integration validation |

### Authorization

Role-based access control (RBAC) at the project level:

| Role | Permissions |
|------|------------|
| Owner | Full control, manage members, delete project |
| Admin | Manage work items, workflows, automation rules |
| Member | Create/edit work items, comment |
| Viewer | Read-only access |

Additionally, work items have a **visibility** field:
- `internal` — Only project members can see
- `portal` — Visible to the submitter via public portal + internal members
- `public` — Visible to anyone on the public portal

## Background Processing

Instead of a separate worker service or message queue, TrackForge uses an internal task scheduler:

```go
// Lightweight background processor
type BackgroundRunner struct {
    db     *sql.DB
    tasks  chan Task
    logger *slog.Logger
}
```

**Background tasks include:**
- Processing automation rules after work item state changes
- Retrying failed webhook deliveries (outbound notifications)
- Cleaning up expired portal sessions
- Aggregating SLA metrics

Tasks are enqueued in-memory via a buffered channel. For crash resilience, pending automation triggers are also written to a `pending_tasks` table and picked up on restart.

This keeps the deployment at a single API process. If scale demands it later, the background runner can be extracted into a separate service that reads from the same `pending_tasks` table.

## Observability

### Traces

OpenTelemetry SDK integrated into the API server:
- Every HTTP request gets a trace
- Database queries are spans within the request trace
- Background task execution is traced
- Outbound webhook calls are traced

Export via OTLP to the operator's collector (e.g., Grafana Alloy, OpenTelemetry Collector).

### Metrics

Prometheus metrics exposed at `/metrics`:

| Metric | Type | Description |
|--------|------|-------------|
| `trackforge_http_requests_total` | Counter | Requests by method, path, status |
| `trackforge_http_request_duration_seconds` | Histogram | Request latency |
| `trackforge_workitems_created_total` | Counter | Work items created by type |
| `trackforge_workitems_open` | Gauge | Currently open work items by type/project |
| `trackforge_automation_rules_triggered_total` | Counter | Automation rule executions |
| `trackforge_webhook_deliveries_total` | Counter | Outbound webhooks by status |
| `trackforge_db_query_duration_seconds` | Histogram | Database query latency |

### Logging

Structured logging via `slog` (Go stdlib). JSON format in production, text format in development. Logs include trace IDs for correlation with traces.

## Deployment Architecture

### Docker Compose (Production)

```yaml
services:
  api:
    build: ./docker/Dockerfile.api
    ports: ["8080:8080"]
    depends_on: [postgres]
    env_file: .env

  web:
    build: ./docker/Dockerfile.web
    ports: ["3000:80"]
    depends_on: [api]

  postgres:
    image: postgres:16-alpine
    volumes: [pgdata:/var/lib/postgresql/data]
    env_file: .env

volumes:
  pgdata:
```

**That's it. Three services.** No Redis, no RabbitMQ, no separate worker, no Nginx sidecar for the API. The web container's Nginx handles static files and API proxying.

### Development

```bash
# Terminal 1: Database
docker compose up postgres

# Terminal 2: API server (with hot reload)
air  # or: go run ./cmd/server

# Terminal 3: Frontend (Vite dev server)
cd web && npm run dev
```

### Resource Requirements

Minimum: 1 CPU, 512MB RAM (API) + 256MB RAM (Postgres). Comfortable for thousands of work items and dozens of concurrent users.

## Future Extensibility

The architecture supports future enhancements without fundamental changes:

- **Full-text search**: PostgreSQL `tsvector` columns and GIN indexes (no Elasticsearch needed)
- **File attachments**: S3-compatible storage (MinIO for self-hosted) with references in the DB
- **Real-time updates**: Server-Sent Events (SSE) from the API server for live board updates
- **Mobile app**: API-first design means a mobile client just talks to the same endpoints
- **Plugin system**: Automation rules with webhook actions already provide a basic plugin model
