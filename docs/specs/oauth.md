---
type: Design Decision
title: OAuth authorization server
description: An OAuth 2.1 authorization server in the Taskwondo API — authorization code + PKCE for pre-registered first-party public clients, rotating refresh tokens grouped into per-device sessions revocable under Connected apps, and every token bound to one resource from day one — with the MCP server (stdio and the Claude Desktop bundle) as its first client, replacing the permanent twk_ key minted on every browser login, while every twk_ key, the legacy CLI consent page and every released MCP build keep working.
tags: [auth, oauth, mcp, api-keys, clients, security]
status: draft
sources:
  - id: rfc6749
    resource: https://www.rfc-editor.org/rfc/rfc6749
    title: The OAuth 2.0 Authorization Framework
  - id: oauth21
    resource: https://datatracker.ietf.org/doc/draft-ietf-oauth-v2-1/
    title: The OAuth 2.1 Authorization Framework (draft)
  - id: rfc7636
    resource: https://www.rfc-editor.org/rfc/rfc7636
    title: Proof Key for Code Exchange (PKCE)
  - id: rfc8252
    resource: https://www.rfc-editor.org/rfc/rfc8252
    title: OAuth 2.0 for Native Apps
  - id: rfc8707
    resource: https://www.rfc-editor.org/rfc/rfc8707
    title: Resource Indicators for OAuth 2.0
  - id: rfc9728
    resource: https://www.rfc-editor.org/rfc/rfc9728
    title: OAuth 2.0 Protected Resource Metadata
  - id: rfc7009
    resource: https://www.rfc-editor.org/rfc/rfc7009
    title: OAuth 2.0 Token Revocation
  - id: rfc8414
    resource: https://www.rfc-editor.org/rfc/rfc8414
    title: OAuth 2.0 Authorization Server Metadata
  - id: rfc9207
    resource: https://www.rfc-editor.org/rfc/rfc9207
    title: OAuth 2.0 Authorization Server Issuer Identification
  - id: rfc8693
    resource: https://www.rfc-editor.org/rfc/rfc8693
    title: OAuth 2.0 Token Exchange (used by the remote MCP server, see remote_mcp.md)
  - id: mcp-auth
    resource: https://modelcontextprotocol.io/specification/2025-11-25/basic/authorization
    title: MCP authorization (2025-11-25), including the token passthrough prohibition
---

# OAuth authorization server

**The ask.** Settled on 2026-11-08, with no work item yet. One problem today, one tomorrow, one design:

1. **The MCP server piles up immortal keys.** `taskwondo-mcp` — run by Claude Code over stdio and shipped to Claude Desktop as the `mcpb` bundle — signs in through the CLI consent page (`/auth/cli/authorize`), which mints a fresh `twk_` key named `CLI: claude-code` with no expiry and full write permission on every sign-in. The `logout` tool deletes `~/.config/taskwondo/credentials.json` and leaves the key valid on the server, so every sign-out and sign-in adds one more live, permanent credential the user never sees again unless they audit Preferences > Authentication > API Keys.
2. **A remote MCP server would need a token that is not a full API credential.** Taskwondo has no remote MCP server or third-party connectors yet. When it gets one, the MCP authorization spec forbids "token passthrough": the token a connector presents to the MCP server must not be accepted by the REST API. That is only cheap to get right if every token records its audience from the first one ever issued.

The fix: the API gains an OAuth 2.1 authorization server for pre-registered first-party public clients; `taskwondo-mcp` obtains tokens through the authorization-code flow for native apps (RFC 8252) with PKCE; tokens are short-lived and refresh-rotating, grouped into per-device sessions the user can see and revoke under Preferences > Authentication; and every token is bound to exactly one resource (RFC 8707), which the API enforces from the start. `twk_` keys, system keys (`twks_`), JWT sessions and the CLI consent page are untouched.

**Compatibility is a hard requirement.** Taskwondo is self-hosted, so server and client versions drift independently and indefinitely:

- Every released `taskwondo-mcp` build and `mcpb` bundle keeps working against a server with this change, with no action from its user (the CLI consent page stays served and unchanged).
- A new `taskwondo-mcp` build keeps working against a server **without** this change, or with the authorization server turned off, or behind a reverse proxy that does not forward the new well-known paths: it detects that and uses the legacy flow.
- Every `twk_` key in a credentials file, a `TASKWONDO_API_KEY` environment variable or an `mcpb` `api_key` setting keeps working.

## What exists

As it stood on 2026-11-08 (at `e820843`), before this work; kept as the record the design starts from.

- **No OAuth authorization server.** `api/internal/service/oauth*.go` are social-login *clients* (Google, GitHub, Microsoft, Discord); nothing issues OAuth tokens.
- **Bearer credentials** (`api/internal/middleware/auth.go`): `twks_` system keys (resource-scoped, `/metrics` and work items), `twk_` user keys (`read` or `write`, optional `expires_at`, SHA-256 hashed in `api_keys`), anything else parsed as a web JWT. `AuthInfo` (`api/internal/model/auth.go`) carries `UserID`, `Email`, `GlobalRole`, `Permissions`, `KeyType` (`""`, `user`, `system`), `KeyID`, `KeyName`. `ValidateAPIKey` re-reads the user and refuses an inactive one. Every `401` is the JSON envelope `{"error":{"code":"UNAUTHORIZED",…}}` with no `WWW-Authenticate` header.
- **The CLI consent page** (`web/src/pages/CliAuthorizePage.tsx`, route `/auth/cli/authorize`) takes `callback_port` and `client_name`, sends a signed-out user to `/login?next=…`, and on Allow calls `POST /api/v1/user/api-keys` with the browser's JWT to mint `CLI: <client_name>`, then navigates to `http://localhost:<port>/callback?key=<key>&theme=…&brand=…`. It carries no `state`, so any local process or web page that reaches the listener first can complete a login with a key of its choosing (login CSRF); the key itself also travels in a URL.
- **The MCP server** (`mcp/`, one Go binary, stdio only). `ResolveAuth` (`mcp/auth.go`) takes `TASKWONDO_URL`, `TASKWONDO_API_KEY` and `TASKWONDO_NAMESPACE` from the environment, else from `<TASKWONDO_CONFIG_DIR or ~/.config/taskwondo>/credentials.json` (`url`, `api_key`, `namespace`). `BrowserLogin` listens on `127.0.0.1:0`, opens the consent page with `client_name=claude-code`, waits 5 minutes, and **rewrites the whole file with `url` and `api_key` only**, dropping `namespace`. `logout` **deletes the file**. The callback page escapes but reflects the `error`, `theme` and `brand` query parameters. Several Claude Code sessions routinely run several `taskwondo-mcp` processes against one credentials file at once.
- **The `mcpb` bundle** (`mcpb/manifest.json`) passes `TASKWONDO_URL` and `TASKWONDO_API_KEY` from its user config; a blank `api_key` means "use browser login". It ships for `darwin` and `win32`.
- **No server-sent events or websockets**, so there is no stream whose lifetime a token revocation must bound.
- **The web app** reaches the API through nginx's `location /api/` only (`docker/nginx.conf`); every other path is the SPA. In the e2e stack the API serves the frontend itself. Both send `X-Frame-Options: DENY` and `frame-ancestors 'none'`.
- **Rate limiting**: `middleware.RateLimit` per IP, used as `authLimiter` (`AUTH_RATE_LIMIT`, `AUTH_RATE_BURST`) on the unauthenticated `/auth/*` routes.
- **Cleanup**: the worker's `token.cleanup` task (`TOKEN_CLEANUP_INTERVAL`) purges expired email-verification and password-reset tokens.
- **Preferences > Authentication** (`web/src/pages/AuthenticationPage.tsx`) has three tabs: Password, Connected accounts (social logins), API Keys.

