# Data Model

## Overview

TrackForge uses a unified work item model where tasks, tickets, bugs, and feedback are all the same underlying entity with different `type` values. This enables tight cross-referencing between project management and support/incident tracking without data duplication.

## Entity Relationship Diagram

```
┌─────────────┐       ┌──────────────────┐       ┌─────────────────┐
│    User      │       │     Project      │       │     Queue       │
│─────────────│       │──────────────────│       │─────────────────│
│ id           │──┐    │ id               │──┐    │ id              │
│ email        │  │    │ name             │  │    │ project_id (FK) │
│ display_name │  │    │ key (e.g. "TF")  │  │    │ name            │
│ role         │  │    │ description      │  │    │ type (support/  │
│ avatar_url   │  │    │ default_workflow │  │    │   alerts/       │
│ created_at   │  │    │ created_at       │  │    │   feedback)     │
└─────────────┘  │    └──────────────────┘  │    │ is_public       │
                  │             │             │    │ workflow_id     │
                  │             │             │    └─────────────────┘
                  │             ▼             │
                  │    ┌──────────────────┐  │
                  │    │ ProjectMember    │  │
                  │    │──────────────────│  │
                  ├───▶│ user_id (FK)     │  │
                  │    │ project_id (FK)  │  │
                  │    │ role             │  │
                  │    └──────────────────┘  │
                  │                          │
                  │    ┌──────────────────────────────────────┐
                  │    │             WorkItem                  │
                  │    │──────────────────────────────────────│
                  │    │ id (UUIDv7)                          │
                  │    │ project_id (FK)                      │◀─┐
                  │    │ queue_id (FK, nullable)              │  │
                  │    │ parent_id (FK, nullable, self-ref)   │──┘
                  │    │ item_number (per-project sequential) │
                  │    │ type (task|ticket|bug|feedback|epic) │
                  │    │ title                                │
                  │    │ description (markdown)               │
                  │    │ status                               │
                  │    │ priority (critical|high|medium|low)  │
                  │    │ assignee_id (FK, nullable)           │◀── User
                  │    │ reporter_id (FK)                     │◀── User
                  │    │ portal_contact_id (FK, nullable)     │◀── PortalContact
                  │    │ visibility (internal|portal|public)  │
                  │    │ labels (text[])                      │
                  │    │ custom_fields (jsonb)                │
                  │    │ due_date (nullable)                  │
                  │    │ sla_deadline (nullable)              │
                  │    │ resolved_at (nullable)               │
                  │    │ created_at                           │
                  │    │ updated_at                           │
                  │    │ deleted_at (nullable)                │
                  │    └──────────────────────────────────────┘
                  │                │          │
                  │                │          │
                  │                ▼          ▼
                  │    ┌───────────────┐  ┌──────────────────┐
                  │    │   Comment     │  │ WorkItemRelation │
                  │    │───────────────│  │──────────────────│
                  │    │ id            │  │ id               │
                  └───▶│ author_id(FK) │  │ source_id (FK)   │
                       │ work_item_id  │  │ target_id (FK)   │
                       │ body (md)     │  │ relation_type    │
                       │ visibility    │  │  (blocks|related │
                       │  (internal|   │  │   |duplicates|   │
                       │   public)     │  │   caused_by)     │
                       │ created_at    │  │ created_at       │
                       │ updated_at    │  └──────────────────┘
                       └───────────────┘
```

## Core Entities

### User

Internal users of the system (team members, admins).

```sql
CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email           TEXT NOT NULL UNIQUE,
    display_name    TEXT NOT NULL,
    password_hash   TEXT,              -- nullable if using OIDC only
    global_role     TEXT NOT NULL DEFAULT 'user' CHECK (global_role IN ('admin', 'user')),
    avatar_url      TEXT,
    is_active       BOOLEAN NOT NULL DEFAULT true,
    last_login_at   TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

### Project

Top-level organizational unit. Everything belongs to a project.

```sql
CREATE TABLE projects (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name                TEXT NOT NULL,
    key                 TEXT NOT NULL UNIQUE,  -- e.g., "TF", "INFRA", "GAME"
    description         TEXT,
    default_workflow_id UUID REFERENCES workflows(id),
    item_counter        INTEGER NOT NULL DEFAULT 0,  -- for sequential numbering
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at          TIMESTAMPTZ
);

