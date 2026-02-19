# Public Portal

## Overview

The public portal is a customer-facing interface where external users (players, community members, customers) can submit tickets, track their status, and browse resolved issues. It's a scoped, limited view of the same data that internal users see — not a separate system.

## Design Principles

1. **Same data, different lens** — Portal tickets are regular work items with `visibility: portal` or `visibility: public`. No data duplication.
2. **Minimal friction** — Email-based authentication, no passwords to remember. Submit a ticket in under 30 seconds.
3. **Privacy by default** — Portal users only see their own tickets and public items. Internal notes, other users' tickets, and internal work items are never visible.
4. **Branded and embeddable** — The portal can be themed per project and embedded in external sites via iframe.

## Portal Architecture

### URL Structure

```
/portal/                          → Portal home (list of public queues)
/portal/q/:queueSlug              → Queue view (submit ticket here)
/portal/q/:queueSlug/submit       → Ticket submission form
/portal/tickets                   → My tickets (requires auth)
/portal/tickets/:displayId        → Ticket detail with public timeline
/portal/kb                        → Knowledge base (resolved public items)
/portal/kb/:displayId             → Knowledge base article detail
```

### Authentication Flow

Portal uses passwordless email verification:

```
1. User enters email
2. System sends 6-digit code (valid 10 minutes)
3. User enters code
4. System creates portal session (30-day expiry)
5. Session stored as httpOnly cookie + returned as bearer token
```

For **anonymous submission** (if enabled on the queue):
- User submits ticket with just email + details
- Verification email sent with link to claim the ticket
- User can verify later to track status and respond

### Portal Contact vs User

Portal contacts are separate from internal users:

| | Internal User | Portal Contact |
|---|---|---|
| Auth | Email/password or OIDC | Email verification code |
| Visibility | All internal + portal + public items | Own tickets + public items only |
| Actions | Full CRUD, admin, automation | Submit tickets, comment on own tickets |
| Data | Full user profile | Email, display name, metadata |

If an internal user accesses the portal, they're treated as a portal contact (separate session). This keeps the portal experience consistent and prevents accidental exposure of internal data.

## Data Visibility Rules

### Work Item Visibility

| Visibility | Internal Users | Portal Contact (owner) | Portal Contact (other) | Anonymous |
|-----------|---------------|----------------------|----------------------|-----------|
| `internal` | ✅ | ❌ | ❌ | ❌ |
| `portal` | ✅ | ✅ (own ticket only) | ❌ | ❌ |
| `public` | ✅ | ✅ | ✅ | ✅ |

### Comment Visibility

| Visibility | Internal Users | Portal Contact (owner) | Anonymous |
|-----------|---------------|----------------------|-----------|
| `internal` | ✅ | ❌ | ❌ |
| `public` | ✅ | ✅ | ✅ (on public items) |

Internal users can see everything. Portal contacts only see `public` comments on their tickets. This lets the team have private discussions (internal comments) alongside customer-facing updates (public comments).

### Event Visibility

Same rules as comments. The portal timeline shows only public events:
- Status changes (e.g., "Status changed to Investigating")
- Public comments
- Resolution

Internal events like assignment changes, internal notes, and automation triggers are hidden from the portal.

## Ticket Submission

### Submission Flow

```
┌─────────────────────┐
│  Portal Home Page    │
│                      │
│  Select a queue:     │
│  ┌─────────────────┐ │
│  │ Game Support    │ │
│  │ Bug Reports     │ │
│  │ Feature Ideas   │ │
│  └─────────────────┘ │
└──────────┬──────────┘
           ▼
┌─────────────────────┐
│  Submission Form     │
│                      │
│  Email: [________]   │
│  Title: [________]   │
│  Description:        │
│  [________________]  │
│  [________________]  │
│                      │
│  Category: [v]       │  ← optional, queue-specific
│  Attachments: [+]    │  ← future
│                      │
│  [Submit Ticket]     │
└──────────┬──────────┘
           ▼
┌─────────────────────┐
│  Confirmation        │
│                      │
│  Ticket GAME-142     │
│  created!            │
│                      │
│  Verify your email   │
│  to track progress.  │
└─────────────────────┘
```

