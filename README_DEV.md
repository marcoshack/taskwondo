# Taskwondo — Development Setup

## Prerequisites

- Go 1.25+
- Node.js 22+
- Docker (container runtime + CLI)
- Docker Compose plugin

## Docker Setup

### Linux

Install Docker Engine and the Compose plugin from the [official docs](https://docs.docker.com/engine/install/).

### macOS

On macOS, use [Colima](https://github.com/abiosoft/colima) as a lightweight Docker runtime (no Docker Desktop required).

1. **Install Colima, Docker CLI, and Docker Compose:**

   ```bash
   brew install colima docker docker-compose
   ```

2. **Start Colima:**

   ```bash
   colima start
   ```

3. **Register the Compose plugin** by adding `cliPluginsExtraDirs` to `~/.docker/config.json`:

   ```json
   {
     "cliPluginsExtraDirs": [
       "/opt/homebrew/lib/docker/cli-plugins"
     ]
   }
   ```

4. **Verify:**

   ```bash
   docker version
   docker compose version
   ```

## Getting Started

```bash
./install.sh --manual-setup -y # generate .env with secrets and defaults
make setup                     # configure git hooks
make dev                       # starts Postgres + MinIO + API (hot-reload) + Vite dev server
```

## Running Tests

```bash
make test                      # Go tests + frontend build
make test-e2e                  # Playwright E2E tests (fully containerized)
```

## Architecture

See [AGENTS.md](AGENTS.md) for full architecture notes and conventions.
