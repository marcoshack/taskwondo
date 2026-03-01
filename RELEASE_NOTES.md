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
