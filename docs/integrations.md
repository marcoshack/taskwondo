# Integrations

## Overview

Taskwondo integrates with external systems through inbound webhooks (receiving alerts and events) and outbound webhooks (sending notifications). The integration system is designed to work with the monitoring and communication tools commonly used in self-hosted infrastructure environments.

## Inbound Webhooks

Inbound webhooks allow external systems to create work items in Taskwondo automatically.

### Architecture

```
External System                      Taskwondo
─────────────                        ──────────
Prometheus Alertmanager  ──POST──▶  /webhooks/:endpointId
Grafana Alerts           ──POST──▶  /webhooks/:endpointId
Generic JSON             ──POST──▶  /webhooks/:endpointId
                                         │
                                    ┌────▼─────┐
                                    │ Validate  │ (HMAC / Bearer token)
                                    │ auth      │
                                    └────┬─────┘
                                         │
                                    ┌────▼─────┐
                                    │ Parse     │ (source-specific parser)
                                    │ payload   │
                                    └────┬─────┘
                                         │
                                    ┌────▼─────┐
                                    │ Create or │ (dedup by fingerprint)
                                    │ update    │
                                    │ work item │
                                    └────┬─────┘
                                         │
                                    ┌────▼─────┐
                                    │ Execute   │ (matching rules)
                                    │ automation│
                                    └──────────┘
```

### Webhook Endpoint Configuration

Each webhook endpoint is configured with:

```json
{
  "name": "Production Alertmanager",
  "source_type": "prometheus",
  "queue_id": "alert-tickets-queue-uuid",
  "secret": "generated-hmac-secret",
  "config": {
    "default_priority_map": {
      "critical": "critical",
      "warning": "high",
      "info": "medium"
    },
    "title_template": "[{{severity}}] {{alertname}}: {{summary}}",
    "description_template": "**Alert:** {{alertname}}\n**Instance:** {{instance}}\n**Severity:** {{severity}}\n\n{{description}}",
    "label_map": {
      "job": true,
      "service": true
    },
    "dedup_key": "alertname+instance",
    "auto_resolve_on_resolved": true
  }
}
```

### Prometheus Alertmanager

Taskwondo accepts the standard Alertmanager webhook payload format.

**Alertmanager webhook config:**
```yaml
# alertmanager.yml
receivers:
  - name: taskwondo
    webhook_configs:
      - url: 'https://taskwondo.yourdomain.com/webhooks/<endpoint-id>'
        http_config:
          bearer_token: 'your-webhook-secret'
        send_resolved: true
```

**Payload processing:**

| Alertmanager Field | Taskwondo Mapping |
|-------------------|-------------------|
| `alerts[].labels.alertname` | Title (via template) |
| `alerts[].annotations.summary` | Title supplement |
| `alerts[].annotations.description` | Description |
| `alerts[].labels.severity` | Priority (via priority_map) |
| `alerts[].labels.*` | Labels (configurable which to include) |
| `alerts[].fingerprint` | Dedup key |
| `alerts[].status` | If "resolved" and auto_resolve enabled, transition ticket |
| `alerts[].startsAt` | Stored in custom_fields |
| `alerts[].generatorURL` | Stored in custom_fields, linked in description |

**Deduplication:** When a webhook arrives with a fingerprint matching an existing open ticket in the same queue, Taskwondo adds a comment to the existing ticket instead of creating a new one. If the alert has `status: resolved`, the existing ticket is transitioned to "resolved" (if `auto_resolve_on_resolved` is enabled).

**Batch handling:** Alertmanager sends arrays of alerts. Each alert in the batch is processed independently, potentially creating or updating multiple tickets from a single webhook call.

### Grafana Alerts

Taskwondo accepts Grafana's webhook notification channel format.

**Grafana contact point config:**
```
Type: Webhook
URL: https://taskwondo.yourdomain.com/webhooks/<endpoint-id>
HTTP Method: POST
Authorization Header: Bearer your-webhook-secret
```

**Payload processing:**

| Grafana Field | Taskwondo Mapping |
|--------------|-------------------|
| `title` | Title |
| `message` | Description |
| `state` | Priority mapping (alerting->high, ok->auto-resolve) |
| `ruleUrl` | Linked in description |
| `evalMatches[].metric` | Added to description |
| `evalMatches[].value` | Added to description |
| `tags.*` | Labels |

### Generic Webhook

For systems that don't match Prometheus or Grafana formats, the generic webhook accepts any JSON payload with configurable field mapping.

**Endpoint config:**
```json
{
  "source_type": "generic",
  "config": {
    "field_mapping": {
      "title": "$.event.name",
      "description": "$.event.details",
      "priority": "$.event.severity",
      "dedup_key": "$.event.id"
    },
    "priority_map": {
      "error": "high",
      "warning": "medium",
      "info": "low"
    }
  }
}
```

Field mapping uses JSONPath expressions to extract values from arbitrary payloads.

### Webhook Security

All inbound webhooks are authenticated:

| Method | How It Works |
|--------|-------------|
| **HMAC signature** | Webhook signs the payload with the shared secret. Taskwondo validates `X-Webhook-Signature: sha256=<hex>` |
| **Bearer token** | Simple `Authorization: Bearer <token>` header. Easier to configure, suitable for most cases |

Additional protections:
- Rate limiting: 300 requests/min per endpoint
- Payload size limit: 1MB
- IP allowlisting (optional, configured per endpoint)
- All webhook receipts are logged with full payload for debugging

---

## Discord Integration

### Outbound: Notifications to Discord

Automated notifications to Discord channels via Discord webhooks. Configured as automation rule actions.

**Setup:**
1. Create a Discord webhook in your server/channel
2. Create an automation rule with `action_type: send_webhook`
3. Use Discord's webhook URL and format the payload as a Discord embed

