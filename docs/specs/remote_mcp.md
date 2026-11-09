---
type: Design Decision
title: Remote MCP server
description: taskwondo-mcp serve — an opt-in MCP Streamable HTTP endpoint at /mcp so hosted clients (Claude.ai, ChatGPT and other MCP connectors) can use Taskwondo with nothing installed locally. It is an OAuth 2.1 resource server for its own MCP resource, and toward the API a confidential client that trades every connector token for a short-lived API token by RFC 8693 token exchange from its first release, never forwarding the connector's token. Covers the authorization-server additions it needs (dynamic client registration, the MCP resource, client_secret_basic, the exchange grant and its budgets), the 401/503 contract, the remote tool surface, deployment and operation.
tags: [mcp, oauth, remote, connectors, deployment, nginx, security]
status: draft
depends_on:
  - oauth.md
sources:
  - id: mcp-transport
    resource: https://modelcontextprotocol.io/specification/2025-11-25/basic/transports
    title: MCP transports (2025-11-25), Streamable HTTP
  - id: mcp-auth
    resource: https://modelcontextprotocol.io/specification/2025-11-25/basic/authorization
    title: MCP authorization (2025-11-25), including the token passthrough prohibition
  - id: rfc8693
    resource: https://www.rfc-editor.org/rfc/rfc8693
    title: OAuth 2.0 Token Exchange
  - id: rfc7591
    resource: https://www.rfc-editor.org/rfc/rfc7591
    title: OAuth 2.0 Dynamic Client Registration
  - id: rfc9728
    resource: https://www.rfc-editor.org/rfc/rfc9728
    title: OAuth 2.0 Protected Resource Metadata
  - id: rfc8707
    resource: https://www.rfc-editor.org/rfc/rfc8707
    title: Resource Indicators for OAuth 2.0
  - id: rfc6749
    resource: https://www.rfc-editor.org/rfc/rfc6749
    title: The OAuth 2.0 Authorization Framework
---

# Remote MCP server

**The ask.** Settled on 2026-11-08, with no work item yet. Today Taskwondo's MCP server runs only over stdio, on the user's own machine, so it reaches Claude Code and Claude Desktop but not Claude.ai on the web or mobile, ChatGPT, or any other hosted MCP client. This spec adds `taskwondo-mcp serve`: the same tool surface over MCP Streamable HTTP at `https://<host>/mcp`, deployed as an optional container beside the API.

**It builds on [the OAuth authorization server](oauth.md)** and cannot ship before it. That spec gives every token a resource and has the API refuse any token not issued for the API; this one adds the second resource, `<issuer>/mcp`, and the machinery for the MCP container to call the API **without ever forwarding a connector's token**:

- A connector (dynamically registered, RFC 7591) signs its user in through the same consent page and gets a token for the **MCP resource**. The API refuses that token on every route, by the audience rule `oauth.md` already enforces.
- The MCP container is a **confidential client**, `taskwondo-mcp-remote`, and the only client allowed **token exchange** (RFC 8693). For each connector token it receives, it obtains a five-minute token for the **API resource**, tied to the connector's token and dying with it, and uses only that toward the API.

**Token exchange from the first release, with no passthrough mode.** The MCP authorization spec forbids passthrough, and there is no released remote server whose behaviour must be kept, so there is no fallback to forwarding tokens, no flag to turn audience enforcement off, and no transition period. A container without a valid client secret refuses to start.

**Compatibility.** Nothing existing changes. The stdio server, the `mcpb` bundle, `twk_` keys, first-party OAuth sign-in and the web app behave exactly as before, and an install that does not opt in to the remote server has no `/mcp` endpoint and no registration endpoint.

## What exists

As it stood on 2026-11-08, with [the OAuth authorization server](oauth.md) taken as built; kept as the record the design starts from.

- **`mcp/` is one `package main`**: `main.go` registers every tool and calls `server.ServeStdio` (`mark3labs/mcp-go` v0.44.0). Tool handlers ignore their `context.Context` and call `getClient()`, which reads credentials from the environment or the user's config directory (`mcp/auth.go`).
- **Namespace state is a process global**: `activeNamespace` in `mcp/tools.go`, set by `set_namespace` and read by `getClient` and `getClientForDisplayID`. That is fine for a one-user stdio process and wrong for a server shared by many users.
- **Two tools touch the local filesystem**: `upload_attachment` reads `file_path` from disk (`Client.UploadAttachment`, `os.Open`), and `download_attachment` writes to `file_path` (`os.Create`). On a shared server, those paths would be the container's filesystem.
- **`login`, `logout`, `whoami`** manage the local credential files.
- **The API** (after `oauth.md`) issues `twko_` tokens for one resource, `<issuer>/api`, to pre-registered public clients only; it has no registration endpoint, no confidential clients and no exchange grant. Its metadata says `token_endpoint_auth_methods_supported: ["none"]`.
- **nginx** (`docker/nginx.conf`) proxies `/api/` and `^~ /.well-known/oauth-` to `api:8080` and serves the SPA for everything else. It has no `resolver`, so it refuses to start if an upstream host does not resolve.
- **Deployment** is `docker-compose.yml` with `api`, `web`, `worker`, `postgres`, storage, `nats`; `api` and `worker` load `.env`, which holds `JWT_SECRET`, database and storage credentials. `install.sh` generates secrets with `openssl rand -hex`.

## Decisions