## Decisions

Each is settled for this spec; the five that were open are recorded under "Settled questions".

1. **An OAuth 2.1 authorization server in the API**, for public clients with PKCE S256 only. The issuer is the canonical `BASE_URL`. It is on by default and can be turned off with `OAUTH_SERVER_ENABLED=false` (settled question 1).
2. **Pre-registered clients only.** One seeded first-party client, `taskwondo-mcp`, used by both the stdio server and the `mcpb` bundle (they are the same binary). No dynamic client registration in this spec; [the remote MCP server](remote_mcp.md) adds it for connectors.
3. **One resource now, recorded everywhere.** The API resource is `<issuer>/api`. Every grant, code and token row records its resource as a canonical URL, and the API accepts an OAuth token only when its resource is the API resource. With one resource the check cannot fail today; it is what makes a future MCP resource safe by construction.
4. **Sessions under a grant.** One live grant per `(user, client)`; one session per authorization-code redemption, i.e. one per device sign-in. Refresh rotation stays inside a session; logout revokes only its session; a refresh-token replay revokes only its session.
5. **OAuth tokens are user credentials with a deny-list.** A `twko_` token acts as its user with `write`, like a `CLI:` key does today, minus the routes that create credentials, manage OAuth itself or administer the instance (see "What an OAuth token may do"). One scope, `taskwondo`.
6. **The MCP server keeps its own OAuth token file** beside `credentials.json`, never writes `api_key` again, never deletes `credentials.json`, and serialises refreshes across processes with a file lock.
7. **API keys stay**, for what OAuth does not fit (see "When a user API key is still the right credential"). The CLI consent page stays served and unchanged; whether it is ever retired is decided later (settled question 5).

# The design

## Endpoints

| Endpoint | Served by | Auth | Purpose |
|---|---|---|---|
| `GET /.well-known/oauth-authorization-server` | API (root router, like `/healthz`) | none | RFC 8414 metadata |
| `GET /.well-known/oauth-protected-resource/api` | API | none | RFC 9728 metadata for the API resource |
| `GET /oauth/authorize` | SPA (`OAuthAuthorizePage`) | web session | consent page |
| `GET /api/v1/oauth/authorize/client` | API | JWT only | consent info for the page |
| `POST /api/v1/oauth/authorize` | API | JWT only | the user's Allow or Deny; returns the redirect |
| `POST /api/v1/oauth/token` | API | client id in form (`none`) | `authorization_code`, `refresh_token` |
| `POST /api/v1/oauth/revoke` | API | none | RFC 7009 |
| `GET /api/v1/oauth/grants` | API | JWT or `twk_` | Connected apps list |
| `DELETE /api/v1/oauth/grants/{grantId}` | API | JWT only | revoke an app |
| `DELETE /api/v1/oauth/grants/{grantId}/sessions/{sessionId}` | API | JWT only | sign one device out |

- **Routing.** nginx gains `location ^~ /.well-known/oauth- { proxy_pass http://api:8080; … }` with the same proxy headers as `/api/`, placed before `location /`; without it the SPA answers and discovery falls back to the legacy flow (by design, not by accident). `web/vite.config.ts` proxies `/.well-known/oauth-` to `:8080`. In the e2e stack the API's root router must register the two well-known routes ahead of its SPA fallback. `MANUAL_INSTALL.md` tells operators with their own reverse proxy to forward `/.well-known/oauth-*` to the API.
- **"JWT only"** means `AuthInfo.KeyType == ""`. An API key or an OAuth token calling these is `403` (`oauth_requires_web_session`). This is what stops a leaked `twk_` key or `twko_` token from minting new OAuth grants or revoking other devices.
- **With the server off** (`OAUTH_SERVER_ENABLED=false`): every row above answers `404` except the grants routes, which keep listing and revoking (so a user can still clean up after the operator turns it off); the token endpoint and validation refuse every `twko_` token; the consent page shows "Browser sign-in for apps is turned off on this server."
- **Protocol responses use OAuth's formats, not the API envelope.** The token and revoke endpoints answer RFC 6749 section 5.2 errors (`{"error": "invalid_grant", "error_description": "…"}`) with `Cache-Control: no-store`, because that is what OAuth client libraries parse. The consent and grants routes use the normal envelope and `model.NewKeyedError` keys.

## Authorization-server metadata

```json
{
  "issuer": "https://taskwondo.example.com",
  "authorization_endpoint": "https://taskwondo.example.com/oauth/authorize",
  "token_endpoint": "https://taskwondo.example.com/api/v1/oauth/token",
  "revocation_endpoint": "https://taskwondo.example.com/api/v1/oauth/revoke",
  "response_types_supported": ["code"],
  "grant_types_supported": ["authorization_code", "refresh_token"],
  "code_challenge_methods_supported": ["S256"],
  "token_endpoint_auth_methods_supported": ["none"],
  "revocation_endpoint_auth_methods_supported": ["none"],
  "scopes_supported": ["taskwondo"],
  "authorization_response_iss_parameter_supported": true
}
```

