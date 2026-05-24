# Taskwondo — Development Setup

## Prerequisites

- Go 1.25+
- Node.js 22+ (with npm 11.10+ — run `npm install -g npm@latest` if needed)
- Docker (container runtime + CLI)
- Docker Compose plugin
- `openssl` (for generating secrets — preinstalled on macOS/most Linux)

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
make setup   # verify tools, install air, generate .env (via install.sh), configure git hooks
make dev     # starts Postgres + MinIO + NATS + Mailpit + Ollama + API (hot-reload) + Vite + Worker
```

`make setup` runs `./install.sh --manual-setup -y` automatically when `.env` is
missing. To regenerate `.env` from scratch, delete it and re-run `make setup`
(or run `./install.sh --manual-setup -y` directly).

## Running Tests

```bash
make test                      # Go tests + frontend build
make test-e2e                  # Playwright E2E tests (fully containerized)
```

## Architecture

See [AGENTS.md](AGENTS.md) for full architecture notes and conventions.