1. **Opt-in.** The remote server is a Compose profile, `remote-mcp`, and the API side switches on only when `OAUTH_MCP_CLIENT_SECRET` is set. `install.sh` asks whether to enable it and, if so, generates the secret and adds the profile. Hosted connectors reach the server from the internet, so it only works on an install with a public HTTPS URL; turning it on is the operator's decision.
2. **Token exchange always, no passthrough** (above).
3. **A second resource, `<issuer>/mcp`**, which only dynamically registered clients may request, and which they can only request. First-party clients keep the API resource only.
4. **The MCP container serves its own protected-resource metadata** and the `401` challenge for `/mcp`; the API remains the only authorization server.
5. **Stateless Streamable HTTP**, with no sessions and no server-to-client streaming, so a container restart or a second replica never strands a client.
6. **One binary, two modes.** `taskwondo-mcp` with no arguments is the stdio server, unchanged; `taskwondo-mcp serve` is the remote one. Both register the same tools from the same code, apart from the differences listed under "Tool surface in remote mode".
7. **No per-user state in the process.** A tool's credential and namespace come from the request, never from a global.

# The design

## Authorization-server additions

These extend [the OAuth authorization server](oauth.md); everything not mentioned here (consent, sessions, rotation, replay, revocation, Connected apps, the deny-list, lifetimes) applies unchanged to connector grants. All of it is active only while remote MCP is on (`OAUTH_MCP_CLIENT_SECRET` set, below); with it off the API behaves exactly as `oauth.md` describes.

### Configuration (API)

| Variable | Default | Meaning |
|---|---|---|
| `OAUTH_MCP_CLIENT_SECRET` | unset | The `taskwondo-mcp-remote` client secret. **Set means remote MCP is on**: the MCP resource exists, registration is open, the exchange grant works. At least 32 characters from `[A-Za-z0-9_-]` (`openssl rand -hex 32` qualifies) |
| `OAUTH_MCP_CLIENT_SECRET_PREVIOUS` | unset | Also accepted during a rotation (see "Rotating the client secret") |
| `OAUTH_MCP_RESOURCE_URL` | `<issuer>/mcp` | The MCP resource identifier, canonicalised as `oauth.md` does. Must equal the container's `TASKWONDO_MCP_RESOURCE_URL` |
| `OAUTH_EXCHANGE_RATE_LIMIT` / `_BURST` | `120` / `30` | Successful exchanges per minute per subject grant |
| `OAUTH_EXCHANGE_FAILURE_LIMIT` / `_BURST` | `600` / `100` | Refused exchanges per minute, globally |

- **A malformed secret** (too short, or characters outside the set) is ignored with an `error` log naming the variable, never its value, and remote MCP stays off; the API still starts, so a typo cannot take down the web app.
- **Startup log**: `oauth: remote mcp on` with the fields `resource` and `secrets` (`1`, or `2` mid-rotation), or `oauth: remote mcp off`.
- **The secret lives in configuration, never the database**, and is compared as SHA-256 digests in constant time against each configured value, with no early exit, before any database read.

### The MCP resource and the client registry

- **`allowed_resources` gains the symbolic name `mcp`**, resolved to `OAUTH_MCP_RESOURCE_URL`. While remote MCP is off, `mcp` resolves to nothing: a client allowed only `mcp` cannot be sent through consent (`invalid_target`), and its existing tokens stop validating at the MCP container (exchange is refused).
- **The audience rule is unchanged**: the API accepts only API-resource tokens, so an MCP-resource token is `401` with `error_description="token not issued for this resource"` on every API route. This is what makes a stolen connector token worth no more than the tool surface.
- **The migration** (`000064_remote_mcp`) adds two nullable columns and one seeded row; nothing existing changes shape:

```sql
ALTER TABLE oauth_tokens
    ADD COLUMN actor_client_id   TEXT REFERENCES oauth_clients(id) ON DELETE CASCADE,
    ADD COLUMN parent_token_hash TEXT REFERENCES oauth_tokens(token_hash) ON DELETE CASCADE;
CREATE INDEX idx_oauth_tokens_parent ON oauth_tokens(parent_token_hash) WHERE parent_token_hash IS NOT NULL;

ALTER TABLE oauth_clients ADD COLUMN registered_by_ip TEXT;  -- dynamic registrations only, for abuse triage

INSERT INTO oauth_clients (id, client_name, client_type, first_party, redirect_uris,
                           allowed_resources, allowed_scopes, allowed_grant_types)
VALUES ('taskwondo-mcp-remote', 'Taskwondo MCP (remote)', 'confidential', true, '{}',
        '{api}', '{taskwondo}', '{urn:ietf:params:oauth:grant-type:token-exchange}')
ON CONFLICT (id) DO NOTHING;
```

### Dynamic client registration

`POST /api/v1/oauth/register` (RFC 7591), unauthenticated, under the OAuth per-IP limiter. `404` while remote MCP is off. Metadata gains `"registration_endpoint": "<issuer>/api/v1/oauth/register"` while on.