No `registration_endpoint`, `client_secret_basic` or token-exchange grant here: [the remote MCP server](remote_mcp.md) adds them to this document while it is enabled.

**Canonicalisation** (issuer, resources, the client's base URL): lowercase scheme and host, default port dropped, no trailing slash, no query or fragment. The API canonicalises `BASE_URL` once at startup and logs `oauth: authorization server on` with the field `issuer`, or `oauth: authorization server off`. A `BASE_URL` whose scheme is `http` on a non-loopback host logs a `warn` (`oauth: issuer is not https`), and the server still starts: plenty of LAN installs run plain HTTP.

## Resources and audience

| Resource | Identifier | Who may hold a token for it | Accepted by |
|---|---|---|---|
| API | `<issuer>/api` | first-party native clients (`taskwondo-mcp`) | the API |
| MCP ([remote MCP](remote_mcp.md)) | `<issuer>/mcp` | connectors, through dynamic registration | the remote MCP server only, which exchanges it for an API token; the API refuses it |

- **Clients name the resource they want**; `allowed_resources` holds the symbolic names (`api` today), resolved to URLs from the issuer at request time so a deployment's host never lives in a migration. A request naming a resource the client may not request is `invalid_target` (`resource is not one this client may request`); an absent `resource` means the client's default (the first entry of `allowed_resources`). On a code or refresh request a present `resource` must equal the one the grant is bound to.
- **Audience enforcement is always on.** `ValidateOAuthAccessToken` refuses a token whose resource is not the API resource with `401` and `error_description="token not issued for this resource"`, logged at `warn` (`oauth token refused: not issued for this resource`, fields `clientID`, `clientName`, `grantID`) at most once per grant per minute. There is no flag: no token without a resource can exist.
- **API protected-resource metadata** at `/.well-known/oauth-protected-resource/api`: `{"resource": "<issuer>/api", "authorization_servers": ["<issuer>"], "scopes_supported": ["taskwondo"], "bearer_methods_supported": ["header"], "resource_name": "Taskwondo API"}`. **Its presence is what tells a client the server supports this spec** (see "Sign-in", step 1).
- **The challenge.** Every `401` from `middleware.Auth` gains `WWW-Authenticate: Bearer resource_metadata="<issuer>/.well-known/oauth-protected-resource/api"`, with `, error="invalid_token"` appended when a bearer was presented (plus the `error_description` above for an audience refusal). The JSON body is unchanged, so the web app and released MCP builds see nothing new. With the server off the header is a bare `Bearer` (plus `error="invalid_token"` when a bearer was sent), never a metadata URL that would 404. Strict RFC 9728 section 3.3 validation of that header fails for sub-path requests, so first-party clients take the resource from the metadata document, never from the header.

## Schema

One migration, `000063_create_oauth_server`, new tables only (nothing existing is altered, so a previous API image runs against it unchanged):

```sql
CREATE TABLE oauth_clients (
    id                  TEXT PRIMARY KEY,           -- fixed readable id for first-party rows
    client_name         TEXT NOT NULL,
    client_type         TEXT NOT NULL DEFAULT 'public' CHECK (client_type IN ('public', 'confidential')),
    first_party         BOOLEAN NOT NULL DEFAULT false,
    redirect_uris       TEXT[] NOT NULL DEFAULT '{}',
    allowed_resources   TEXT[] NOT NULL,            -- symbolic: 'api' (later 'mcp')
    allowed_scopes      TEXT[] NOT NULL,
    allowed_grant_types TEXT[] NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE oauth_grants (
    id           UUID PRIMARY KEY,
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    client_id    TEXT NOT NULL REFERENCES oauth_clients(id) ON DELETE CASCADE,
    resource     TEXT NOT NULL,                     -- canonical URL
    scope        TEXT NOT NULL,                     -- last consent; display only
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_used_at TIMESTAMPTZ,
    revoked_at   TIMESTAMPTZ
);
CREATE UNIQUE INDEX idx_oauth_grants_live ON oauth_grants(user_id, client_id) WHERE revoked_at IS NULL;

CREATE TABLE oauth_sessions (
    id           UUID PRIMARY KEY,
    grant_id     UUID NOT NULL REFERENCES oauth_grants(id) ON DELETE CASCADE,
    scope        TEXT NOT NULL,
    user_agent   TEXT NOT NULL DEFAULT '',          -- token request's User-Agent, 200 chars max, display only
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_used_at TIMESTAMPTZ,
    revoked_at   TIMESTAMPTZ
);
CREATE INDEX idx_oauth_sessions_grant ON oauth_sessions(grant_id);

CREATE TABLE oauth_authorization_codes (
    code_hash      TEXT PRIMARY KEY,                -- SHA-256 hex
    grant_id       UUID NOT NULL REFERENCES oauth_grants(id) ON DELETE CASCADE,
    redirect_uri   TEXT NOT NULL,                   -- exactly as sent
    code_challenge TEXT NOT NULL,
    resource       TEXT NOT NULL,
    scope          TEXT NOT NULL,                   -- consented scope
    expires_at     TIMESTAMPTZ NOT NULL,
    redeemed_at    TIMESTAMPTZ,
    session_id     UUID REFERENCES oauth_sessions(id) ON DELETE SET NULL
);

CREATE TABLE oauth_tokens (
    token_hash TEXT PRIMARY KEY,                    -- SHA-256 hex
    kind       TEXT NOT NULL CHECK (kind IN ('access', 'refresh')),
    session_id UUID NOT NULL REFERENCES oauth_sessions(id) ON DELETE CASCADE,
    resource   TEXT NOT NULL,
    scope      TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    rotated_at TIMESTAMPTZ,                         -- refresh: set when redeemed
    revoked_at TIMESTAMPTZ
);
CREATE INDEX idx_oauth_tokens_session ON oauth_tokens(session_id);

INSERT INTO oauth_clients (id, client_name, client_type, first_party, redirect_uris,
                           allowed_resources, allowed_scopes, allowed_grant_types)
VALUES ('taskwondo-mcp', 'Taskwondo MCP', 'public', true,
        '{http://127.0.0.1/callback,http://[::1]/callback}',
        '{api}', '{taskwondo}', '{authorization_code,refresh_token}')
ON CONFLICT (id) DO NOTHING;
```

- **IDs**: grants and sessions are UUIDv4 (not time-ordered entities in the AGENTS.md sense; their `created_at` is what the UI sorts by).
- **Secrets are stored hashed.** Codes, access and refresh tokens are 32 random bytes; only their SHA-256 is stored, as for `twk_` keys. Formats: code `<64 hex>`, access `twko_<64 hex>`, refresh `twkor_<64 hex>`.
- **The middleware dispatch order is explicit**: `twks_`, then `twkor_` (a refresh token presented as a bearer is `401`, never looked up as anything else), then `twko_`, then `twk_`, then JWT. `strings.HasPrefix("twko_…", "twk_")` is already false, but the order is written so a later prefix can't silently shadow another.
- **Expand-only later.** [The remote MCP server](remote_mcp.md) adds a seeded row and nullable columns (`actor_client_id`, `parent_token_hash` for exchanged tokens); nothing here changes shape for it.

## Authorization endpoint and consent page

**Request** (`GET <issuer>/oauth/authorize`): `response_type=code`, `client_id`, `redirect_uri`, `state`, `code_challenge`, `code_challenge_method=S256`, `scope` (optional), `resource` (optional).

**Validation order**, as RFC 6749 section 4.1.2.1 requires:

1. Unknown `client_id`, or a `redirect_uri` that does not match → the page shows an error and **never redirects** (`oauth_invalid_client`, `oauth_invalid_redirect_uri`).
2. Everything else redirects back with `error`, `state` and `iss`: `response_type` other than `code` → `unsupported_response_type`; missing `code_challenge` or a method other than `S256` → `invalid_request`; a missing `state` → `invalid_request` (PKCE alone would do, but every client this spec defines sends one, and requiring it makes the legacy flow's login-CSRF impossible here by construction); a scope outside `allowed_scopes` → `invalid_scope`; a bad `resource` → `invalid_target`; a grant type the client may not use → `unauthorized_client`.

**Redirect matching** (RFC 8252 sections 7.3 and 8.3): an `http` redirect whose host is the literal `127.0.0.1` or `[::1]` matches a registered one with any port, the path compared exactly; `localhost` is not accepted. Every other redirect is compared byte for byte.

**The page** (`web/src/pages/OAuthAuthorizePage.tsx`, route `/oauth/authorize`):

- Signed out → `/login?next=<the full authorize URL>`, as `CliAuthorizePage` does today. The `next` round trip must survive password, registration with email verification, and every social-login provider (social login binds its own `state` to the browser since `6f4fc54`; the `next` has to ride alongside it).
- It renders only from `GET /api/v1/oauth/authorize/client?client_id&redirect_uri&scope&resource&response_type&code_challenge_method`, which runs validation step 1 and 2 without side effects and returns `{"client_id", "client_name", "first_party", "scopes": ["taskwondo"], "resource", "redirect_host"}`, or the step-1 error, or `{"redirect_error": "invalid_scope"}` for a step-2 error (the page then redirects without showing Allow).
- It shows the client name, a "Taskwondo app" label for a first-party client, "Signed in as <name> (<email>)", what the app may do ("Read and change everything your account can, except API keys, connected apps and system administration"), and, for a loopback redirect, "This app runs on your computer. Only allow it if you just started signing in." Allow and Deny buttons.
- **Consent is always asked**; no client is approved silently, first-party included (RFC 8252 section 8.6). An attacker who can open a URL on the user's machine still cannot mint a session without a click.
- It is not frameable (the existing `X-Frame-Options: DENY` and `frame-ancestors 'none'` already cover every SPA route; a test asserts the headers on `/oauth/authorize`).
- All strings in `en.json` and the eight other locales, under `oauthAuthorize.*`. Every API error through `getLocalizedError`.

**Decision** (`POST /api/v1/oauth/authorize`, JWT only): body is the request parameters plus `"decision": "allow" | "deny"`. The server re-runs the full validation (never trusting the page). Deny → `{"redirect_to": "<redirect_uri>?error=access_denied&state=…&iss=…"}`. Allow → reuse the user's live grant for the client or create one (the partial unique index decides a race; the loser re-reads), set its `scope` to the consented one, insert a code (60 s), and return `{"redirect_to": "<redirect_uri>?code=…&state=…&iss=<issuer>"}`. The page then sets `window.location.href`. The code never appears in an API log line: the request logger logs the path only for this route's response.

## Token endpoint

`POST /api/v1/oauth/token`, `application/x-www-form-urlencoded`. Public clients authenticate with `client_id` in the form (`none`). A request carrying an `Authorization` header is `invalid_client`; [the remote MCP server](remote_mcp.md) replaces this rule with `client_secret_basic` for its one confidential client. `allowed_grant_types` is enforced on every grant (`unauthorized_client`); an unknown grant type is `unsupported_grant_type`.

**`grant_type=authorization_code`**: `code`, `client_id`, `redirect_uri` (byte-identical to the authorize request), `code_verifier`, `resource` (optional; must equal the code's).

- In one transaction: lock the code row; refuse if expired, redeemed, for another client, wrong redirect, or the PKCE check fails (`invalid_grant`); refuse if the grant is revoked or the user inactive (`invalid_grant`). Create a session with the code's scope and the request's `User-Agent`, record it on the code, mark the code redeemed, issue the first pair.
- **Code reuse** (`redeemed_at` already set) revokes the session the first redemption created, and its tokens, then answers `invalid_grant`. The code row is re-read after the lock clears, so a concurrent loser sees the winner's session. A failed PKCE or redirect check burns the code (sets `redeemed_at` with no session), so a guessed verifier cannot be retried.

**`grant_type=refresh_token`**: `refresh_token`, `client_id`, `resource` (optional; must equal the session's).

- Lock the refresh row. Live and unrotated → set `rotated_at`, issue a new pair in the same session, slide the session's expiry, update `last_used_at` on session and grant.
- **Replay** of a rotated refresh token (OAuth 2.1 section 4.3.1) revokes its session and every token in it, then `invalid_grant`. Only that device is signed out; strict, no grace window — the client's lock prevents the benign race.
- Expired, revoked, other client's, revoked session or grant, inactive user → `invalid_grant`.

**Response** (`200`, `Cache-Control: no-store`, `Pragma: no-cache`): `{"access_token": "twko_…", "token_type": "Bearer", "expires_in": 3600, "refresh_token": "twkor_…", "scope": "taskwondo"}`.

**Lifetimes**: code 60 s; access 1 h; refresh 30 days, sliding (each rotation issues a refresh token valid 30 days from now). A session idle for 30 days is dead.

**Rate limit**: the token and revoke endpoints (and registration, when [the remote MCP server](remote_mcp.md) is on) share a per-IP limiter, `OAUTH_RATE_LIMIT` (default 120/min) with `OAUTH_RATE_BURST` (default 30), separate from `authLimiter` so a desk full of Claude Code sessions refreshing hourly cannot lock its users out of the login form, and vice versa. A refusal is `429` with `Retry-After` and the body `{"error": "temporarily_unavailable"}`: RFC 6749 has no rate-limit error code for the token endpoint, so this borrows the retryable one from section 4.1.2.1 (`slow_down` belongs to the device flow and means something else).

## Revocation and sessions

`POST /api/v1/oauth/revoke` (RFC 7009): `token`, optional `token_type_hint`, optional `client_id`. Always `200` for a well-formed request, whether or not the token existed (section 2.2).

- **A refresh token revokes its session** and every token in it. RFC 7009 section 2.1 lets a server revoke related tokens on a refresh revoke; the session is that unit here.
- **An access token revokes only itself.**
- **Revoking a grant's last live session revokes the grant** in the same transaction, so a signed-out app leaves Connected apps. **Revoking the grant revokes every session.**
- **Validation of a `twko_` token** joins token → session → grant → client → user and refuses on any `revoked_at`, an expired token, a resource that is not the API resource, or an inactive user. It updates `last_used_at` on the session and grant at most once a minute per session (an in-memory throttle), not on every request.
- **Deleting a user** cascades. **Deactivating a user** (`is_active = false`) refuses their tokens at the next request, like `twk_` keys; their grants stay so reactivation needs no re-consent (settled question 4).

## The principal

`AuthInfo` for a `twko_` token: `UserID`, `Email`, `GlobalRole` (re-read per request, so a demotion takes effect at once), `KeyType: "oauth"`, `KeyID: <grant id>`, `KeyName: <client name>`, `Permissions: ["write"]`, plus new fields `ClientID`, `SessionID`, `Resource`, `Scopes`, `ExpiresAt`. Every existing consumer of `KeyID`/`KeyName` keeps working. The request log line gains `client_id` for OAuth principals, through a recorder `Auth` fills in (`Logging` runs outside `Auth`), so a log line says which app acted.

## What an OAuth token may do

An OAuth token is barred, `403` with code `FORBIDDEN` and the key `oauth_route_barred`, from:

| Route | Why |
|---|---|
| `POST`, `PATCH`, `DELETE /api/v1/user/api-keys…` | would mint a permanent credential from a revocable one |
| `POST /api/v1/auth/change-password`, `POST /api/v1/auth/refresh` | password and web-session credentials |
| `/api/v1/oauth/authorize…`, `/api/v1/oauth/grants…` (writes) | consent and Connected apps are web-session only (above) |
| `DELETE /api/v1/user/connected-accounts/…` | unlinking a social login changes how the account signs in |
| `/api/v1/admin…`, every method | instance administration; no MCP tool calls it (`Client.ListWorkflows`, the one caller of `/admin/workflows`, is dead code and is removed with this work) |
| `/metrics` | operator surface |

Everything else a `write` user key can do, an OAuth token can do, under the same RBAC (namespaces, projects, items, inbox, portal). **The bar is a deny-list on purpose**: the MCP server reaches most of the user surface and its tool list grows often; an allowlist would break a new tool silently. Each barred row has a handler test sending the request with a `twko_` token. The check runs in `middleware.Auth` on the matched path after chi's routing context is known, and any OAuth request whose `RawPath` is non-empty under a barred prefix is refused too, so a percent-encoded path cannot reach a barred handler.

## Connected apps

- **`GET /api/v1/oauth/grants`** → `{"data": [{"id", "client_id", "client_name", "first_party", "scope", "created_at", "last_used_at", "sessions": [{"id", "created_at", "last_used_at", "user_agent"}]}]}`, live grants and live sessions only, newest first. A consented, never-redeemed grant lists no sessions.
- **`DELETE /api/v1/oauth/grants/{grantId}`** → `204`; `404` when not the caller's or already revoked.
- **`DELETE /api/v1/oauth/grants/{grantId}/sessions/{sessionId}`** → `204`; `404` likewise. The last session's revocation revokes the grant.
- **The web app**: a fourth tab under Preferences > Authentication, "Connected apps", between Connected accounts and API Keys. Each grant shows its name, the "Taskwondo app" label, created and last used; under it its sessions, the user agent raw and muted ("Unknown device" when empty), created and last used. "Revoke" on the grant; "Sign out" on a session only when the grant has two or more (one would equal Revoke). Both confirm in a `<Modal>`; success is the inline `<Check>` pattern. Empty state: "No apps are signed in. Apps like the Taskwondo MCP server appear here after you sign in to them with your browser." If the tabs are deep-linkable, the new tab gets the same treatment, and the command palette reaches it through `sidebarNav.ts` like its siblings. New hook `useOAuthGrants` in `web/src/hooks/`, client in `web/src/api/oauth.ts`. Nine locales.
- **The API Keys tab** gains one line: "For scripts and tools that can't sign in through a browser. Taskwondo apps sign in for you." Nine locales.

## Configuration

| Variable | Default | Meaning |
|---|---|---|
| `OAUTH_SERVER_ENABLED` | `true` | the kill switch above |
| `OAUTH_RATE_LIMIT` | `120` | per IP per minute, token and revoke endpoints (and registration with remote MCP on) |
| `OAUTH_RATE_BURST` | `30` | burst for the above |

The issuer comes from `BASE_URL`; there is no separate variable, so a server can't advertise one host and redirect to another. `.env.template`, `MANUAL_INSTALL.md` and `README_DEV.md` document all three.

## Cleanup

The worker's `token.cleanup` task also deletes codes expired more than a day ago, tokens expired more than a day ago, and sessions and grants revoked (or dead by idleness) more than 30 days ago. No new interval.

# Client migration guide

The contract `taskwondo-mcp` implements, in new `mcp/oauth.go` plus changes to `mcp/auth.go`, `mcp/client.go` and the `login`, `logout` and `whoami` tools. It is written so any future first-party client (a CLI, a desktop app) follows the same steps with its own client id.

## Sign-in (authorization code + PKCE over loopback, RFC 8252)

1. **Discover.** Canonicalise the resolved base URL. `GET <base_url>/.well-known/oauth-protected-resource/api`. **Anything other than a `200` with a JSON body whose `resource` canonicalises to `<base_url>/api` means the server predates this spec, has the server off, or sits behind a proxy that doesn't forward the path: use the legacy CLI consent flow**, unchanged. Then `GET <base_url>/.well-known/oauth-authorization-server` and require `issuer` to be byte-identical to the canonical `<base_url>` (RFC 8414 section 3.3). **A mismatch also falls back to the legacy flow**, with a notice in the tool result ("This server's BASE_URL is <issuer>; signing in with a permanent key instead. Ask the administrator to fix BASE_URL, or set TASKWONDO_URL to it."): on a self-hosted install a mismatch is almost always a misconfigured `BASE_URL`, and the consent page would open on the wrong host (settled question 2). A present `grant_types_supported` lacking `authorization_code` also means legacy.
2. **Prepare.** `state`: 32 random bytes, base64url. `code_verifier`: 32 random bytes, base64url (43 characters). `code_challenge` = base64url(SHA-256(verifier)).
3. **Listen** on `127.0.0.1`, port `0`. Bind the literal, not `localhost`.
4. **Open the browser** at `authorization_endpoint` with `response_type=code`, `client_id=taskwondo-mcp`, `redirect_uri=http://127.0.0.1:<port>/callback`, `state`, `code_challenge`, `code_challenge_method=S256`, `scope=taskwondo`, `resource=<resource from step 1>`. If the browser can't be opened, the tool result carries the URL to visit by hand, as today.
5. **Receive** on `/callback`. **Only a request whose `state` matches exactly ends the wait**; then require `iss` to equal the discovered issuer (RFC 9207, RFC 8252 section 8.10) and read `code` or `error`. A missing or wrong `state`, or a wrong `iss`, gets `400` with static text and the listener keeps waiting (RFC 8252 section 8.9). **Every answer page is static and interpolates no query parameter** — the legacy callback's pages stop reflecting `error`, `theme` and `brand` too (they render the light theme and the "Taskwondo" brand; the `theme`/`brand` parameters are ignored). Log the path, never the query.
6. **Redeem.** `POST token_endpoint` form: `grant_type=authorization_code`, `code`, `client_id`, `redirect_uri` (byte-identical to step 4), `code_verifier`, `resource`.
7. **Verify** with `GET /api/v1/auth/me` using the access token, then store (below).
8. **Timeout**: 5 minutes, as today.

## Storage

- **Token file**: `<TASKWONDO_CONFIG_DIR or ~/.config/taskwondo>/oauth/taskwondo-mcp.json`, directory `0700`, file `0600`, write-then-rename, opened without following a symlink (`O_NOFOLLOW` on Unix; on Windows, refuse a reparse point).
- **Fields**: `url` (canonical base URL), `issuer`, `client_id`, `resource`, `scope`, `access_token`, `access_expires_at` (RFC 3339), `refresh_token`, `signed_out` (bool). Unknown keys preserved.
- **A token file whose `url` differs from the canonical resolved base URL** (after `TASKWONDO_URL`) is not used and its tokens are never sent.
- **`credentials.json` stays the home of `url` and `namespace`.** The MCP server reads and writes it read-modify-write as a JSON object of raw values, preserving keys it does not own, write-then-rename. **It never writes `api_key`** except when a legacy-flow sign-in returns one, never deletes the file, and stops dropping `namespace` on login. If the file is missing (an old build's logout deleted it), `url` is recovered from the token file.
- **Signing in again** best-effort revokes the token file's previous refresh token first, so a re-login doesn't leave a dangling session.
- **Tokens never reach a log or a tool result**: the types holding them format as `twko_…` / `twkor_…`.
- **User-Agent** on every request: `taskwondo-mcp/<version> (<os>)`, the version from the build (`-ldflags`), replacing the hard-coded `0.1.0` in `main.go`. This is what Connected apps shows per session.

## Credential precedence

Highest first:

1. `TASKWONDO_API_KEY` in the environment, non-empty — the explicit unattended path, and the `mcpb` bundle's `api_key` setting; OAuth is not attempted.
2. The token file for the resolved `url`, when it holds tokens.
3. The same file with `signed_out: true` — **signed out**: the legacy `api_key` in `credentials.json` is not used.
4. `api_key` in `credentials.json`, only when there is no token file for the resolved `url` and `credentials.json`'s own `url`, canonicalised, is the resolved one (a key is never sent to another server). An upgraded install keeps working on its legacy key with no action. `whoami` adds one line while on it: "Signed in with a permanent API key. Run the login tool to switch to browser sign-in; you can then delete the old key under Preferences > Authentication > API Keys."
5. Signed out: tools answer "Not signed in. Run the login tool, or set TASKWONDO_API_KEY."

## Using and refreshing

- **Send** `Authorization: Bearer <access_token>`. The `Client` holds a token source, not a fixed string, so a long-running process keeps working across expiries.
- **Refresh** when the access token expires within 60 s, or once after a `401`: take an exclusive lock on `oauth/taskwondo-mcp.lock` (`flock` on Unix, `LockFileEx` on Windows, both via `golang.org/x/sys`; 30 s timeout), **re-read the file** — another `taskwondo-mcp` process may already have refreshed — and refresh only if the refresh token on disk is still the one this process holds. `POST token_endpoint` with `grant_type=refresh_token`, `refresh_token`, `client_id`, `resource`. Write the new pair, release the lock, retry the request once. A lock timeout fails the tool call with a retryable error and keeps the file.
- **`invalid_grant` on refresh means signed out** (session revoked from Connected apps, replay, grant revoked, 30 days idle): write the file as `signed_out: true` and answer "Your Taskwondo session ended; run the login tool to sign in again." A network error, `429` or `5xx` keeps the file and fails the call as retryable; it never signs the user out.

## Logout

1. `POST revocation_endpoint` with `token=<refresh_token>`, `client_id`, best effort, 5 s timeout. This revokes the session — this device only.
2. Rewrite the token file as `{"url": …, "client_id": …, "signed_out": true}` (a tombstone, not a delete), so precedence step 3 keeps the legacy key from being picked up again.
3. Do not touch `credentials.json`. The result is "Signed out on this device." plus, when `credentials.json` still holds an `api_key`, "An older permanent API key is still stored for this server; delete it under Preferences > Authentication > API Keys if you no longer need it."
4. A logout on the legacy path (no token file, an `api_key`) keeps today's behaviour of removing the key from the file, but now by removing only the `api_key` field, not the file.

## The `mcpb` bundle

- No new user config. A blank `api_key` already means browser login; the description becomes "Optional. Leave blank to sign in with your browser (recommended); set an API key only for machines without a browser."
- Claude Desktop starts the binary with `TASKWONDO_API_KEY` set to the empty string when the field is blank; precedence step 1 treats an empty value as unset (as `ResolveAuth` already does).
- The bundle and a Claude Code install on the same machine share the token file and the lock, so signing in through one signs in both, and they never replay each other's refresh.

# When a user API key is still the right credential

OAuth needs a person at a browser to start, and someone to come back within 30 days of idleness. Where that does not hold, a `twk_` key created under Preferences > Authentication > API Keys (or minted by the legacy CLI page) remains the supported credential, with no plan to remove it:

- **Unattended automation** — CI jobs, cron scripts, webhooks, anything with no browser and nobody to re-consent. `TASKWONDO_API_KEY` in the environment.
- **Long idle periods** — a script run monthly would find its refresh token dead.
- **A narrower or dated credential** — `read`-only keys and keys with an explicit `expires_at` exist only as API keys; OAuth grants carry `write`.
- **Third-party tools without OAuth support** — anything that takes a bearer token.
- **A machine whose browser can't reach the client's loopback** — an SSH session on a remote box. (The device authorization grant, RFC 8628, would cover this later; out of scope.)
- **Old client builds** — until their users upgrade.

# Rollout

Taskwondo ships as one release that operators upgrade on their own schedule, so there is no staged production rollout; the requirement is that every combination works:

| Server | `taskwondo-mcp` | Result |
|---|---|---|
| old | old | unchanged |
| new | old | legacy CLI flow, unchanged |
| old (or server off, or proxy not forwarding `/.well-known/oauth-*`) | new | discovery fails → legacy CLI flow |
| new | new | OAuth; an existing `api_key` keeps working until the user runs `login` |

- **Server and web** land together: the migration, the endpoints, the consent page, Connected apps, the route bar, the challenge header, nginx and Vite routes, the worker cleanup, docs. Nothing an existing client does changes.
- **The MCP server and `mcpb` bundle** land in the same release or later; they work against either server.
- **`RELEASE_NOTES.md`** says: what changed for users (browser sign-in for the MCP server now creates a revocable session listed under Connected apps instead of a permanent key; old keys keep working and can be deleted), and for operators with their own reverse proxy, that `/.well-known/oauth-*` must reach the API (otherwise clients quietly keep using the legacy flow) and that `BASE_URL` must be the URL users open.
- **Operator check** after upgrading (also in `MANUAL_INSTALL.md`), with `H` the public URL:

  | Check | Command | Expected |
  |---|---|---|
  | AS metadata | `curl -s $H/.well-known/oauth-authorization-server \| jq -r .issuer` | `$H` exactly |
  | API resource | `curl -s $H/.well-known/oauth-protected-resource/api \| jq -r .resource` | `$H/api` (HTML or an error means the proxy isn't forwarding the path) |
  | Challenge | `curl -si $H/api/v1/auth/me \| grep -ci '^www-authenticate: Bearer resource_metadata='` | `1` |
  | Startup | API log | `oauth: authorization server on` with `issuer` equal to `$H` |

- **Not in this release:** [the remote MCP server](remote_mcp.md) (with its dynamic registration and token exchange, which ships on top of this one), a read-only scope, device flow, retiring the CLI consent page.

# Remote MCP server

[The remote MCP server](remote_mcp.md) is specified separately and builds on this one. It adds the MCP resource (`<issuer>/mcp`), dynamic client registration for connectors (limited to that resource), the confidential client `taskwondo-mcp-remote`, and RFC 8693 token exchange, which the MCP container uses **from its first release** to trade every connector token for a short-lived API-resource token. There is no passthrough mode: the API keeps refusing every token not issued for the API, as this spec's audience rule already requires. The deny-list above applies unchanged to exchanged tokens.

# Security notes

- **No more immortal keys from browser sign-in.** A stolen token file is worth at most one session: the access token lives an hour; the refresh token is single-use, so whichever of the thief and the real device refreshes second presents a rotated token, and that replay kills the session for both. The user sees an unexpected sign-out and the session disappears from Connected apps.
- **Per-device revocation** from Connected apps, and logout that actually revokes.
- **Audience binding from day one**: no OAuth token exists without a resource, so a future connector token can't become an API credential through an oversight.
- **A revocable credential can't mint a permanent one**: OAuth tokens are barred from API-key writes, consent and Connected apps; and only a web session can consent, so a leaked `twk_` key can't mint OAuth grants either.
- **`state` and `iss` checks** close the login-CSRF the legacy flow has (RFC 8252 sections 8.9 and 8.10). **PKCE** (section 8.1) and the exact-path loopback match (section 8.3) stop another local process from redeeming an intercepted code. Codes are single-use, 60 s, burned on a failed check, and reuse revokes the session they created.
- **No silent consent**, first-party included (RFC 8252 section 8.6), and the consent page can't be framed.
- **Tokens and codes are stored hashed**, never logged, never in a tool result. The code is in a URL only for the one loopback redirect, which is what RFC 8252 prescribes; the legacy flow put a permanent key there.
- **The legacy flow's weaknesses remain** for old clients and old servers until the page is retired (settled question 5); new clients use it only against servers that can't do better.

# Open-source considerations

Taskwondo's source, its client ids and this spec are public, and anyone may build, fork or redistribute `taskwondo-mcp`. The design assumes that:

- **Nothing depends on a secret in the client.** `taskwondo-mcp` is a public client (RFC 8252 section 8.5): no client secret is compiled in or shipped in the `mcpb` bundle, and security rests on PKCE, the loopback redirect, `state`/`iss` checks and the user's consent. Publishing the client id gives nothing away.
- **"Taskwondo app" on the consent page means "registered with this server", not "built by the Taskwondo project".** A fork or a modified build can use the `taskwondo-mcp` id, and no public client can prove otherwise. That is why consent is never skipped for first-party clients and why the loopback hint asks the user whether they just started signing in. Connected apps and per-session revocation are the remedy for a build the user regrets trusting.
- **Each install is its own authorization server.** The issuer is that install's `BASE_URL`, clients discover everything from the server they are pointed at, and nothing calls a hosted service, a central registry or any telemetry endpoint. A token from one install is meaningless to another (its resource names the issuer).
- **Operators are strangers.** The defaults have to be safe without tuning (server on, conservative lifetimes, rate limits on), every operator-facing behaviour is in `MANUAL_INSTALL.md` and `.env.template`, and misconfiguration (a wrong `BASE_URL`, a proxy that drops `/.well-known/oauth-*`) degrades to the legacy flow, never to a broken sign-in.
- **Third-party clients are welcome but not registered by migration.** Only clients maintained in this repository are seeded. Someone building their own tool uses an API key today; [the remote MCP server](remote_mcp.md) opens dynamic registration, but only for the MCP resource, so a third-party tool calling the REST API directly keeps using an API key.
- **Docs, examples and tests use placeholder hosts** (`taskwondo.example.com`) and no real deployment's names, URLs or accounts.

# Testing

- **API unit and handler tests** (`api/internal/handler`, `service`, `repository`; at the HTTP entry point, per AGENTS.md): metadata documents and their `404` with the server off; canonicalisation; redirect matching (loopback any port, exact path, `localhost` refused, non-loopback byte-exact); the authorize validation order (no redirect on bad client or redirect, redirect with `state` and `iss` otherwise, `state` required, PKCE required and S256 only); consent and grant routes refusing `twk_` and `twko_` principals; one live grant per user and client under a concurrent Allow; code redemption success, expiry, wrong redirect, wrong verifier burning the code, reuse revoking the recorded session, a concurrent double redemption; refresh rotation, replay revoking only its session while a second session keeps working, sliding expiry, 30-day idle death; `invalid_target` and `invalid_scope`; `Authorization` on the token endpoint → `invalid_client`; RFC 7009 on each token kind, the last session revoking the grant, unknown tokens answering `200`; the middleware dispatch order (`twkor_` as a bearer is `401`); every barred route refused with a `twko_` token and allowed with a `write` `twk_` key, plus a percent-encoded barred path; audience refusal and its throttled log; the `WWW-Authenticate` header on every `401` shape and its bare form with the server off; the inactive user; `last_used_at` throttling; the per-IP token limiter; the migration up and down; the worker cleanup.
- **MCP tests** (`mcp/`, `httptest` server standing in for the API, `t.Setenv` for `TASKWONDO_URL`/`TASKWONDO_CONFIG_DIR`): discovery against a pre-change server (PRM `404`, the SPA's HTML at `200`, an issuer mismatch) choosing the legacy flow; a wrong `state` or `iss` keeping the listener waiting; static callback pages (a `<script>` in `error` never rendered); precedence, including legacy key → login → logout → signed out (not back on the legacy key); a `url` mismatch never sending tokens; two processes refreshing at once producing exactly one refresh (lock); `invalid_grant` signing out and a `5xx` not; a `401` mid-session refreshing and retrying once; logout revoking and leaving `credentials.json` and its unknown keys (and `namespace`) intact; empty `TASKWONDO_API_KEY` treated as unset.
- **Playwright** (`test/e2e/tests/auth/oauth-first-party.spec.ts`): the consent page for `taskwondo-mcp` (label, loopback hint, signed-out round trip through `/login?next=`); Deny returning `access_denied`; the whole flow driven from the test — Playwright opens the authorize URL with a test-generated PKCE pair, clicks Allow, captures the redirect to `http://127.0.0.1:<port>/callback` with `page.route` instead of a listener, redeems the code through the request context, calls `/auth/me`, refreshes, replays the old refresh token and sees the session die; Connected apps listing two sessions, signing one out in the modal, revoking the grant; the barred routes `403` with the OAuth token; and the existing CLI consent and API keys specs passing unchanged. Each `testUser` spends login budget, so these share one user where the flow allows (the `/auth/login` limit noted in AGENTS.md).
- **Coverage**: the new packages and files hold the 80% target.

# Documents to update when it lands

`README.md` (MCP sign-in), `README_DEV.md` (Vite proxy, new env vars), `MANUAL_INSTALL.md` (reverse-proxy path, `BASE_URL`, operator check), `.env.template`, `mcpb/README.md` and `mcpb/manifest.json` (the `api_key` description), `RELEASE_NOTES.md`, `docs/overview.md` if it lists authentication methods, and AGENTS.md's "Auth" line (JWT + API key + OAuth `twko_`).

# Settled questions

Settled on 2026-11-08, before implementation.

1. **The authorization server is on by default** (`OAUTH_SERVER_ENABLED=true`). It adds no exposure beyond endpoints that act only after a signed-in user clicks Allow, and an off-by-default server would leave most installs minting permanent keys forever.
2. **An issuer mismatch falls back to the legacy flow with a notice**, never a hard error. RFC 8414 says a mismatched issuer must not be used, and the fallback honours that: the legacy flow uses no metadata. A hard error would break sign-in for every install whose `BASE_URL` disagrees with the URL its users type, which is common on LAN and reverse-proxied setups.
3. **No read-only scope in this release.** Nothing asks for it, and adding `taskwondo:read` later is additive. Read-only access stays the job of `read` API keys.
4. **Deactivating a user keeps their grants**: their tokens are refused while the account is inactive and work again on reactivation, as `twk_` keys do. Deleting the user removes everything.
5. **The CLI consent page stays served and unchanged** in this release. Whether to retire it is decided once released MCP builds have moved to OAuth. Interim hardening (the page forwarding a `state` when a client sends one) helps nobody until a client sends it, so it is not done.

# Related

- `api/internal/middleware/auth.go` — bearer dispatch and the new route bar.
- `api/internal/service/auth.go` — `ValidateAPIKey`, the model for `ValidateOAuthAccessToken`.
- `web/src/pages/CliAuthorizePage.tsx` — the legacy consent page this spec leaves in place.
- `web/src/pages/AuthenticationPage.tsx` — where Connected apps lands.
- `mcp/auth.go`, `mcp/tools.go` — the client side being migrated.
