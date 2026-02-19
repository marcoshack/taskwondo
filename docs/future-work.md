# Future Work

## Queues & Milestones

**Goal:** Queues for inbound work, milestones for tracking progress.

1. Create migration: `queues`, `milestones` tables, add `queue_id` and `milestone_id` to work items
2. Implement queue CRUD with default assignment and workflow override
3. Implement milestone CRUD with progress tracking (count of open/closed items)
4. Test: Create queues, assign items to queues, milestone progress updates


## Public Portal

**Goal:** Public-facing portal for ticket submission and tracking.

1. Create migration: `portal_contacts`, `portal_sessions` tables
2. Implement portal auth (email verification code flow)
3. Implement portal ticket submission endpoint
4. Implement portal ticket list/detail (scoped to own tickets, public visibility only)
5. Build portal React pages (separate route tree under `/portal/`)
6. Test: Submit ticket as anonymous user, verify email, track ticket, see public comments

## Automation Engine

**Goal:** Automation rules with triggers and actions.

1. Create migration: `automation_rules` table
2. Implement rule evaluation engine (match trigger conditions against events)
3. Implement action executors (set_field, add_comment, send_webhook, etc.)
4. Implement background runner for async action execution
5. Wire automation into work item service (evaluate rules after creates/updates)
6. Build automation rule management UI
7. Test: Create rules, trigger them via work item actions, verify actions execute

## Inbound Webhooks

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

## Outbound Notifications

**Goal:** Discord and email notifications for events.

1. Implement outbound webhook delivery with retry logic
2. Implement SMTP email sender
3. Implement notification preference management
4. Build notification settings UI
5. Test: Trigger automation rule with webhook action, verify delivery and retry

## Observability

**Goal:** OpenTelemetry traces and Prometheus metrics.

1. Add OpenTelemetry SDK initialization
2. Instrument HTTP middleware (traces + metrics)
3. Instrument database layer (query spans)
4. Instrument automation engine (rule execution spans)
5. Add Prometheus `/metrics` endpoint
6. Add custom business metrics (work items created, open count, etc.)
7. Test: Verify traces appear in collector, metrics scrapeable

## Polish & Hardening

1. SLA tracking implementation (timers, breach detection)
2. Knowledge base view in portal
3. API key management UI
4. Rate limiting implementation
5. Comprehensive error handling audit
6. Performance testing and query optimization
7. Documentation updates