- **Body**: `{"client_name": "Claude", "redirect_uris": ["https://claude.ai/api/mcp/auth_callback"]}`. Other RFC 7591 fields are ignored. Whatever `token_endpoint_auth_method` is asked for, the client is registered as `none` (RFC 7591 section 3.2.1 lets the server substitute, and the response says so).
- **`201`** → `{"client_id": "tc_<32 hex>", "client_id_issued_at": <unix>, "client_name", "redirect_uris", "grant_types": ["authorization_code", "refresh_token"], "response_types": ["code"], "token_endpoint_auth_method": "none"}`.
- **The row**: `first_party = false`, `client_type = 'public'`, `allowed_resources = {mcp}`, `allowed_scopes = {taskwondo}`, `allowed_grant_types = {authorization_code, refresh_token}`, `registered_by_ip`. Nothing in the body can set these columns. The `tc_` prefix can never collide with a first-party id.
- **Redirect URIs**: one to ten, each absolute, no fragment or userinfo, and either `https` or `http` on a loopback host (`localhost`, `127.0.0.0/8`, `::1`; loopback ports match any port as in `oauth.md`). Otherwise `400 {"error": "invalid_redirect_uri", …}`; an unparseable body is `invalid_client_metadata`.
- **Names**: `client_name` is trimmed, cut to 100 characters, and defaults to `MCP client`. A name equal (case-insensitive, after trimming) to a first-party client's id or name, or to `Taskwondo` alone, is registered as `MCP client`, so no connector can appear on the consent page as a Taskwondo app.
- **Scopes from a dynamic client are filtered, never refused**: anything outside `{taskwondo}` (`offline_access`, unknown values) is dropped and the grant is `taskwondo`, as RFC 6749 section 3.3 permits. Connectors routinely ask for scopes they saw elsewhere, and refusing them would break sign-in for no gain. First-party clients keep `oauth.md`'s strict `invalid_scope`.
- **An absent `resource`** means the client's default, the MCP resource, because clients written to the 2025-03-26 MCP spec omit it.
- **Cleanup**: the worker's `token.cleanup` task deletes dynamic clients that have no grant and were registered more than 7 days ago, so anonymous registrations cannot pile up.

### The consent page for connectors

Same page, same flow as `oauth.md`, with two differences for a non-first-party client:

- No "Taskwondo app" label. Instead: "**<name>** registered itself with this server. Only allow it if you are connecting it from **<redirect host>** right now." The redirect host (`claude.ai`, `chatgpt.com`, …) comes from consent-info's `redirect_host` and is shown prominently, because the name is self-declared and the host is the part that can't be faked.
- What it may do reads "Use Taskwondo's tools on your behalf: read and change everything your account can, except API keys, connected apps and system administration."

Nine locales, as for every new string.

### Confidential client authentication

- **`client_secret_basic` only** (`Authorization: Basic base64(urlencode(client_id) ":" urlencode(client_secret))`, RFC 6749 section 2.3.1), on the token endpoint only. `client_secret_post` is not accepted. Metadata gains `"client_secret_basic"` in `token_endpoint_auth_methods_supported` while remote MCP is on.
- **`invalid_client`** is `401 {"error": "invalid_client", "error_description": "client authentication failed"}` with `WWW-Authenticate: Basic realm="taskwondo"`, for a wrong secret, a missing one (any exchange request without Basic), a public client presenting a secret, an unknown client, a malformed header, and a Basic `client_id` that differs from a form `client_id`. A Basic header naming a public client with an empty password presents no secret: it is the `none` method with that `client_id`. Basic credentials are checked first, whatever the grant type. This replaces `oauth.md`'s rule that any `Authorization` header on the token endpoint is `invalid_client`.
- **The confidential client may only exchange**: a verified Basic request for `authorization_code` or `refresh_token` is `unauthorized_client`.
- **`/oauth/revoke` ignores `Authorization`** (`revocation_endpoint_auth_methods_supported` stays `["none"]`); the container never revokes.

### Token exchange

`POST /api/v1/oauth/token`, form-encoded, authenticated as above. Metadata gains `urn:ietf:params:oauth:grant-type:token-exchange` in `grant_types_supported` while remote MCP is on.

| Parameter | Value |
|---|---|
| `grant_type` | `urn:ietf:params:oauth:grant-type:token-exchange` |
| `subject_token` | the connector's `twko_` access token (required) |
| `subject_token_type` | `urn:ietf:params:oauth:token-type:access_token` (anything else: `invalid_request`) |
| `resource` | the API resource; absent means the API resource; anything else `invalid_target` |
| `audience` | optional; when present must equal the API resource, else `invalid_target` |
| `scope` | optional; must be a subset of the subject's scope, else `invalid_scope` |
| `requested_token_type` | optional; only `urn:ietf:params:oauth:token-type:access_token`, else `invalid_request` |
| `actor_token`, `actor_token_type` | not accepted (`invalid_request`): the client authentication is the actor |

- **Only `taskwondo-mcp-remote` may call it** (its `allowed_grant_types`); any other client is `unauthorized_client`.
- **The subject must be a live access token bound to the MCP resource**, not itself an exchanged token, on a live session and grant, of an active user. **Every subject refusal** (unknown, malformed, a refresh token, expired, revoked, bound to the API resource, an exchanged token, a revoked session or grant, an inactive user) **is `400 {"error": "invalid_request", "error_description": "subject_token invalid"}`** with exactly that description, as RFC 8693 section 2.2.2 directs; the specific reason is logged at `debug` only. The container maps exactly this answer to a `401` for the connector.
- **The issued token**: `twko_<64 hex>`, kind `access`, resource the API resource, the subject's session (and so its grant), the subject's scope or the narrower one asked for, `actor_client_id = 'taskwondo-mcp-remote'`, `parent_token_hash` = the subject's hash, `expires_at = min(now + 5 min, subject.expires_at)`. No refresh token: the container exchanges again.
- **Response** (`200`, `Cache-Control: no-store`): `{"access_token": "twko_…", "issued_token_type": "urn:ietf:params:oauth:token-type:access_token", "token_type": "Bearer", "expires_in": <seconds>, "scope": "taskwondo"}`.
- **A child dies with its parent.** Validation of an exchanged token joins its parent and refuses it when the parent is revoked or missing (fail closed). So RFC 7009 revocation of the connector's access token, the connector's refresh revoke (its session), Sign out in Connected apps, and Revoke on the grant all reach it on its next API request. The `ON DELETE CASCADE` removes children when cleanup deletes the parent.
- **Validation of an exchanged token does not update `last_used_at` on its own**; the session's `last_used_at` moves through the throttled update `oauth.md` already describes, so Connected apps shows the connector as used when its tools are.
- **The principal** is `oauth.md`'s, plus `Actor: "taskwondo-mcp-remote"`; the request log line carries `client_id` (the connector's) and `actor`, so a log line reads "Claude, through the remote MCP server". The deny-list applies unchanged.

