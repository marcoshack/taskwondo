# v0.3.0

Namespace multi-tenancy, AI-powered semantic search, SLA breach monitoring with escalations, Prometheus observability, user profiles with avatars, admin dashboards, CI/CD with GitHub Actions, and 130+ commits across the full stack.

## New Features

- **Namespace multi-tenancy** — organize projects under namespaces with URL-scoped paths, custom icons/colors, ownership limits, per-user namespace filtering, and cross-namespace project switcher
- **Semantic search** — pgvector + Ollama embeddings with Ctrl+K global search modal, FTS fallback, deep-linking, and result snippets
- **Unified search endpoint** — combined FTS and semantic search behind `/api/v1/search` with status, display ID, and namespace in results
- **SLA breach monitoring** — email notification engine for SLA breaches with escalation lists, per-type assignment, and priority-based SLA targets
- **Prometheus metrics** — `/metrics` endpoint with gauge metrics for resource counts (projects, users, work items, etc.)
- **User profiles** — profile page with display name and avatar management, avatars shown across all UI locations
- **Milestone dashboard** — dedicated milestone page with stats API and progress visualization
- **Admin dashboard** — Projects & Namespaces inspection page with stats summary on the admin Projects page
- **System API keys** — resource-based permissions for system-level API keys (distinct from user API keys)
- **Email project invites** — invite users to projects via email with notification delivery
- **UI layout modes** — Expanded and Centered layout options for different screen preferences
- **Children progress** — parent work items show progress bar based on children completion
- **Activity graph time range** — custom time range selector with persistent period preference
- **Mention search modal** — replaced `@` autocomplete with mini search modal using unified search
- **Localized API errors** — API error messages returned with error keys for client-side i18n
- **Namespace deny list** — reserved slugs and project keys blocked from user creation

## Improvements

- **CI/CD pipeline** — GitHub Actions with build, test, smoke test, Docker publish to GHCR, nightly releases, and tagged versioned releases
- **Completed item styling** — strikethrough preference toggle and done/cancelled visual effect across all views (list, board, cards)
- **Unified top bar layout** — consistent search bar and action row across Inbox, Watchlist, and Work Items pages
- **Inbox enhancements** — project filter, refresh button with auto-refresh, highlight previously opened items, back-to-source navigation
- **Email notifications** — remaining notification types implemented, sent in user's preferred language, assignee names resolved in watcher emails
- **Board view** — navigation chevrons for horizontal scrolling, Backlog status ordering fix
- **Mobile responsiveness** — milestone in detail metadata, reworked mobile top bar, rem-based toolbar heights
- **Project privacy** — admin option to hide non-member projects, extended to filter namespaces
- **Project switcher** — all-namespaces toggle, show projects across all namespaces, auto-select current namespace in New Project modal
- **Search result navigation** — uses result's namespace for correct routing
- **Admin workflow routes** — moved system workflow endpoints to `/api/v1/admin/workflows`
- **User scoping** — `/api/v1/users` scoped to co-project members for privacy
- **API key rename** — ability to rename existing API keys
- **MCP server** — MCPB bundle for Claude Desktop on Windows, milestone filter and labels in `list_work_items`, namespace support
- **Expired token cleanup** — periodic background job to purge expired email verification tokens
- **Notification preferences** — "Added to project" notification moved to global preferences

## Bug Fixes

- Fix SLA ordering and backfill on policy creation
- Fix retroactive SLA notifications and duplicate email race condition
- Fix "Project not found" flash when switching namespaces
- Fix namespace icon not showing in New Project modal
- Fix sidebar namespace badge not showing after creation
- Fix duplicate `display_id` constraint violation across namespaces
- Fix namespace settings mobile icon spacing
- Fix mention modal positioning on scrolled pages
- Fix reporter name showing UUID for non-member users
- Fix missing `backlog → in_progress` workflow transition
- Fix "Back to..." link to return to source page instead of list
- Fix inbox search box losing focus while typing
- Fix refresh button height mismatch on mobile Inbox
- Fix attachment list filename overflow on mobile
- Fix modal header text overlapping action icons on mobile
- Fix top bar overflow on User Preferences on mobile
- Fix missing bottom border on DataTable
- Strengthen email validation in registration
- Backfill `resolved_at` for work items resolved before the field was managed
- Fix various flaky E2E tests (namespace, notifications, keyboard shortcuts, board view)

---

# v0.2.0

Major productivity features for daily work item management, project invite links, background job processing with NATS JetStream, an MCP server for AI tool integration, broader authentication options, and 100+ automated E2E tests.

## New Features

