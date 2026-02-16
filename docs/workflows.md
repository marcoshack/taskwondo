# Workflows

## Overview

TrackForge uses a flexible workflow system that defines valid statuses and transitions for work items. Different projects, queues, and work item types can use different workflows, enabling the same system to handle agile task boards, incident response pipelines, and customer support queues.

## Concepts

### Status

A named state that a work item can be in. Each status belongs to one of four **categories**:

| Category | Meaning | Board Column | Counted as Open? |
|----------|---------|-------------|-------------------|
| `todo` | Work hasn't started | To Do / Backlog | Yes |
| `in_progress` | Work is actively happening | In Progress | Yes |
| `done` | Work is complete | Done | No |
| `cancelled` | Work was abandoned | (hidden) | No |

Categories enable universal behaviors regardless of the specific status names a workflow uses:
- "Open items" = `todo` + `in_progress`
- Board views group by category
- SLA timers run while category is `todo` or `in_progress`
- Charts and metrics can aggregate across workflows

### Transition

A valid movement from one status to another. Transitions can be:
- **Named** — e.g., "Start Work" (open → in_progress), shown as action buttons in the UI
- **Conditional** — restricted by role or work item type (future enhancement)

### Workflow

A named collection of statuses and transitions. Workflows are assigned to:
- A **project** (default workflow for all items in the project)
- A **queue** (overrides the project workflow for items in that queue)

## Default Workflows

TrackForge ships with two built-in workflows. These are created during initial setup and can be customized.

### Task Workflow

For tasks, bugs, and epics. A simple kanban-style flow.

```
                    ┌──────────────────────┐
                    │                      ▼
┌────────┐    ┌─────────────┐    ┌───────────┐    ┌────────┐
│  Open  │───▶│ In Progress │───▶│ In Review │───▶│  Done  │
│ (todo) │    │(in_progress)│    │(in_progress)│   │ (done) │
└────────┘    └─────────────┘    └───────────┘    └────────┘
     │              │                  │               │
     │              ▼                  ▼               │
     │         ┌──────────┐      ┌──────────┐         │
     └────────▶│Cancelled │◀─────│          │◀────────┘
               │(cancelled)│      └──────────┘
               └──────────┘
```

**Statuses:**
| Name | Display Name | Category | Position |
|------|-------------|----------|----------|
| `open` | Open | todo | 0 |
| `in_progress` | In Progress | in_progress | 1 |
| `in_review` | In Review | in_progress | 2 |
| `done` | Done | done | 3 |
| `cancelled` | Cancelled | cancelled | 4 |

**Transitions:**
| From | To | Name |
|------|----|------|
| `open` | `in_progress` | Start Work |
| `open` | `cancelled` | Cancel |
| `in_progress` | `in_review` | Submit for Review |
| `in_progress` | `open` | Move to Backlog |
| `in_progress` | `cancelled` | Cancel |
| `in_review` | `done` | Approve |
| `in_review` | `in_progress` | Request Changes |
| `done` | `open` | Reopen |

### Ticket Workflow

For support tickets, alert-generated tickets, and feedback. Designed for incident response and customer support.

```
┌───────┐    ┌─────────┐    ┌───────────────┐    ┌─────────────────────┐
│  New  │───▶│ Triaged │───▶│ Investigating │───▶│ Waiting on Customer │
│(todo) │    │ (todo)  │    │ (in_progress) │    │    (in_progress)    │
└───────┘    └─────────┘    └───────────────┘    └─────────────────────┘
                                    │                       │
                                    ▼                       ▼
                             ┌───────────┐           ┌──────────┐
                             │ Resolved  │──────────▶│  Closed  │
                             │  (done)   │           │  (done)  │
                             └───────────┘           └──────────┘
```

**Statuses:**
| Name | Display Name | Category | Position |
|------|-------------|----------|----------|
| `new` | New | todo | 0 |
| `triaged` | Triaged | todo | 1 |
| `investigating` | Investigating | in_progress | 2 |
| `waiting_on_customer` | Waiting on Customer | in_progress | 3 |
| `resolved` | Resolved | done | 4 |
| `closed` | Closed | done | 5 |
| `cancelled` | Cancelled | cancelled | 6 |