### Rate limits and CORS

- **The per-IP OAuth limiter** (`OAUTH_RATE_LIMIT`, `OAUTH_RATE_BURST`, defined in `oauth.md`) also covers registration. Connector platforms call the token endpoint server-to-server from a few shared egress IPs, so on a busy install the operator may need to raise it; `MANUAL_INSTALL.md` says so.
- **A request whose `client_secret_basic` verifies skips the per-IP bucket**, because every exchange arrives from the MCP container's one address. It is charged instead in the handler: a successful exchange **per subject grant** (`OAUTH_EXCHANGE_RATE_LIMIT`), any other refusal to the global **failure budget** (`OAUTH_EXCHANGE_FAILURE_LIMIT`). A spent budget answers `429 {"error": "temporarily_unavailable"}` with `Retry-After`. A server error is not charged. A request with no Basic header, or one that fails, is charged per IP as usual.
- **A refused subject is neither charged nor answered `429`.** The container turns `subject_token invalid` into the `401` that makes a connector refresh; if a flood of random well-shaped bearers could spend the failure budget, every legitimate expired-token answer would become a `503` and connectors would stop recovering. The container's own per-IP limit and negative cache (below) are what bound that flood.
- **CORS**: the metadata documents, `register`, `token` and `revoke` answer any origin (`Access-Control-Allow-Origin: *`, methods `GET, POST, OPTIONS`, headers `Content-Type, Authorization, MCP-Protocol-Version`, no credentials, preflight `204`), so a browser-based MCP client on its own origin can register and redeem codes. Every other API route keeps the `BASE_URL` origin.

## The MCP container

### Environment

Serve mode **fails at startup** when a required variable is missing or malformed. A half-configured container would otherwise fail later, more quietly, and per user.

| Variable | Required | Meaning |
|---|---|---|
| `TASKWONDO_API_URL` | yes | The API as this container reaches it, `http://api:8080` in Compose. Used for every exchange and every tool call; never the public URL, so tokens and the secret stay on the Docker network |
| `TASKWONDO_MCP_RESOURCE_URL` | yes | The canonical public MCP URL, e.g. `https://taskwondo.example.com/mcp`. It is the OAuth resource identifier and must equal the API's `OAUTH_MCP_RESOURCE_URL`; an explicit default port is stripped before it is advertised |
| `TASKWONDO_MCP_ISSUER` | no | The issuer to advertise; defaults to the resource URL's scheme and host. Must equal the API's issuer (its canonical `BASE_URL`) |
| `TASKWONDO_MCP_CLIENT_SECRET` | yes | The `taskwondo-mcp-remote` secret, equal to the API's `OAUTH_MCP_CLIENT_SECRET`. Missing, empty or malformed **refuses startup**, without printing it |
| `TASKWONDO_MCP_ADDR` | no | Listen address, default `:8090` |
| `TASKWONDO_MCP_RATE_LIMIT` / `_BURST` | no | Requests per minute per client IP on `/mcp`, default `300` / `60` |
| `TASKWONDO_MCP_TRUSTED_PROXIES` | no | CIDRs whose `X-Forwarded-For` is believed, default `172.16.0.0/12` (Docker's private range, where nginx is) |
| `TASKWONDO_MCP_MAX_ATTACHMENT_BYTES` | no | Largest attachment the remote upload and download tools move inline, default `10485760` (10 MiB) |

`TASKWONDO_URL`, `TASKWONDO_API_KEY`, `TASKWONDO_NAMESPACE` and `TASKWONDO_CONFIG_DIR` are never read in serve mode; if any is set, startup logs a `warn` saying it is ignored. No code path in serve mode opens a file under a config directory. At startup the container logs `taskwondo-mcp serve: listening` with the resource and issuer, and `token exchange enabled as client taskwondo-mcp-remote`.

### Endpoints

| Method and path | Auth | Answer |
|---|---|---|
| `POST /mcp` | bearer | MCP Streamable HTTP, stateless; one `application/json` reply per request. Rate-limited per client IP |
| `GET /mcp` | bearer | `405`: server-to-client streaming is disabled (see "Why stateless") |
| `DELETE /mcp` | bearer | `405`: there are no sessions to end |
| `OPTIONS /mcp` | none | CORS preflight, `204` |
| `GET /.well-known/oauth-protected-resource/mcp` | none | Protected Resource Metadata (RFC 9728) |
| `GET /.well-known/oauth-protected-resource` | none | The same document, for clients that ask at the root |
| `GET /healthz` | none | `200 ok`, liveness only; it does not call the API |

The metadata (`Content-Type: application/json`, `Access-Control-Allow-Origin: *`, `Cache-Control: no-store`):

```json
{"resource": "https://taskwondo.example.com/mcp", "authorization_servers": ["https://taskwondo.example.com"], "scopes_supported": ["taskwondo"], "bearer_methods_supported": ["header"], "resource_name": "Taskwondo"}
```

### The 401 contract

Every `/mcp` request other than a preflight needs `Authorization: Bearer twko_…`. Refusals use the API's error envelope (`{"error": {"code": "UNAUTHORIZED", "message": "…"}}`) and the challenge:

```
WWW-Authenticate: Bearer resource_metadata="https://taskwondo.example.com/.well-known/oauth-protected-resource/mcp", scope="taskwondo"
```

| Request | Status | `error=` attribute |
|---|---|---|
| Over the per-IP limit | `429`, `Retry-After`, code `RATE_LIMIT_EXCEEDED` | no challenge |
| No `Authorization`, or a scheme other than `Bearer` | `401` | none |
| A bearer not shaped `twko_` + 64 lowercase hex (a `twk_` key, a JWT, a refresh token, a truncated token) | `401`, without an exchange | `, error="invalid_token"` |
| Exchange answered `subject_token invalid` | `401` | `, error="invalid_token"` |
| A tool call whose re-exchange (below) is refused the same way | `401`; the JSON-RPC reply is discarded | `, error="invalid_token"` |
| Any other exchange outcome, or a tool call that could not get an API token the API accepts | `503`, `Retry-After: 5`, code `SERVICE_UNAVAILABLE` | no challenge |

- **An outage or a misconfiguration is a `503`, never a `401`.** A `401` tells the connector to throw its token away and send the user back through consent. Doing that because the API was restarting, or because the operator rotated the secret wrongly, would turn every upgrade or configuration slip into a forced re-login of every connected user.
- **A `twk_` key is refused without a round trip.** It is a real credential to the API but not one issued for this resource, and accepting it here would let a key holder skip consent, and would put a permanent key into a connector platform's storage.
- **CORS headers** (`Access-Control-Allow-Origin: *`, `Access-Control-Expose-Headers: WWW-Authenticate, Retry-After`) are on the `401`, `429` and `503` too: a browser client that can't read the challenge can't find the authorization server, and one that can't read `Retry-After` can't back off. No cookies are involved, so the wildcard grants nothing a token does not.
- **The per-IP limit** is a token bucket keyed by client IP: the peer, or, when the peer is in `TASKWONDO_MCP_TRUSTED_PROXIES`, `X-Forwarded-For` walked right to left to the first untrusted entry, as the API's `clientIP` reads it. It sits inside CORS (preflights aren't counted) and before the shape check (malformed bearers are).