### Submission Form Configuration

Each queue can customize what the submission form collects:

```json
{
  "queue_id": "uuid",
  "form_config": {
    "fields": [
      {"name": "title", "type": "text", "required": true, "label": "Subject"},
      {"name": "description", "type": "textarea", "required": true, "label": "Describe your issue"},
      {"name": "category", "type": "select", "required": false, "label": "Category",
       "options": ["Connectivity", "Gameplay", "Account", "Other"]},
      {"name": "player_id", "type": "text", "required": false, "label": "Your Player ID"},
      {"name": "discord_username", "type": "text", "required": false, "label": "Discord Username"}
    ],
    "description_placeholder": "Please include any error messages, what you were doing when the issue occurred, and your platform (PC/console).",
    "success_message": "Thanks for reaching out! We'll review your ticket within 24 hours."
  }
}
```

Custom fields from the form are stored in `work_item.custom_fields` and `portal_contact.metadata`.

## Knowledge Base

The knowledge base is an optional feature that surfaces resolved public work items as searchable reference articles.

### How It Works

1. When a ticket/bug is resolved and has `visibility: public`, it becomes a KB candidate
2. Team members can "promote" an item to the KB by adding a `kb_article` label
3. The KB view shows these items with their title, description, and public comments as a thread
4. Full-text search across KB articles uses the same PostgreSQL search infrastructure

### KB Display

```
┌─────────────────────────────────────────┐
│  Knowledge Base                          │
│                                          │
│  Search: [_________________________]     │
│                                          │
│  Recent Solutions:                       │
│                                          │
│  GAME-89: Can't connect after update     │
│  Resolved 3 days ago · Connectivity      │
│                                          │
│  GAME-76: Character inventory not loading│
│  Resolved 1 week ago · Gameplay          │
│                                          │
│  INFRA-31: API intermittent timeouts     │
│  Resolved 2 weeks ago · Infrastructure   │
└─────────────────────────────────────────┘
```

## Portal Theming

The portal supports basic theming per project:

```json
{
  "project_id": "uuid",
  "portal_theme": {
    "logo_url": "/portal/assets/logo.png",
    "primary_color": "#4F46E5",
    "accent_color": "#10B981",
    "header_text": "Game Server Support",
    "footer_text": "Powered by Taskwondo",
    "custom_css": ""
  }
}
```

This allows the portal to feel integrated with the community or game's branding rather than looking like a generic tool.

## Spam & Abuse Prevention

### Rate Limiting

- 5 ticket submissions per email per hour
- 20 comments per portal session per hour
- 3 verification code requests per email per hour

### Content Filtering

- Basic profanity filter (configurable word list)
- Honeypot field in submission form (hidden field that bots fill)
- Optional CAPTCHA integration (hCaptcha or Turnstile) for anonymous submissions

### Moderation

- Queue admins can ban portal contacts (by email or email domain)
- Banned contacts can't submit new tickets or comment
- Existing tickets from banned contacts remain visible to the team

## Email Notifications

Portal contacts receive email notifications for:

| Event | Email Sent |
|-------|-----------|
| Ticket created | Confirmation with ticket ID and portal link |
| Public comment added by team | "New update on your ticket" |
| Status changed to resolved/closed | "Your ticket has been resolved" |
| Verification code requested | 6-digit code for portal login |

Emails are sent via SMTP (configurable in environment variables). Templates are simple text/HTML with the standard template variables.

## Embedding

The portal can be embedded in external sites:

```html
<!-- Full portal embed -->
<iframe src="https://taskwondo.yourdomain.com/portal/q/game-support" 
        width="100%" height="600"></iframe>

<!-- Submission form only -->
<iframe src="https://taskwondo.yourdomain.com/portal/q/game-support/submit?embed=true" 
        width="100%" height="500"></iframe>
```

The `embed=true` parameter triggers a minimal layout without header/footer navigation.

## API Reference

See [API Design — Public Portal API](api-design.md#public-portal-api) for the full endpoint specification.