**Transitions:**
| From | To | Name |
|------|----|------|
| `new` | `triaged` | Triage |
| `new` | `investigating` | Start Investigation |
| `new` | `cancelled` | Cancel |
| `triaged` | `investigating` | Start Investigation |
| `triaged` | `cancelled` | Cancel |
| `investigating` | `waiting_on_customer` | Waiting on Customer |
| `investigating` | `resolved` | Resolve |
| `investigating` | `triaged` | Back to Triage |
| `waiting_on_customer` | `investigating` | Customer Responded |
| `waiting_on_customer` | `resolved` | Resolve |
| `resolved` | `closed` | Close |
| `resolved` | `investigating` | Reopen |
| `closed` | `investigating` | Reopen |

## Work Item Lifecycle

### Creation

When a work item is created:
1. Validate required fields (title, type, project)
2. Assign `item_number` (atomic increment on project counter)
3. Set initial status (first status in the applicable workflow)
4. Apply queue defaults (priority, assignee) if created in a queue
5. Record `created` event
6. Execute matching automation rules (trigger: `work_item_created`)

### Status Transitions

When a status change is requested:
1. Validate the transition is allowed by the workflow
2. Update the `status` field
3. If transitioning to a `done` category, set `resolved_at`
4. If transitioning from `done` to a non-done category, clear `resolved_at`
5. Record `status_changed` event
6. Execute matching automation rules (trigger: `status_changed`)
7. Evaluate SLA impact

### Closure

There is no special "close" action beyond transitioning to a `done` or `cancelled` status. The workflow defines what "done" means for each context.

### Deletion

Soft delete only. Sets `deleted_at` timestamp. Deleted items:
- Don't appear in list queries
- Can be viewed directly by ID (with a "deleted" indicator)
- Can be restored by clearing `deleted_at`
- Are permanently purged by a background cleanup job (configurable retention, default 90 days)

## Automation Rules

Automation rules execute actions in response to work item events. They're the backbone of TrackForge's operational automation.

### Rule Structure

```json
{
  "id": "uuid",
  "name": "Auto-assign critical alerts to on-call",
  "trigger_type": "work_item_created",
  "trigger_config": {
    "conditions": {
      "type": "ticket",
      "priority": "critical",
      "queue_id": "alert-queue-uuid"
    }
  },
  "action_type": "set_field",
  "action_config": {
    "field": "assignee_id",
    "value": "oncall-user-uuid"
  }
}
```

### Trigger Types

| Trigger | Fires When | Config Options |
|---------|-----------|----------------|
| `work_item_created` | New work item is created | Filter by type, queue, priority, labels |
| `status_changed` | Status transitions | Filter by from_status, to_status, type |
| `field_changed` | Any field is updated | Filter by field_name, old/new value patterns |
| `comment_added` | Comment is posted | Filter by visibility (internal/public) |
| `webhook_received` | Inbound webhook fires | Filter by source_type, payload patterns |
| `sla_approaching` | SLA deadline is near | Filter by time remaining (e.g., 30min before breach) |
| `sla_breached` | SLA deadline passed | Filter by priority, queue |
| `schedule` | Cron-based schedule | Cron expression, filter for matching items |

### Action Types

| Action | What It Does | Config |
|--------|-------------|--------|
| `set_field` | Update a field on the work item | `{field, value}` |
| `add_label` | Add a label | `{label}` |
| `remove_label` | Remove a label | `{label}` |
| `add_comment` | Post an automated comment | `{body, visibility}` |
| `create_work_item` | Create a linked work item | `{type, title_template, ...}` |
| `send_webhook` | POST to an external URL | `{url, headers, body_template}` |
| `send_notification` | Notify via configured channels | `{channels, message_template}` |
| `transition` | Move to a different status | `{status}` |
| `assign` | Assign to a user | `{user_id}` or `{strategy: "round_robin"}` |

### Template Variables