### Token exchange in the container

- **The exchange is the validation.** A well-shaped bearer is sent to `POST <TASKWONDO_API_URL>/api/v1/oauth/token` with `client_secret_basic` (`taskwondo-mcp-remote`), `grant_type=urn:ietf:params:oauth:grant-type:token-exchange`, `subject_token=<the bearer>`, `subject_token_type=urn:ietf:params:oauth:token-type:access_token`. `resource` is omitted, which the API reads as its own, so the container never needs to know the API resource URL. A `200` with a `twko_` `access_token`, `token_type: Bearer` and a positive `expires_in` is the whole check. There is no separate `/auth/me` call.
- **Cache.** The exchanged token is cached under the SHA-256 of the subject until 30 s before its expiry (so at most about 4.5 minutes), bounded at 4096 entries, expired entries evicted first. `initialize`, `tools/list` and every tool call in that window cost no exchange. A token living 30 s or less is used once and not cached.
- **Negative cache.** `subject_token invalid` is remembered for 30 s under the subject hash, so a connector retrying a dead token costs one exchange per 30 s. A `5xx`, a `429` or no answer is never cached, so an API restart costs nothing once it's over.
- **Tools see only the exchanged token, never the subject.** The request's context carries a token source, not a fixed string; the tool's API client asks it on every API request, so a call that outlives the exchanged token picks up a fresh one.
- **An API `401` on a tool call** drops the cache entry, re-exchanges, and retries that request once, body included. A refused re-exchange makes the HTTP reply the `401` challenge; any other re-exchange failure, or a `401` on the retry (the API refusing a token it just issued), makes it the `503`.
- **Failure mapping.** Only `400 invalid_request` with `error_description` exactly `subject_token invalid` becomes a `401`. Everything else (`invalid_client`, `unauthorized_client`, `unsupported_grant_type`, `invalid_target`, `invalid_scope`, any other `invalid_request`, `429`, `5xx`, an unparseable body, no answer) becomes a `503` with one `error` log line, `token exchange refused: HTTP <status> error=<code>` or `token exchange got no answer`, which names the code and never a token or the secret. A misconfigured secret therefore shows up as an outage in the operator's logs, not as a mass forced re-consent.
- **The revoke window.** Revocation reaches exchanged children at the API at once, so a revoked connector's next tool call fails. `initialize` and `tools/list` never call the API, so they keep answering from the cached exchange for up to about 4.5 minutes after a revoke; neither reaches any data.
- **A tool middleware refuses any call whose context carries no token source**, so a request that somehow bypassed the bearer check is refused, never served with some other credential.

### Why stateless

The server runs with mcp-go's `WithStateLess(true)` and `WithDisableStreaming(true)`: no `Mcp-Session-Id`, every request stands alone, `GET /mcp` is `405` (Streamable HTTP allows it). Taskwondo's tools are thin calls to the API and emit no notifications, so nothing is lost. A stateful server would strand every connected client on each container restart (each upgrade), and would need sticky routing for a second replica. Without disabling streaming, mcp-go would hold an idle event stream open on every `GET` under the one empty session id stateless mode shares across callers, which is pointless and a route for one user's notifications to reach another. DNS-rebinding (localhost) protection is off because the server sits behind nginx with the public `Host`, and every request it serves needs a bearer no rebinding page holds.

**Timeouts.** No `ReadTimeout` (net/http would cancel a running handler when it fires); `ReadHeaderTimeout` 10 s; `WriteTimeout` 5 minutes, enough for a 10 MiB attachment through the API. nginx's `proxy_read_timeout` on `/mcp` matches.

