# Access Control

TrackForge uses a **two-tier RBAC system** (global + project-level).

## 1. Global Roles

Two system-wide roles stored on the `User.GlobalRole` field:

| Role | Effect |
|------|--------|
| `admin` | **Bypasses all project-level checks** — full access everywhere |
| `user` | Access governed entirely by project membership |

An admin user is seeded on startup via `SeedAdminUser()`.

## 2. Project Roles

Four project-scoped roles stored in the `project_members` table:

| Role | Can do |
|------|--------|
| `owner` | Everything — delete project, manage all members including other owners |
| `admin` | Manage members (except owners), update project settings |
| `member` | Create/edit/delete work items, comments, relations, attachments |
| `viewer` | Read-only access to project data |

The project creator is automatically assigned `owner`. The system enforces **last-owner protection** — the sole remaining owner can't be removed or demoted.

## 3. Authorization Helpers

Two reusable methods drive all authorization:

- **`requireMembership()`** — user must be a project member OR global admin. Used for read operations.
- **`requireRole(allowedRoles...)`** — user must have one of the specified roles OR be global admin. Used for write operations.

Both return `ErrNotFound` (404) instead of `ErrForbidden` (403) for non-members — this **prevents leaking project existence** to unauthorized users.

## 4. Per-Resource Permissions

| Resource | Create | Read | Update | Delete |
|----------|--------|------|--------|--------|
| **Project** | Any user | member+ | owner/admin | owner only |
| **Work Items** | member+ | viewer+ | member+ | member+ |
| **Comments** | member+ | viewer+ | author OR owner/admin | author OR owner/admin |
| **Attachments** | member+ | viewer+ | — | uploader OR owner/admin |
| **Relations** | member+ | viewer+ | — | member+ |
| **Members** | owner/admin | member+ | owner/admin | owner/admin |

Comments and attachments use an **author/uploader override** — you can always manage your own content without needing an elevated role.

## 5. Authentication

Two auth methods, both via `Authorization: Bearer <token>`:

- **JWT** — issued on login, contains `UserID`, `Email`, `GlobalRole` in claims (HMAC-SHA256)
- **API Keys** — prefixed `tfk_`, SHA-256 hashed for storage, supports expiration and `LastUsedAt` tracking

The middleware detects which type based on the `tfk_` prefix, validates it, and injects `AuthInfo` into the request context. Disabled accounts (`IsActive = false`) are rejected at both JWT-login and API-key-validation time.