- **Time tracking** — log time entries on work items with duration, description, and start time
- **Work item watchers** — watch items and receive notifications on changes and comments
- **Personal inbox** — "Work Next" queue for triaging and prioritizing your own work, with reordering
- **Saved searches** — save and reorder work item filter combinations for quick access
- **Email notifications** — assignment and watcher notifications via SMTP
- **MCP server** — Model Context Protocol integration with full API coverage (work items, comments, relations, attachments, time entries, inbox, milestones)
- **Project invite links** — share a link to let users join a project with a specific role
- **Project activity graph** — stats timeline on the project overview page (feature-toggled)
- **Mermaid diagrams** — render mermaid syntax in markdown fields
- **Description preview** — expandable description snippets in list view
- **Async worker system** — background job processing with NATS JetStream
- **Email registration** — self-service sign-up with email verification, configurable in auth settings
- **OAuth providers** — GitHub and Microsoft OAuth sign-in alongside existing Discord and Google
- **OAuth provider ordering** — configure the display order of sign-in providers
- **Welcome onboarding** — carousel walkthrough for new users

## Improvements

- **Data export/import** replaced with `pg_dump`/`pg_restore` + `mc mirror` for reliability
- **Admin workflow management** in system settings
- **Viewer role enforcement** — read-only access, prevented from assignment
- **API key management** with scoped permissions in user preferences
- **Activity diffs** — full multiline diffs with line-level and word-level highlighting on click
- **Resizable table columns** with persistent widths per user
- **Assignee filter** shows project members with search
- **Status filter** quick-select links for Open and Closed
- **Multi-value milestone filter** with back-to-list persistence
- **Milestone cards** show time estimate/spent totals
- **Work item form** requires type selection before filling fields
- **Tooltips, click-to-copy, and visibility info** on work item detail page
- **Workflow status translations** in settings pages
- **Responsive card view** for screens under 1024px (tablets in portrait, phones)
- **E2E test infrastructure** — fully containerized Playwright tests with `make test-e2e`
- **i18n test** to detect untranslated keys automatically

## Bug Fixes

- Fix milestone progress counter using `resolved_at` instead of workflow category lookup
- Preserve original URL across login redirect
- Fix project member list overflow on mobile
- Fix OAuth toggle enabled without provider configuration
- Fix keyboard shortcut `g then i` navigating to wrong route
- Fix paste/drop image upload in comment edit mode
- Fix SLA color feedback when paused
- Auto-reset status when changing work item type across workflows
- Fix new work items defaulting to Backlog instead of Open
- Fix filter state persistence across navigation and reload
- Fix saved search dropdown causing filter row layout shift
- Fix milestone dropdown missing in mobile properties modal
- Fix relations form layout cropping on mobile
- Fix various mobile layout issues (comments, time entries, cards, sidebar, menus)

---

# v0.1.0

First public release of Taskwondo, a self-hosted project management and issue tracking tool.

## Core Platform

- **REST API** built in Go with JWT authentication and API key support
- **React frontend** with Tailwind CSS, served via Nginx
- **PostgreSQL** for data storage with auto-running migrations
- **MinIO/S3** for file attachment storage
- **Docker Compose** deployment with a single `install.sh` setup script
- **Manual installation** option with standalone binary and static web bundle

## Project Management

- Create multiple projects with unique keys (e.g. `PROJ`)
- Role-based membership: owner, admin, member
- Per-project workflows with customizable statuses and transitions
- Per-type workflow mappings (different workflows for tasks, bugs, etc.)
- Milestones with progress tracking and due dates
- Queues for organizing incoming work (support, alerts, feedback)

## Work Items

- Tasks, bugs, stories, and epics with sequential per-project numbering (PROJ-1, PROJ-2, ...)
- Priority levels, complexity estimation, labels, assignees, and due dates
- Full-text search across titles and descriptions
- Cursor-based pagination with filtering by type, priority, status, assignee, and milestone
- Sortable columns in list view
- Kanban board view with drag-and-drop status changes
- Comments with markdown support, edit history, and paste-to-upload
- Work item relations (blocks, relates to, duplicates) with cross-project support
- Activity timeline with field change diffs
- File attachments with preview modal and inline images
- Copy work item summary as markdown
- SLA tracking with business hours, timezone support, and deadline column

## Authentication & Security

- Email/password login with bcrypt hashing
- Discord OAuth integration (optional)
- JWT tokens with configurable expiry and refresh
- API keys (`twk_` prefix) with SHA-256 hashed storage
- Rate limiting on auth endpoints
- Admin-only workflow and system mutations

## User Experience

- Dark theme with system, light, and dark modes
- Configurable font size (small, normal, large)
- Collapsible sidebar with keyboard shortcuts (`?` for help modal)
- Project switcher modal with arrow key navigation
- Inline editing for titles, descriptions, and metadata fields
- Comment draft preservation across tab switches
- `@` mention links to projects and work items
- Responsive mobile layout with touch-friendly controls

## Internationalization

9 languages: English, Portuguese, Spanish, French, German, Japanese, Chinese, Korean, and Arabic (with RTL support).

## Administration

- System settings with configurable brand name
- User management: create users, set roles, temporary passwords
- Data export/import for backup and restore