**Example automation rule for critical alerts:**
```json
{
  "action_type": "send_webhook",
  "action_config": {
    "url": "https://discord.com/api/webhooks/XXXX/YYYY",
    "headers": {"Content-Type": "application/json"},
    "body_template": {
      "embeds": [{
        "title": "{{item.display_id}}: {{item.title}}",
        "url": "{{item.url}}",
        "color": 15158332,
        "fields": [
          {"name": "Priority", "value": "{{item.priority}}", "inline": true},
          {"name": "Status", "value": "{{item.status}}", "inline": true},
          {"name": "Assignee", "value": "{{item.assignee.name}}", "inline": true}
        ],
        "timestamp": "{{now}}"
      }]
    }
  }
}
```

### Inbound: Discord Bot (Future Enhancement)

A Discord bot allowing users to interact with Taskwondo from Discord:

**Commands:**
```
/ticket create <title>     Create a ticket from Discord
/ticket status <id>        Check ticket status
/ticket comment <id> <msg> Add a comment
/ticket assign <id> <user> Assign a ticket
/ticket list               List open tickets
```

**Implementation approach:**
- Separate Go process or goroutine within the API server
- Uses Discord Gateway API (discordgo library)
- Authenticates via API key to Taskwondo's own API
- Maps Discord users to Taskwondo users via linked accounts

Scoped as a future enhancement — outbound webhook notifications cover the most critical Discord use case initially.

---

## Email Integration

### Outbound Email

Taskwondo sends emails for:
- Portal verification codes
- Portal ticket notifications (new comment, status change, resolution)
- Internal user notifications (if email channel is enabled)

**Configuration (environment variables):**
```env
SMTP_HOST=smtp.example.com
SMTP_PORT=587
SMTP_USERNAME=taskwondo@example.com
SMTP_PASSWORD=...
SMTP_FROM=Taskwondo <taskwondo@example.com>
SMTP_TLS=true
```

**Email templates** are stored as Go templates in `internal/email/templates/`.

### Inbound Email (Future Enhancement)

Accept emails to create or update tickets:

```
Incoming email to: support+GAME@taskwondo.yourdomain.com
                          └──┘
                        queue key

1. Parse email (sender, subject, body, attachments)
2. Match sender email to portal contact
3. If subject contains ticket ID (e.g., "Re: [GAME-42]"):
   → Add comment to existing ticket
4. If no ticket ID:
   → Create new ticket in the matched queue
```

Requires either a self-hosted SMTP server or a third-party inbound email service. Scoped as a future enhancement.

---

## Outbound Webhooks

Taskwondo can POST to external URLs when events occur, enabling integration with any system that accepts webhooks.

### Configuration

Outbound webhooks are configured as automation rule actions (see [Workflows — Automation Rules](workflows.md#automation-rules)).

### Delivery and Retry

- Outbound webhooks are delivered asynchronously via the background runner
- Timeout: 10 seconds per delivery attempt
- Retry policy: 3 attempts with exponential backoff (10s, 60s, 300s)
- Failed deliveries are logged with full request/response for debugging
- Delivery status visible in the automation rule's activity log

### Delivery Log Schema

```sql
CREATE TABLE webhook_deliveries (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    automation_rule_id  UUID NOT NULL REFERENCES automation_rules(id),
    work_item_id        UUID NOT NULL REFERENCES work_items(id),
    url                 TEXT NOT NULL,
    request_headers     JSONB,
    request_body        JSONB,
    response_status     INTEGER,
    response_body       TEXT,
    status              TEXT NOT NULL CHECK (status IN ('pending', 'success', 'failed', 'retrying')),
    attempt_count       INTEGER NOT NULL DEFAULT 0,
    next_retry_at       TIMESTAMPTZ,
    error_message       TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at        TIMESTAMPTZ
);
```

---

## OpenTelemetry Integration

Taskwondo exports observability data via OpenTelemetry, integrating with your existing monitoring stack.

### Configuration

```env
OTEL_ENABLED=true
OTEL_EXPORTER_OTLP_ENDPOINT=http://grafana-alloy:4317
OTEL_SERVICE_NAME=taskwondo
OTEL_RESOURCE_ATTRIBUTES=deployment.environment=production
```

### What's Instrumented

| Signal | What | Details |
|--------|------|---------|
| Traces | HTTP requests | Every API request with method, path, status, duration |
| Traces | Database queries | SQL operations as child spans |
| Traces | Webhook processing | Inbound/outbound webhook handling |
| Traces | Automation rules | Rule evaluation and action execution |
| Metrics | HTTP metrics | Request count, latency histogram by endpoint |
| Metrics | Work item metrics | Created/resolved counts, open gauge by type |
| Metrics | Automation metrics | Rule triggers, action executions, failures |
| Metrics | Webhook metrics | Delivery count by status, latency |
| Logs | Structured logs | JSON format with trace ID correlation |

### Prometheus Metrics Endpoint

In addition to OTLP export, Taskwondo exposes a `/metrics` endpoint for direct Prometheus scraping. This is useful if you prefer pull-based metrics or don't have an OTLP collector.

---

## Integration Priority

For implementation, integrations should be built in this order:

1. **Prometheus Alertmanager inbound webhook** — Core use case for automated ticket creation
2. **Discord outbound webhooks** — Notifications for critical events
3. **SMTP outbound email** — Portal notifications
4. **Grafana inbound webhook** — Alternative alerting source
5. **Generic inbound webhook** — Catch-all for other systems
6. **OpenTelemetry** — Observability (can be added incrementally)
7. **Discord bot** — Future enhancement
8. **Inbound email** — Future enhancement
