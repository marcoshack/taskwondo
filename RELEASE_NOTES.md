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