## Tool surface in remote mode

Every stdio tool is registered in serve mode, with these differences. The stdio server's behaviour does not change except where noted.

| Tool | Remote mode |
|---|---|
| `login`, `logout` | **Not registered.** The connector's OAuth flow is the sign-in; Connected apps in the web app is the sign-out |
| `whoami` | Reports the account, and `OAuth connection (<connector name>)` as the credential source; calls `/auth/me` with the exchanged token |
| `set_namespace` | **Not registered**: a stateless server has nowhere to keep it. See "Namespaces" |
| `upload_attachment` | Takes `filename`, `content_base64`, optional `content_type` and `comment` instead of `file_path` |
| `download_attachment` | Takes no `file_path`; returns the file inline (below) |
| everything else | Unchanged, apart from the optional `namespace` argument below |

**What a tool may do is decided by the user who consented**, under the same RBAC as their web session, minus `oauth.md`'s deny-list.

### Namespaces

- `activeNamespace` stops being a package global. A small `session` value on the request context holds it: in stdio mode one process-wide value, set by `set_namespace` exactly as today; in serve mode none.
- **Every tool that today uses the active namespace gains an optional `namespace` argument** (the namespace slug), in both modes. In stdio it overrides the active namespace for that call; in serve mode it is the only way to choose one. Without it, a tool behaves as stdio does with no active namespace: display-ID tools resolve the namespace through search, and the rest use the API's default namespace routes.
- **An ambiguous display ID** (the same `KEY-123` in two namespaces) answers, in serve mode, "TF-141 exists in namespaces `acme` and `personal`; call the tool again with `namespace` set to one of them" instead of pointing at `set_namespace`.
- `list_namespaces` is unchanged and is how a connector discovers the slugs.

### Attachments without a filesystem

- **No serve-mode code path reads or writes a local path.** The `file_path` variants are not registered, and a test asserts that no serve-mode tool schema has a `file_path` property. Otherwise any connector user could read files from the container (its environment holds the client secret) or write into it.
- **Upload**: `content_base64` is decoded, refused above `TASKWONDO_MCP_MAX_ATTACHMENT_BYTES` with "File is larger than <n> MiB; upload it in the web app", and sent to the API as today's multipart upload. `filename` is required and passed through the API's own validation.
- **Download**: the API response is read up to the cap (checked first against `size_bytes` from the attachment listing, and enforced while streaming). Text types (`text/*`, `application/json`) are returned as text content; anything else as an MCP embedded resource with a `blob` (base64) and `mimeType`. Over the cap, the tool answers with the attachment's size and a link to the work item in the web app. Nothing touches disk; the body is held in memory only for the call.
- The `/mcp` request body limit is the cap times 4/3 plus 1 MiB (base64 overhead plus the JSON-RPC envelope); nginx's `client_max_body_size` on `/mcp` is 16m for the default cap.

### Code layout

The stdio and serve modes share every tool handler, so the refactor is the bulk of the work:

- **Handlers take their context.** `getClient()` becomes `getClient(ctx)` and `getClientForDisplayID(ctx, …)`. In stdio, the context carries the credential resolution from `oauth.md` (token file, legacy key, env) and the process session; in serve mode, the exchanged-token source and no session.
- **Tool definitions become mode-aware**: one function per tool returning its `mcp.Tool` for a given mode, so the remote upload/download schemas and the omitted tools are visible in one place.
- **New files**: `mcp/serve.go` (flag parsing, environment, HTTP server, routes, metadata, CORS), `mcp/serve_bearer.go` (shape check, challenge, the 401/503 mapping), `mcp/serve_exchange.go` (exchange client, caches, token source), `mcp/serve_ratelimit.go` (per-IP limiter, trusted proxies). Moving `mcp/` into `internal/` packages is worthwhile but not part of this spec.
- **`main.go`**: `taskwondo-mcp serve` runs serve mode; no arguments runs stdio as today; any other argument prints usage and exits `2`.

## Deployment

### Image and Compose

- **New image** `ghcr.io/marcoshack/taskwondo/mcp`, built from a new `docker/Dockerfile.mcp` (static Go binary on a minimal base, non-root user, entrypoint `taskwondo-mcp serve`, `HEALTHCHECK` on `/healthz`). The CI workflow builds and publishes it with the other images, under the same tags. The stdio binaries and the `mcpb` bundle are built as today.
- **`docker-compose.yml`** gains the service under the `remote-mcp` profile:

```yaml
  mcp:
    image: ghcr.io/marcoshack/taskwondo/mcp:${IMAGE_TAG:-latest}
    profiles: ["remote-mcp"]
    environment:
      - TASKWONDO_API_URL=http://api:8080
      - TASKWONDO_MCP_RESOURCE_URL=${BASE_URL:?required}/mcp
      - TASKWONDO_MCP_ISSUER=${BASE_URL:?required}
      - TASKWONDO_MCP_CLIENT_SECRET=${OAUTH_MCP_CLIENT_SECRET:?required when the remote-mcp profile is on}
    depends_on:
      api:
        condition: service_healthy
    restart: unless-stopped
```

