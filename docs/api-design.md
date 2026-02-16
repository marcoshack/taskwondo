# API Design

## Overview

TrackForge exposes a RESTful JSON API. All internal and portal functionality is available via the API — the React frontend is a pure API client.

**Base URLs:**
- Internal API: `/api/v1/`
- Public Portal API: `/portal/api/v1/`
- Webhooks: `/webhooks/`

## Conventions

### General

- All request/response bodies are JSON (`Content-Type: application/json`)
- Dates are ISO 8601 with timezone (`2025-01-15T14:30:00Z`)
- IDs are UUIDs
- Pagination uses cursor-based pagination (keyset) with `?cursor=<id>&limit=50`
- Sorting via `?sort=created_at&order=desc`
- Filtering via query parameters: `?status=open&type=ticket&assignee=me`
- Bulk operations use `POST /api/v1/<resource>/bulk` endpoints
- Partial updates use `PATCH` with only the fields to change

### Responses

**Success:**
```json
{
  "data": { ... },
  "meta": {
    "cursor": "next-page-cursor",
    "has_more": true,
    "total": 142
  }
}
```

**Error:**
```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Title is required",
    "details": [
      {"field": "title", "message": "must not be empty"}
    ]
  }
}
```

**Standard error codes:**
| HTTP Status | Code | Meaning |
|-------------|------|---------|
| 400 | `VALIDATION_ERROR` | Invalid request body or parameters |
| 401 | `UNAUTHORIZED` | Missing or invalid authentication |
| 403 | `FORBIDDEN` | Insufficient permissions |
| 404 | `NOT_FOUND` | Resource doesn't exist or not accessible |
| 409 | `CONFLICT` | Conflicting state (e.g., invalid status transition) |
| 422 | `UNPROCESSABLE` | Valid syntax but semantically invalid |
| 429 | `RATE_LIMITED` | Too many requests |
| 500 | `INTERNAL_ERROR` | Server error |

### Authentication

**Internal API:**
- JWT via `Authorization: Bearer <token>` header or `session` httpOnly cookie
- API key via `Authorization: Bearer tfk_<key>` header

**Portal API:**
- Portal token via `Authorization: Bearer tfp_<token>` header or `portal_session` cookie
- Some portal endpoints are unauthenticated (e.g., ticket submission with email verification)

**Webhooks:**
- HMAC signature in `X-Webhook-Signature` header, or
- Bearer token in `Authorization` header

---

## Authentication Endpoints

### `POST /api/v1/auth/login`

Login with email and password. Returns JWT.

**Request:**
```json
{
  "email": "marcos@example.com",
  "password": "..."
}
```

**Response (200):**
```json
{
  "data": {
    "token": "eyJhbG...",
    "user": {
      "id": "uuid",
      "email": "marcos@example.com",
      "display_name": "Marcos",
      "global_role": "admin"
    }
  }
}
```

### `POST /api/v1/auth/refresh`
### `POST /api/v1/auth/logout`
### `GET /api/v1/auth/me`

---

## Project Endpoints

### `GET /api/v1/projects`

List projects the authenticated user is a member of.

**Query params:** `?cursor=&limit=20`

### `POST /api/v1/projects`

Create a new project.

**Request:**
```json
{
  "name": "Infrastructure",
  "key": "INFRA",
  "description": "Home lab and server infrastructure management"
}
```

### `GET /api/v1/projects/:projectKey`
### `PATCH /api/v1/projects/:projectKey`
### `DELETE /api/v1/projects/:projectKey`

### `GET /api/v1/projects/:projectKey/members`
### `POST /api/v1/projects/:projectKey/members`
### `PATCH /api/v1/projects/:projectKey/members/:userId`
### `DELETE /api/v1/projects/:projectKey/members/:userId`

---

## Work Item Endpoints

### `GET /api/v1/projects/:projectKey/items`

List work items with filtering and pagination.

**Query params:**
```
?type=ticket,bug           # filter by type (comma-separated)
&status=open,in_progress   # filter by status
&priority=critical,high    # filter by priority
&assignee=me               # "me" or user ID
&assignee=unassigned       # unassigned items
&queue=<queue-id>          # filter by queue
&label=urgent              # filter by label
&milestone=<milestone-id>  # filter by milestone
&parent=<item-id>          # children of a specific item
&parent=none               # top-level items only
&q=search+terms            # full-text search
&sort=created_at           # sort field (created_at, updated_at, priority, due_date)
&order=desc                # sort direction
&cursor=<cursor>           # pagination cursor
&limit=50                  # page size (max 100)
```

