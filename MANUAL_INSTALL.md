# Taskwondo Manual Installation

This guide covers installing Taskwondo without Docker Compose. You'll need to provide your own PostgreSQL, NATS, and S3-compatible storage (MinIO, AWS S3, etc.) and a reverse proxy for the web frontend.

## Prerequisites

- PostgreSQL 15+
- NATS with JetStream enabled
- S3-compatible object storage (MinIO, AWS S3, DigitalOcean Spaces, etc.)
- Nginx (or any web server that can serve static files and reverse-proxy)

## Download

Download the server bundle (`taskwondo-server-*.tar.gz`) from the [Releases](https://github.com/marcoshack/taskwondo/releases) page. The [Dev Build](https://github.com/marcoshack/taskwondo/releases/tag/dev) release always has the latest build from `main`.

## Archive Contents

```
taskwondo-server-<version>/
  bin/taskwondo-api          # API server (static Linux binary)
  bin/taskwondo-worker             # Background worker (static Linux binary)
  html/                  # Web frontend (static files)
  nginx.conf             # Nginx site configuration (reference)
  .env.template          # Configuration template
  install.sh             # Setup script
```

## 1. Configuration

Generate a `.env` file from the template:

```sh
./install.sh --manual-setup
```

This walks you through the required settings interactively, generating secrets automatically. Use `-y` for non-interactive mode with auto-generated defaults:

```sh
./install.sh --manual-setup -y
```

Then edit `.env` to set your database URL, storage credentials, and NATS URL.

To see all available configuration variables with descriptions:

```sh
./install.sh --manual-setup-info
```

### Generating secrets

```sh
# JWT_SECRET (64-char hex)
openssl rand -hex 32

# ADMIN_PASSWORD (32-char base64)
openssl rand -base64 24
```

## 2. Database Setup

Create the database and user in PostgreSQL:

```sql
CREATE USER taskwondo WITH PASSWORD 'your-password';
CREATE DATABASE taskwondo OWNER taskwondo;
```

Migrations run automatically on startup. To run them without starting the server:

```sh
bin/taskwondo-api -migrate-only
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
bin/taskwondo-api
```

The API listens on `API_HOST:API_PORT` (default `0.0.0.0:8080`). On first start it runs migrations and seeds the admin user.

To run as a systemd service, create `/etc/systemd/system/taskwondo-api.service`:

```ini
[Unit]
Description=Taskwondo API
After=network.target postgresql.service
Requires=taskwondo-worker.service

[Service]
Type=simple
EnvironmentFile=/opt/taskwondo/.env
ExecStart=/opt/taskwondo/bin/taskwondo-api
Restart=on-failure
User=taskwondo
WorkingDirectory=/opt/taskwondo

[Install]
WantedBy=multi-user.target
```

## 5. Running the Worker

The worker processes background jobs (email notifications, stats aggregation). It requires a running NATS server with JetStream enabled.

```sh
set -a && source .env && set +a
bin/taskwondo-worker
```

To run as a systemd service, create `/etc/systemd/system/taskwondo-worker.service`:

```ini
[Unit]
Description=Taskwondo Worker
After=network.target postgresql.service

[Service]
Type=simple
EnvironmentFile=/opt/taskwondo/.env
ExecStart=/opt/taskwondo/bin/taskwondo-worker
Restart=on-failure
User=taskwondo
WorkingDirectory=/opt/taskwondo

[Install]
WantedBy=multi-user.target
```

## 6. Web Frontend

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

## 7. Health Checks

- **Liveness:** `GET /healthz` — returns `200` if the process is running
- **Readiness:** `GET /readyz` — returns `200` when the database is reachable