Action configs support template variables for dynamic values:

```
{{item.display_id}}      → "INFRA-42"
{{item.title}}           → "Prometheus server unreachable"
{{item.priority}}        → "critical"
{{item.status}}          → "new"
{{item.assignee.name}}   → "Marcos"
{{item.reporter.name}}   → "System (Alertmanager)"
{{item.url}}             → "https://trackforge.local/app/INFRA/items/42"
{{event.type}}           → "status_changed"
{{event.old_value}}      → "open"
{{event.new_value}}      → "investigating"
{{now}}                  → "2025-01-15T14:30:00Z"
```

### Rule Execution

Rules are evaluated in `position` order within a project. When a triggering event occurs:

1. Find all enabled rules for the project matching the trigger type
2. Evaluate each rule's `trigger_config.conditions` against the event/item
3. For matching rules, enqueue the action for background execution
4. Actions execute asynchronously (but typically within milliseconds)
5. Each execution is logged in `work_item_events` with `event_type: automation_triggered`

### Example Automation Scenarios

**Auto-triage Prometheus alerts by severity:**
```json
{
  "name": "Critical alerts → P1 ticket",
  "trigger_type": "webhook_received",
  "trigger_config": {
    "conditions": {
      "source_type": "prometheus",
      "payload_match": {"severity": "critical"}
    }
  },
  "action_type": "set_field",
  "action_config": {
    "field": "priority",
    "value": "critical"
  }
}
```

**Auto-resolve stale waiting tickets:**
```json
{
  "name": "Auto-close after 7 days waiting",
  "trigger_type": "schedule",
  "trigger_config": {
    "cron": "0 9 * * *",
    "item_filter": {
      "status": "waiting_on_customer",
      "updated_before": "-7d"
    }
  },
  "action_type": "transition",
  "action_config": {
    "status": "resolved"
  }
}
```

**Notify Discord on critical ticket creation:**
```json
{
  "name": "Discord alert for critical tickets",
  "trigger_type": "work_item_created",
  "trigger_config": {
    "conditions": {
      "priority": "critical",
      "type": "ticket"
    }
  },
  "action_type": "send_webhook",
  "action_config": {
    "url": "https://discord.com/api/webhooks/...",
    "body_template": {
      "content": "🚨 Critical ticket: {{item.display_id}} - {{item.title}}\n{{item.url}}"
    }
  }
}
```

**Link duplicate customer reports:**
```json
{
  "name": "Auto-label portal submissions",
  "trigger_type": "work_item_created",
  "trigger_config": {
    "conditions": {
      "type": "ticket",
      "queue_type": "support"
    }
  },
  "action_type": "add_label",
  "action_config": {
    "label": "customer-reported"
  }
}
```

## SLA Tracking

### SLA Policies

SLA policies define response and resolution time targets by priority level.

```json
{
  "name": "Standard Support SLA",
  "rules": [
    {"priority": "critical", "response_minutes": 30,   "resolution_minutes": 240},
    {"priority": "high",     "response_minutes": 120,  "resolution_minutes": 480},
    {"priority": "medium",   "response_minutes": 480,  "resolution_minutes": 2880},
    {"priority": "low",      "response_minutes": 1440, "resolution_minutes": 10080}
  ]
}
```

### SLA Timer Behavior

- **Response SLA** starts when the ticket is created, stops when a team member first comments or changes status from `new`
- **Resolution SLA** starts when the ticket is created, pauses while in `waiting_on_customer` status, stops when transitioning to a `done` category
- `sla_deadline` on the work item reflects the nearest upcoming SLA deadline
- `sla_status` is computed: `ok`, `at_risk` (within 20% of deadline), `breached`

### SLA in the API

Work items include computed SLA fields in responses:
```json
{
  "sla_deadline": "2025-01-15T16:30:00Z",
  "sla_status": "at_risk",
  "sla_response_at": "2025-01-15T14:45:00Z",
  "sla_paused_minutes": 0
}
```

SLA breaches can trigger automation rules (`sla_approaching`, `sla_breached` triggers) for escalation.