**Response (200):**
```json
{
  "data": [
    {
      "id": "uuid",
      "project_key": "INFRA",
      "item_number": 42,
      "display_id": "INFRA-42",
      "type": "ticket",
      "title": "Prometheus server unreachable",
      "description": "Alertmanager fired critical alert...",
      "status": "investigating",
      "status_category": "in_progress",
      "priority": "critical",
      "assignee": {
        "id": "uuid",
        "display_name": "Marcos",
        "avatar_url": "..."
      },
      "reporter": {
        "id": "uuid",
        "display_name": "System (Alertmanager)"
      },
      "queue": {
        "id": "uuid",
        "name": "Alert Tickets"
      },
      "labels": ["infrastructure", "monitoring"],
      "milestone": null,
      "parent": null,
      "child_count": 2,
      "comment_count": 5,
      "visibility": "internal",
      "due_date": null,
      "sla_deadline": "2025-01-15T16:30:00Z",
      "sla_status": "at_risk",
      "created_at": "2025-01-15T14:30:00Z",
      "updated_at": "2025-01-15T15:12:00Z"
    }
  ],
  "meta": {
    "cursor": "next-cursor",
    "has_more": true,
    "total": 87
  }
}
```

### `POST /api/v1/projects/:projectKey/items`

Create a work item.

**Request:**
```json
{
  "type": "task",
  "title": "Upgrade PostgreSQL to 17",
  "description": "Current version is 16.x, need to plan upgrade path...",
  "priority": "medium",
  "assignee_id": "uuid",
  "labels": ["database", "upgrade"],
  "milestone_id": "uuid",
  "parent_id": "uuid",
  "queue_id": null,
  "visibility": "internal",
  "due_date": "2025-02-01",
  "custom_fields": {
    "estimated_hours": 4
  }
}
```

**Response (201):**
```json
{
  "data": {
    "id": "uuid",
    "display_id": "INFRA-43",
    ...
  }
}
```

### `GET /api/v1/projects/:projectKey/items/:itemNumber`

Get a single work item by display number (e.g., `GET /api/v1/projects/INFRA/items/42`).

### `PATCH /api/v1/projects/:projectKey/items/:itemNumber`

Update a work item. Only include fields to change.

**Request:**
```json
{
  "status": "in_progress",
  "assignee_id": "uuid",
  "labels": ["database", "upgrade", "urgent"]
}
```

Each field change generates a `work_item_event` record and may trigger automation rules.

### `DELETE /api/v1/projects/:projectKey/items/:itemNumber`

Soft delete a work item.

### `POST /api/v1/projects/:projectKey/items/bulk`

Bulk operations on work items.

**Request:**
```json
{
  "item_ids": ["uuid1", "uuid2", "uuid3"],
  "action": "update",
  "fields": {
    "status": "done",
    "labels_add": ["released"],
    "labels_remove": ["in-sprint"]
  }
}
```

---

## Comments

### `GET /api/v1/projects/:projectKey/items/:itemNumber/comments`

**Query params:** `?visibility=all|internal|public`

### `POST /api/v1/projects/:projectKey/items/:itemNumber/comments`

**Request:**
```json
{
  "body": "Root cause identified: disk space exhaustion on /var/log",
  "visibility": "internal"
}
```

### `PATCH /api/v1/projects/:projectKey/items/:itemNumber/comments/:commentId`
### `DELETE /api/v1/projects/:projectKey/items/:itemNumber/comments/:commentId`

---

## Relations

### `GET /api/v1/projects/:projectKey/items/:itemNumber/relations`

### `POST /api/v1/projects/:projectKey/items/:itemNumber/relations`

**Request:**
```json
{
  "target_display_id": "INFRA-38",
  "relation_type": "caused_by"
}
```

The API accepts `target_display_id` (e.g., "INFRA-38") for convenience. Cross-project relations are supported.

### `DELETE /api/v1/projects/:projectKey/items/:itemNumber/relations/:relationId`

---

## Activity / Events

### `GET /api/v1/projects/:projectKey/items/:itemNumber/events`

Returns the full activity timeline for a work item.

**Query params:** `?visibility=all|internal|public`

**Response:**
```json
{
  "data": [
    {
      "id": "uuid",
      "event_type": "status_changed",
      "actor": {"id": "uuid", "display_name": "Marcos"},
      "field_name": "status",
      "old_value": "open",
      "new_value": "investigating",
      "metadata": {},
      "visibility": "public",
      "created_at": "2025-01-15T14:35:00Z"
    },
    {
      "id": "uuid",
      "event_type": "comment_added",
      "actor": {"id": "uuid", "display_name": "Marcos"},
      "metadata": {"comment_id": "uuid", "preview": "Root cause identified..."},
      "visibility": "internal",
      "created_at": "2025-01-15T15:12:00Z"
    }
  ]
}
```

---

## Queues