- **The container never loads `.env`.** It receives exactly the variables above, interpolated by Compose, so it never holds `JWT_SECRET`, database or storage credentials. Its only secret is its own.
- **The `api` service** gets `OAUTH_MCP_CLIENT_SECRET` and `OAUTH_MCP_CLIENT_SECRET_PREVIOUS` from `.env` as it gets everything else. An install without the profile simply leaves them unset.
- **`install.sh`** asks "Enable the remote MCP server for Claude.ai, ChatGPT and other hosted AI assistants? (needs a public HTTPS URL)", and on yes writes `OAUTH_MCP_CLIENT_SECRET=$(openssl rand -hex 32)` and `COMPOSE_PROFILES=remote-mcp` to `.env`. On no, nothing is written.
- **The e2e compose** runs the `mcp` service with a fixed test secret on both sides, `TASKWONDO_MCP_RESOURCE_URL` and `OAUTH_MCP_RESOURCE_URL` set to the URL Playwright reaches it at, and a raised `TASKWONDO_MCP_RATE_LIMIT` (every spec comes from one container). **`make dev`** runs `taskwondo-mcp serve` under air when `DEV_REMOTE_MCP=1`, with a fixed dev secret on both sides and `http://localhost:5173/mcp` as the resource.

### nginx and Vite

- **`docker/nginx.conf`** gains `resolver 127.0.0.11 valid=10s;` and these locations, each using a variable in `proxy_pass` so nginx starts when the `mcp` host doesn't exist (the profile is off):
  - `location = /mcp`: to `mcp:8090`, `proxy_buffering off`, `proxy_read_timeout 300s`, `client_max_body_size 16m`, forwarding `Authorization`, `Host`, `X-Forwarded-For`, `X-Forwarded-Proto`.
  - `location = /.well-known/oauth-protected-resource/mcp` and `location = /.well-known/oauth-protected-resource`: to `mcp:8090`. Exact matches win over `oauth.md`'s `^~ /.well-known/oauth-` prefix, which keeps sending the authorization-server metadata and the API resource's metadata to the API.
  - When `mcp` is not running these answer `404` through `error_page 502 503 504 = @mcp_off` (`return 404;`), not a gateway error, so a connector pointed at an install without the profile gets a clean "not here".
- **`web/vite.config.ts`** sends `^/mcp$` and the two MCP metadata paths to `localhost:8090`, listed before the `/.well-known/oauth-` key that goes to the API (Vite takes the first matching key).

### Checking an install

With `H` the public URL, after `docker compose up -d`:

| Check | Command | Expected |
|---|---|---|
| API side on | API log | `oauth: remote mcp on` with `resource` = `$H/mcp` and `secrets` = `1` |
| Container on | `docker compose logs mcp` | `token exchange enabled as client taskwondo-mcp-remote` |
| MCP metadata | `curl -s $H/.well-known/oauth-protected-resource/mcp \| jq -r .resource` | `$H/mcp` |
| Challenge | `curl -si -X POST $H/mcp \| grep -ci '^www-authenticate: Bearer resource_metadata='` | `1` |
| Issuers agree | `curl -s $H/.well-known/oauth-authorization-server \| jq -r .issuer` | `$H`, equal to the metadata's `authorization_servers[0]` |
| Registration | `curl -s $H/.well-known/oauth-authorization-server \| jq -r .registration_endpoint` | `$H/api/v1/oauth/register` |

A `404` on the MCP metadata means the profile is off or nginx lacks the blocks; `oauth: remote mcp off` means the API has no valid secret; a different issuer means `BASE_URL` isn't the URL users and connectors open, and connectors will reject the authorization server.

### Rotating the client secret

Two phases, so no exchange is refused in between:

1. Set `OAUTH_MCP_CLIENT_SECRET_PREVIOUS` to the current value and `OAUTH_MCP_CLIENT_SECRET` to a new one (`openssl rand -hex 32`) in `.env`.
2. `docker compose up -d api mcp`. The API now accepts both (`secrets` = `2`), and the container uses the new one.
3. Once the container is up, remove `OAUTH_MCP_CLIENT_SECRET_PREVIOUS` and `docker compose up -d api` (`secrets` = `1`).

A changed environment needs `up -d`, not `restart`. A container briefly holding a secret the API does not accept answers `503`, never `401`, so a botched rotation is an outage to fix, not a mass sign-out.

### Turning it off

Remove `remote-mcp` from `COMPOSE_PROFILES` and unset `OAUTH_MCP_CLIENT_SECRET`, then `docker compose up -d --remove-orphans`. Registration and the exchange grant disappear, connector tokens stop working at once (nothing can exchange them), and the connectors' grants stay listed in Connected apps until their users revoke them or they expire. Turning it back on with a new secret brings existing connector sessions back as long as their refresh tokens are still alive.

## Adding a connector

Documented for users in the README and the help text, with placeholders:

- **Claude.ai** (web, desktop, mobile): Settings > Connectors > Add custom connector, URL `https://<your Taskwondo>/mcp`. Claude discovers the authorization server from the `401` and the metadata, registers itself, and opens the Taskwondo consent page; approve it signed in as the account the tools should act as.
- **ChatGPT**: in developer mode, create a connector with the MCP server URL `https://<your Taskwondo>/mcp` and OAuth authentication, then approve the consent page the same way.
- **Any MCP client that implements the 2025-03-26 or later authorization spec** works the same way.
- **The connector platform's servers must reach the URL**: an install on a private network, or behind VPN-only access, can't be used by a hosted connector. The local stdio server remains the answer there.

# Open-source considerations

- **Nothing about the deployment is assumed.** Every URL derives from `BASE_URL`, the feature is opt-in, and the image, Compose service, nginx blocks and install prompt ship in the repository so any operator gets the same setup.
- **The client secret is per install.** It is generated by `install.sh` (or by the operator) and never committed, baked into an image, or shared between installs. The public source reveals how the exchange works, not how to perform it.
- **Dynamic registration is open by design**, because connectors require it, and is bounded by the per-IP limiter, the cleanup of unused registrations, the name rules that keep a connector from posing as a Taskwondo app, and the consent page's emphasis on the redirect host. A registration grants nothing until a signed-in user clicks Allow.
- **The connector platforms are third parties**: their tokens are bound to the MCP resource and refused by the API, so whatever a platform stores is worth only the tool surface, and the user can revoke it in Connected apps.
- **Examples and tests use placeholder hosts** (`taskwondo.example.com`) and no real deployment's names, URLs or accounts.