-- Project key is used for work item display IDs: TF-42, INFRA-108
CREATE UNIQUE INDEX idx_projects_key ON projects(key) WHERE deleted_at IS NULL;
```

### ProjectMember

Associates users with projects and their role within that project.

```sql
CREATE TABLE project_members (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id  UUID NOT NULL REFERENCES projects(id),
    user_id     UUID NOT NULL REFERENCES users(id),
    role        TEXT NOT NULL DEFAULT 'member' CHECK (role IN ('owner', 'admin', 'member', 'viewer')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE (project_id, user_id)
);
```

### Queue

Inbound work channels within a project. A project can have multiple queues (e.g., "Customer Support", "Alert Tickets", "Community Feedback").

```sql
CREATE TABLE queues (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id      UUID NOT NULL REFERENCES projects(id),
    name            TEXT NOT NULL,
    description     TEXT,
    queue_type      TEXT NOT NULL CHECK (queue_type IN ('support', 'alerts', 'feedback', 'general')),
    is_public       BOOLEAN NOT NULL DEFAULT false,   -- visible on public portal
    default_priority TEXT NOT NULL DEFAULT 'medium',
    default_assignee_id UUID REFERENCES users(id),    -- auto-assign new items
    workflow_id     UUID REFERENCES workflows(id),
    sla_policy_id   UUID REFERENCES sla_policies(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE (project_id, name)
);
```

### WorkItem

The central entity. Every task, ticket, bug, feedback item, and epic is a work item.

```sql
CREATE TABLE work_items (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id          UUID NOT NULL REFERENCES projects(id),
    queue_id            UUID REFERENCES queues(id),
    parent_id           UUID REFERENCES work_items(id),  -- for sub-tasks, child items
    item_number         INTEGER NOT NULL,                 -- sequential within project
    type                TEXT NOT NULL CHECK (type IN ('task', 'ticket', 'bug', 'feedback', 'epic')),
    title               TEXT NOT NULL,
    description         TEXT,                             -- markdown
    status              TEXT NOT NULL DEFAULT 'open',
    priority            TEXT NOT NULL DEFAULT 'medium' CHECK (priority IN ('critical', 'high', 'medium', 'low')),
    assignee_id         UUID REFERENCES users(id),
    reporter_id         UUID NOT NULL REFERENCES users(id),
    portal_contact_id   UUID REFERENCES portal_contacts(id),  -- if submitted via portal
    visibility          TEXT NOT NULL DEFAULT 'internal' CHECK (visibility IN ('internal', 'portal', 'public')),
    labels              TEXT[] NOT NULL DEFAULT '{}',
    custom_fields       JSONB NOT NULL DEFAULT '{}',
    due_date            DATE,
    sla_deadline        TIMESTAMPTZ,
    resolved_at         TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at          TIMESTAMPTZ,

    UNIQUE (project_id, item_number)
);

-- Display ID is derived: project.key + '-' + item_number → "TF-42"

-- Core indexes
CREATE INDEX idx_work_items_project_status ON work_items(project_id, status) WHERE deleted_at IS NULL;
CREATE INDEX idx_work_items_assignee ON work_items(assignee_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_work_items_type ON work_items(project_id, type) WHERE deleted_at IS NULL;
CREATE INDEX idx_work_items_queue ON work_items(queue_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_work_items_parent ON work_items(parent_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_work_items_labels ON work_items USING GIN(labels) WHERE deleted_at IS NULL;
CREATE INDEX idx_work_items_created ON work_items(project_id, created_at DESC) WHERE deleted_at IS NULL;

-- Full-text search index
ALTER TABLE work_items ADD COLUMN search_vector tsvector
    GENERATED ALWAYS AS (
        setweight(to_tsvector('english', coalesce(title, '')), 'A') ||
        setweight(to_tsvector('english', coalesce(description, '')), 'B')
    ) STORED;
CREATE INDEX idx_work_items_search ON work_items USING GIN(search_vector);
```

### Comment

Comments on work items. Support both internal notes and public replies.

```sql
CREATE TABLE comments (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    work_item_id    UUID NOT NULL REFERENCES work_items(id),
    author_id       UUID REFERENCES users(id),              -- null if from portal contact
    portal_contact_id UUID REFERENCES portal_contacts(id),  -- null if from internal user
    body            TEXT NOT NULL,                           -- markdown
    visibility      TEXT NOT NULL DEFAULT 'internal' CHECK (visibility IN ('internal', 'public')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ
);

CREATE INDEX idx_comments_work_item ON comments(work_item_id, created_at) WHERE deleted_at IS NULL;
```

### WorkItemRelation

Directed relationships between work items.

```sql
CREATE TABLE work_item_relations (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_id       UUID NOT NULL REFERENCES work_items(id),
    target_id       UUID NOT NULL REFERENCES work_items(id),
    relation_type   TEXT NOT NULL CHECK (relation_type IN ('blocks', 'blocked_by', 'relates_to', 'duplicates', 'caused_by', 'parent_of', 'child_of')),
    created_by      UUID NOT NULL REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE (source_id, target_id, relation_type),
    CHECK (source_id != target_id)
);

CREATE INDEX idx_relations_source ON work_item_relations(source_id);
CREATE INDEX idx_relations_target ON work_item_relations(target_id);
```

### WorkItemEvent (Audit Log)

Every state change, assignment, comment, and action is recorded as an event. This provides a complete audit trail and powers the activity feed.

```sql
CREATE TABLE work_item_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    work_item_id    UUID NOT NULL REFERENCES work_items(id),
    actor_id        UUID REFERENCES users(id),          -- null for system/automation events
    event_type      TEXT NOT NULL,                       -- see event types below
    field_name      TEXT,                                -- which field changed
    old_value       TEXT,                                -- previous value (serialized)
    new_value       TEXT,                                -- new value (serialized)
    metadata        JSONB NOT NULL DEFAULT '{}',         -- additional context
    visibility      TEXT NOT NULL DEFAULT 'internal' CHECK (visibility IN ('internal', 'public')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Event types:
-- 'created', 'status_changed', 'assigned', 'unassigned', 'priority_changed',
-- 'label_added', 'label_removed', 'comment_added', 'relation_added',
-- 'relation_removed', 'description_updated', 'title_updated',
-- 'due_date_set', 'sla_breached', 'merged', 'reopened',
-- 'automation_triggered', 'webhook_received'

CREATE INDEX idx_events_work_item ON work_item_events(work_item_id, created_at);
CREATE INDEX idx_events_type ON work_item_events(event_type, created_at);
```

## Workflow System

### Workflow

Defines valid statuses and transitions for work items. Different projects or queues can use different workflows.

```sql
CREATE TABLE workflows (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    description TEXT,
    is_default  BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE workflow_statuses (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id     UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,           -- e.g., "open", "in_progress", "resolved"
    display_name    TEXT NOT NULL,           -- e.g., "Open", "In Progress", "Resolved"
    category        TEXT NOT NULL CHECK (category IN ('todo', 'in_progress', 'done', 'cancelled')),
    position        INTEGER NOT NULL,        -- display order
    color           TEXT,                    -- hex color for UI

    UNIQUE (workflow_id, name)
);

CREATE TABLE workflow_transitions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id     UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    from_status     TEXT NOT NULL,
    to_status       TEXT NOT NULL,
    name            TEXT,                    -- e.g., "Start Work", "Resolve", "Reopen"

    UNIQUE (workflow_id, from_status, to_status)
);
```

### Default Workflows

**Task Workflow** (for tasks, bugs, epics):
```
open → in_progress → in_review → done
  ↑         │            │         │
  └─────────┴────────────┴─────────┘ (reopen)
  any → cancelled
```

**Ticket Workflow** (for tickets, feedback):
```
new → triaged → investigating → waiting_on_customer → resolved → closed
 ↑       │          │                │                    │
 └───────┴──────────┴────────────────┴────────────────────┘ (reopen)
 any → cancelled
```

## Portal System

### PortalContact

External users who interact via the public portal. These are NOT internal users.

```sql
CREATE TABLE portal_contacts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email           TEXT NOT NULL,
    display_name    TEXT,
    is_verified     BOOLEAN NOT NULL DEFAULT false,
    metadata        JSONB NOT NULL DEFAULT '{}',    -- e.g., player ID, Discord username
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_portal_contacts_email ON portal_contacts(lower(email));
```

### PortalSession

Session management for portal users.

```sql
CREATE TABLE portal_sessions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    portal_contact_id UUID NOT NULL REFERENCES portal_contacts(id),
    token_hash      TEXT NOT NULL UNIQUE,
    expires_at      TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

## Automation System

### AutomationRule

Rules that trigger actions based on work item events.

```sql
CREATE TABLE automation_rules (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id      UUID NOT NULL REFERENCES projects(id),
    name            TEXT NOT NULL,
    description     TEXT,
    is_enabled      BOOLEAN NOT NULL DEFAULT true,
    trigger_type    TEXT NOT NULL,           -- 'work_item_created', 'status_changed', 'webhook_received', 'sla_approaching', 'schedule'
    trigger_config  JSONB NOT NULL,          -- conditions: {type: 'ticket', queue_id: '...', etc.}
    action_type     TEXT NOT NULL,           -- 'set_field', 'add_comment', 'send_webhook', 'create_work_item', 'send_notification'
    action_config   JSONB NOT NULL,          -- action parameters
    position        INTEGER NOT NULL DEFAULT 0,  -- execution order
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

### WebhookEndpoint

Registered inbound webhook endpoints for receiving alerts.

```sql
CREATE TABLE webhook_endpoints (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id      UUID NOT NULL REFERENCES projects(id),
    queue_id        UUID REFERENCES queues(id),     -- route incoming webhooks to a queue
    name            TEXT NOT NULL,
    source_type     TEXT NOT NULL CHECK (source_type IN ('prometheus', 'grafana', 'generic', 'discord', 'email')),
    secret_hash     TEXT NOT NULL,                   -- HMAC secret or bearer token hash
    config          JSONB NOT NULL DEFAULT '{}',     -- source-specific config (field mappings, etc.)
    is_enabled      BOOLEAN NOT NULL DEFAULT true,
    last_received_at TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

## SLA Policies

```sql
CREATE TABLE sla_policies (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            TEXT NOT NULL,
    description     TEXT,
    rules           JSONB NOT NULL,
    -- rules example:
    -- [
    --   {"priority": "critical", "response_minutes": 30, "resolution_minutes": 240},
    --   {"priority": "high", "response_minutes": 120, "resolution_minutes": 480},
    --   {"priority": "medium", "response_minutes": 480, "resolution_minutes": 2880},
    --   {"priority": "low", "response_minutes": 1440, "resolution_minutes": 10080}
    -- ]
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

## API Keys

```sql
CREATE TABLE api_keys (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id),
    name            TEXT NOT NULL,
    key_hash        TEXT NOT NULL UNIQUE,        -- bcrypt hash of the API key
    key_prefix      TEXT NOT NULL,               -- first 8 chars for identification
    permissions     TEXT[] NOT NULL DEFAULT '{}', -- scoped permissions
    last_used_at    TIMESTAMPTZ,
    expires_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

## Notification Preferences

```sql
CREATE TABLE notification_preferences (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id),
    channel         TEXT NOT NULL CHECK (channel IN ('email', 'discord', 'webhook')),
    event_types     TEXT[] NOT NULL DEFAULT '{}',  -- which events to notify on
    config          JSONB NOT NULL DEFAULT '{}',   -- channel-specific config (discord webhook URL, etc.)
    is_enabled      BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE (user_id, channel)
);
```

## Milestones

```sql
CREATE TABLE milestones (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id      UUID NOT NULL REFERENCES projects(id),
    name            TEXT NOT NULL,
    description     TEXT,
    due_date        DATE,
    status          TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'closed')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Work items reference milestones via:
ALTER TABLE work_items ADD COLUMN milestone_id UUID REFERENCES milestones(id);
CREATE INDEX idx_work_items_milestone ON work_items(milestone_id) WHERE deleted_at IS NULL;
```

## Key Design Decisions

1. **UUIDv7 for primary keys** — Time-ordered UUIDs give good index locality while remaining globally unique. No auto-increment ID leaking entity counts.

2. **Sequential item_number per project** — Users see `TF-42`, not a UUID. The counter is maintained via `UPDATE projects SET item_counter = item_counter + 1 ... RETURNING item_counter` in the same transaction as the work item insert.

3. **Labels as text arrays** — Simple, queryable with GIN indexes, no need for a separate labels table. `WHERE 'urgent' = ANY(labels)`.

4. **JSONB for custom fields and automation config** — Provides flexibility without schema changes. Custom fields can be defined per-project and stored as structured JSON.

5. **Soft deletes** — `deleted_at` on work items, comments, and projects. Hard delete on sessions and expired tokens.

6. **Generated tsvector column** — Full-text search on work items without external search infrastructure. PostgreSQL handles this natively with good performance.

7. **Event sourcing for audit** — The `work_item_events` table captures every change. This powers the activity timeline, enables undo patterns, and provides compliance-grade audit trails.