### `GET /api/v1/projects/:projectKey/queues`
### `POST /api/v1/projects/:projectKey/queues`
### `PATCH /api/v1/projects/:projectKey/queues/:queueId`
### `DELETE /api/v1/projects/:projectKey/queues/:queueId`

---

## Milestones

### `GET /api/v1/projects/:projectKey/milestones`
### `POST /api/v1/projects/:projectKey/milestones`
### `PATCH /api/v1/projects/:projectKey/milestones/:milestoneId`
### `DELETE /api/v1/projects/:projectKey/milestones/:milestoneId`

---

## Workflows

### `GET /api/v1/workflows`
### `POST /api/v1/workflows`
### `GET /api/v1/workflows/:workflowId`
### `PATCH /api/v1/workflows/:workflowId`
### `GET /api/v1/workflows/:workflowId/transitions`

Returns valid transitions from each status, useful for UI to show available actions.

---

## Automation Rules

### `GET /api/v1/projects/:projectKey/automation`
### `POST /api/v1/projects/:projectKey/automation`
### `PATCH /api/v1/projects/:projectKey/automation/:ruleId`
### `DELETE /api/v1/projects/:projectKey/automation/:ruleId`
### `POST /api/v1/projects/:projectKey/automation/:ruleId/test`

Test a rule against a specific work item without executing actions.

---

## Webhook Endpoints (Inbound)

### `GET /api/v1/projects/:projectKey/webhooks`
### `POST /api/v1/projects/:projectKey/webhooks`
### `PATCH /api/v1/projects/:projectKey/webhooks/:endpointId`
### `DELETE /api/v1/projects/:projectKey/webhooks/:endpointId`

### `POST /webhooks/:endpointId`

The actual inbound webhook receiver. Accepts payloads from Prometheus Alertmanager, Grafana, or generic JSON and creates work items based on the endpoint's configuration.

---

## Search

### `GET /api/v1/search`

Global search across all accessible projects.

**Query params:**
```
?q=prometheus+disk+space
&projects=INFRA,GAME        # limit to specific projects
&type=ticket,bug
&status=open,investigating
&limit=20
```

---

## Public Portal API

All portal endpoints are under `/portal/api/v1/`. Authentication is via portal session token.

### `POST /portal/api/v1/auth/request-code`

Request a verification code via email.

```json
{"email": "player@example.com"}
```

### `POST /portal/api/v1/auth/verify`

Verify email code and get a session token.

```json
{"email": "player@example.com", "code": "123456"}
```

### `GET /portal/api/v1/queues`

List public queues (across all projects that have public queues).

### `POST /portal/api/v1/queues/:queueId/tickets`

Submit a ticket to a public queue.

```json
{
  "title": "Can't connect to game server",
  "description": "Getting timeout error when trying to join...",
  "contact_metadata": {
    "player_id": "PLAYER-12345",
    "discord_username": "player#1234"
  }
}
```

### `GET /portal/api/v1/tickets`

List the authenticated portal contact's tickets.

### `GET /portal/api/v1/tickets/:ticketDisplayId`

Get a specific ticket with public comments and events.

### `POST /portal/api/v1/tickets/:ticketDisplayId/comments`

Add a public comment to the contact's own ticket.

```json
{"body": "I tried restarting but the issue persists"}
```

### `GET /portal/api/v1/kb`

Browse resolved/public items as a knowledge base (optional feature).

---

## API Keys

### `GET /api/v1/user/api-keys`
### `POST /api/v1/user/api-keys`

**Response includes the full key only once at creation time:**
```json
{
  "data": {
    "id": "uuid",
    "name": "CI/CD Pipeline",
    "key": "tfk_live_a1b2c3d4e5f6...",
    "key_prefix": "tfk_live_a1",
    "permissions": ["work_items:write", "comments:write"],
    "expires_at": "2026-01-15T00:00:00Z"
  }
}
```

### `DELETE /api/v1/user/api-keys/:keyId`

---

## Notifications

### `GET /api/v1/user/notifications`

Get notification preferences.

### `PUT /api/v1/user/notifications`

Update notification preferences.

---

## Health & Metrics

### `GET /healthz`

Basic health check (returns 200 if API is running).

### `GET /readyz`

Readiness check (returns 200 if database is reachable).

### `GET /metrics`

Prometheus metrics endpoint.

---

## Rate Limiting

| Endpoint Group | Limit |
|---------------|-------|
| Internal API (authenticated) | 1000 req/min per user |
| Portal API (authenticated) | 100 req/min per contact |
| Portal API (unauthenticated) | 20 req/min per IP |
| Webhook receivers | 300 req/min per endpoint |
| Auth endpoints (login, code request) | 10 req/min per IP |

Rate limit headers: `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset`
