# TrackForge

**A self-hosted task and ticket management system with integrated project management and public-facing support portal.**

TrackForge combines project/task management (like a simplified JIRA) with a ticketing system for incident tracking and customer support — all in a single, tightly integrated platform.

## Why TrackForge?

Most self-hosted project management tools fall into two camps:
- **Enterprise behemoths** with 10+ services, multiple databases, and painful setup (looking at you, Plane)
- **Glorified to-do lists** that lack the workflow, automation, and integration capabilities needed for real operations

TrackForge sits in the middle: powerful enough for real project management and incident tracking, simple enough to deploy with a minimal Docker Compose stack.

## Core Concepts

### Unified Work Item Model

Tasks, tickets, bugs, and feedback are all **work items** — the same underlying data model with different workflows, visibility rules, and automation triggers. This means:

- A customer support ticket can be directly linked to an engineering task
- An automated alert from Prometheus creates a ticket that links to the project tracking its resolution
- A public feedback item can be promoted to an internal feature task
- All history, comments, and relationships are preserved across types

### Projects & Queues

- **Projects** organize internal work (tasks, bugs, epics) with boards, milestones, and sprint-like cycles
- **Queues** handle inbound work (support tickets, alerts, feedback) with triage workflows and SLA tracking
- Both use the same work item model, so cross-referencing is native

### Public Portal

A scoped, read-only (plus submission) interface where external users (players, community members, customers) can:
- Submit tickets and feedback
- Track the status of their submissions
- See public comments and resolutions
- Browse a knowledge base of resolved issues

### Automation & Integrations

- **Webhook receiver** for Prometheus Alertmanager / Grafana alerts → auto-create tickets
- **Discord bot** for notifications and quick ticket creation
- **Email inbound** (optional) for ticket creation via email
- **Automation rules** for auto-assignment, labeling, status transitions, and escalation

## Tech Stack

| Component | Technology |
|-----------|-----------|
| Backend API | Go (stdlib net/http + chi router) |
| Database | PostgreSQL 16 |
| Frontend | React 18 + TypeScript + Vite |
| Styling | Tailwind CSS |
| Auth | JWT + API keys (optional OIDC) |
| Deployment | Docker Compose (3 services: api, frontend, postgres) |
| Observability | OpenTelemetry → Prometheus/Grafana |

## Architecture Overview

```
┌──────────────────────────────────────────────────────────┐
│                     Docker Compose                       │
│                                                          │
│  ┌─────────────┐  ┌─────────────┐  ┌──────────────────┐  │
│  │  Frontend   │  │  API Server │  │   PostgreSQL     │  │
│  │  (React/TS) │──│  (Go)       │──│                  │  │
│  │  Nginx      │  │             │  │  Single DB       │  │
│  │  :3000      │  │  :8080      │  │  :5432           │  │
│  └─────────────┘  └──────┬──────┘  └──────────────────┘  │
│                          │                               │
└──────────────────────────┼───────────────────────────────┘
                           │
              ┌────────────┼────────────┐
              │            │            │
         Webhooks     Discord Bot   Email Inbound
         (Prometheus,  (optional)   (optional)
          Grafana)
```

## Deployment

```bash
# Clone and configure
git clone https://github.com/youruser/trackforge.git
cd trackforge
cp .env.example .env
# Edit .env with your settings

# Start
docker compose up -d

# Access
# Frontend:     http://localhost:3000
# API:          http://localhost:8080
# Health check: http://localhost:8080/healthz
```

## Project Structure

```
trackforge/
├── api/                     # Go API server
│   ├── cmd/
│   │   └── server/          # Application entrypoint
│   │       └── main.go
│   ├── internal/
│   │   ├── config/          # Configuration loading
│   │   ├── database/        # Database connection, migrations
│   │   │   └── migrations/  # SQL migration files
│   │   ├── handler/         # HTTP handlers (grouped by domain)
│   │   ├── middleware/      # Auth, logging, CORS, rate limiting
│   │   ├── model/           # Domain types and interfaces
│   │   ├── repository/      # Database access layer
│   │   ├── service/         # Business logic layer
│   │   ├── automation/      # Rule engine, webhook processing
│   │   └── otel/            # OpenTelemetry setup
│   ├── go.mod
│   └── go.sum
├── web/                     # React frontend
│   ├── src/
│   │   ├── components/
│   │   ├── pages/
│   │   ├── hooks/
│   │   ├── api/             # API client
│   │   ├── types/
│   │   └── portal/          # Public portal pages
│   ├── package.json
│   └── vite.config.ts
├── docs/                    # Design documents
├── docker/
│   ├── Dockerfile.api
│   ├── Dockerfile.web
│   └── nginx.conf
├── docker-compose.yml
├── .env.example
├── Makefile
└── AGENTS.md                # AI agent implementation guide
```

## Documentation

- [Architecture](docs/architecture.md) — System design, component details, deployment model
- [Data Model](docs/data-model.md) — Entities, relationships, database schema
- [API Design](docs/api-design.md) — REST endpoints, request/response schemas, auth
- [Access Control](docs/access-control.md) - How user permissions are structured
- [Workflows](docs/workflows.md) — Work item lifecycles, automation rules, state machines
- [Public Portal](docs/public-portal.md) — Public interface spec, permissions, submission flow
- [Integrations](docs/integrations.md) — Prometheus, Discord, email, webhooks
- [AGENTS.md](AGENTS.md) — Implementation guidance for AI coding agents

## License

MIT