# Security notes

- **No token passthrough, ever**: the API never sees a connector token as a bearer, and refuses it if it does. A stolen connector token is worth the tool surface, which a thief could reach at `/mcp` anyway, until the user revokes it.
- **Exchanged tokens never leave the Docker network** and live at most five minutes; they die with their parent.
- **The client secret is the new crown jewel.** Leaked together with a connector token, it turns that token into an API token: the passthrough gap reopened for that holder, not widened. It is never logged, never in a URL, never in the database, held only by the `api` service and the `mcp` container, and rotated in two phases.
- **No anonymous path drains exchange capacity**: malformed bearers are refused in the container, `/mcp` is limited per IP, refused subjects are negative-cached and never masked by the failure budget, and successful exchanges are charged per grant.
- **No filesystem in serve mode**: no tool can read or write the container's disk.
- **No cross-user state**: no namespace, credential or cache entry is shared between requests except the exchange cache, which is keyed by the hash of the caller's own token.
- **Consent is never skipped** for a connector, and its self-declared name can't borrow a first-party one.

# Testing

- **API** (handler level, per AGENTS.md): registration (success, redirect rules, name rewriting, `404` with remote MCP off, the per-IP limit); consent-info and authorize for a `tc_` client (scope filtering, default MCP resource, `invalid_target` for the API resource); the MCP-resource token refused on API routes with the audience error; `client_secret_basic` success and every `invalid_client` case, including a Basic `client_id` that differs from the form's and a public client with an empty password; the confidential client refused on code and refresh; exchange success and every refusal (`subject_token invalid` for unknown, refresh, expired, revoked, API-resource, exchanged, revoked-session, revoked-grant and inactive-user subjects; `invalid_target`, `invalid_scope`, `invalid_request` for actor tokens and bad token types); the child refused after each revocation path; per-grant and failure budgets, a refused subject charging neither; the previous-secret rotation window; a malformed secret leaving remote MCP off and the API up; metadata gaining and losing its fields with the switch; CORS on the protocol routes; dynamic-client cleanup; the migration up and down.
- **MCP container** (`httptest` server standing in for the API): startup refusing each missing or malformed required variable; metadata and challenge; shape check refusing `twk_`, JWT and refresh tokens without an exchange; per-IP limit with and without trusted proxies; exchange as validation; cache and negative cache (and none on `5xx`); a tool call sending the exchanged token, never the subject; `401` mid-call → re-exchange and one retry, then `401` or `503`; the failure mapping for every listed outcome; the no-token-source middleware; serve-mode schemas having no `file_path` and no `login`, `logout` or `set_namespace`; upload and download caps; inline download as text and as blob; the ambiguous-namespace message; two concurrent requests from different users never sharing a namespace or token.
- **Stdio regression**: the existing `mcp/tools_test.go` suite passes after the context refactor, plus the new optional `namespace` argument overriding the active one.
- **Playwright** (`test/e2e/tests/api/remote-mcp.spec.ts`, the e2e compose running the `mcp` service): register a client over HTTP; drive consent with Playwright and capture the redirect; redeem; `POST /mcp` `initialize`, `tools/list`, and a real tool call (`list_projects` on the test user's project); the connector token `401` at `/api/v1/auth/me` with the audience error; the API request log line carrying `actor`; upload and download an attachment through `/mcp`; revoke the connector in Connected apps and see the next tool call answer `401`; a malformed bearer `401` at `/mcp`. And in `tests/auth/oauth-connectors.spec.ts`: the consent page for a `tc_` client (no Taskwondo label, the redirect-host warning) and its grant listed in Connected apps. Shared users per AGENTS.md's login rate-limit note.

# Documents to update when it lands

`README.md` (connectors, and that the local MCP server stays the option for private installs), `MANUAL_INSTALL.md` (the profile, the variables, the nginx blocks for operators with their own proxy, the install check, rotation, turning it off, raising `OAUTH_RATE_LIMIT`), `.env.template`, `install.sh`, `README_DEV.md` (`DEV_REMOTE_MCP`, the Vite routes), `RELEASE_NOTES.md`, and [the OAuth authorization server](oauth.md), whose follow-up section points here.

# Settled questions

Settled on 2026-11-08, before implementation.

1. **The remote server is opt-in**, through the `remote-mcp` Compose profile and `OAUTH_MCP_CLIENT_SECRET`, with `install.sh` asking. It opens an anonymous registration endpoint and hands tokens to third-party platforms, which is the operator's choice to make; the install prompt makes it one keystroke.
2. **The inline attachment cap is 10 MiB**, configurable with `TASKWONDO_MCP_MAX_ATTACHMENT_BYTES`. Connectors put tool results into a model's context, where large binaries are rarely useful; bigger files belong in the web app.
3. **Client ID Metadata Documents come later**, as an addition. The major connectors still register dynamically, and the client registry can take URL-shaped client ids without a schema change.

# Related

- [OAuth authorization server](oauth.md): the server, sessions, the deny-list and the audience rule this spec extends.
- `mcp/main.go`, `mcp/tools.go`, `mcp/client.go`: the tool surface both modes share.
- `docker/nginx.conf`, `docker-compose.yml`, `install.sh`: where the deployment changes land.
