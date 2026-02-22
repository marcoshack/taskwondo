# Taskwondo Manual Installation

This guide covers installing Taskwondo without Docker Compose. You'll need to provide your own PostgreSQL and S3-compatible storage (MinIO, AWS S3, etc.) and a reverse proxy for the web frontend.

## Prerequisites

- PostgreSQL 15+
- S3-compatible object storage (MinIO, AWS S3, DigitalOcean Spaces, etc.)
- Nginx (or any web server that can serve static files and reverse-proxy)

## Archive Contents

```
taskwondo-<version>/
  bin/taskwondo          # API server (static Linux binary)
  html/                  # Web frontend (static files)
  nginx.conf             # Nginx site configuration (reference)
  .env.template          # Configuration template
  README.md              # This file
```

## 1. Configuration

Copy `.env.template` to `.env` and edit it:

```sh
cp .env.template .env
```

### Required values

| Variable | Description |
|---|---|
| `DATABASE_URL` | PostgreSQL connection string, e.g. `postgres://user:pass@localhost:5432/taskwondo?sslmode=disable` |
| `JWT_SECRET` | Random hex string, minimum 32 characters |
| `ADMIN_EMAIL` | Email for the initial admin user |
| `ADMIN_PASSWORD` | Password for the initial admin user |
| `STORAGE_ENDPOINT` | S3-compatible endpoint, e.g. `localhost:9000` or `s3.amazonaws.com` |
| `STORAGE_ACCESS_KEY` | S3 access key |
| `STORAGE_SECRET_KEY` | S3 secret key |
| `STORAGE_BUCKET` | Bucket name for attachments |
| `STORAGE_USE_SSL` | `true` for HTTPS endpoints, `false` for local MinIO |

### Generating secrets

```sh
# JWT_SECRET (64-char hex)
openssl rand -hex 32

# ADMIN_PASSWORD (32-char base64)
openssl rand -base64 24
```

### Optional values

| Variable | Default | Description |
|---|---|---|
| `API_PORT` | `8080` | Port the API listens on |
| `API_HOST` | `0.0.0.0` | Bind address |
| `JWT_EXPIRY` | `24h` | Token lifetime |
| `BASE_URL` | `http://localhost:3000` | Public URL (used for OAuth redirects) |
| `STORAGE_REGION` | `us-east-1` | S3 region |
| `MAX_UPLOAD_SIZE` | `52428800` | Max file upload in bytes (default 50 MB) |
| `LOG_LEVEL` | `debug` | `debug`, `info`, `warn`, `error` |
| `LOG_FORMAT` | `text` | `text` or `json` |
| `DISCORD_CLIENT_ID` | _(empty)_ | Discord OAuth app ID (leave empty to disable) |
| `DISCORD_CLIENT_SECRET` | _(empty)_ | Discord OAuth app secret |
| `DISCORD_REDIRECT_URI` | | OAuth callback URL |

## 2. Database Setup

Create the database and user in PostgreSQL:

```sql
CREATE USER taskwondo WITH PASSWORD 'your-password';
CREATE DATABASE taskwondo OWNER taskwondo;
```

Migrations run automatically on startup. To run them without starting the server:

```sh
bin/taskwondo -migrate-only
```

## 3. Object Storage

Create a bucket in your S3-compatible storage matching the `STORAGE_BUCKET` value. For MinIO:

```sh
mc mb myminio/taskwondo-attachments
```

## 4. Running the API

Source the environment and start the binary:

```sh
set -a && source .env && set +a
bin/taskwondo
```

The API listens on `API_HOST:API_PORT` (default `0.0.0.0:8080`). On first start it runs migrations and seeds the admin user.

To run as a systemd service, create `/etc/systemd/system/taskwondo-api.service`:

```ini
[Unit]
Description=Taskwondo API
After=network.target postgresql.service

[Service]
Type=simple
EnvironmentFile=/opt/taskwondo/.env
ExecStart=/opt/taskwondo/bin/taskwondo
Restart=on-failure
User=taskwondo
WorkingDirectory=/opt/taskwondo

[Install]
WantedBy=multi-user.target
```

## 5. Web Frontend

The `html/` directory contains the built frontend. Serve it with Nginx (or any static file server) and proxy `/api/` requests to the API.

The included `nginx.conf` is a working reference configuration. Key points:

- Serves static files from the `html/` directory
- Proxies `/api/` and health endpoints (`/healthz`, `/readyz`) to the API backend
- SPA fallback: all non-file routes serve `index.html`
- Static asset caching with `Cache-Control: public, immutable`
- Upload limit set to 55 MB (`client_max_body_size`)

Adjust `proxy_pass http://api:8080` to match your API address (e.g. `http://127.0.0.1:8080`).

Example Nginx setup:

```sh
cp nginx.conf /etc/nginx/sites-available/taskwondo
ln -s /etc/nginx/sites-available/taskwondo /etc/nginx/sites-enabled/
# Edit the file: update root path, proxy_pass address, server_name
nginx -t && systemctl reload nginx
```

## 6. Health Checks

- **Liveness:** `GET /healthz` — returns `200` if the process is running
- **Readiness:** `GET /readyz` — returns `200` when the database is reachable
