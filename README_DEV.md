# Taskwondo — Development Setup

## Prerequisites

[mise](https://mise.jdx.dev) is the only development tool you install yourself. `mise.toml`
pins Go, Node, npm and `air`, and `make setup` installs them from it.

```bash
curl https://mise.run | sh   # other install methods: https://mise.jdx.dev/getting-started.html
```

The two things mise cannot provide:

- Docker (container runtime + CLI) and the Docker Compose plugin — see [Docker Setup](#docker-setup) below
- `openssl`, which `install.sh` uses to generate secrets (preinstalled on macOS and most Linux)

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
make setup   # install the toolchain from mise.toml, generate .env (via install.sh), configure git hooks
make dev     # starts Postgres + MinIO + NATS + Mailpit + Ollama + API (hot-reload) + Vite + Worker
```

`make setup` runs `./install.sh --manual-setup -y` automatically when `.env` is
missing. To regenerate `.env` from scratch, delete it and re-run `make setup`
(or run `./install.sh --manual-setup -y` directly).

It ends by offering to activate mise in your shell — see [Toolchain pins](#toolchain-pins).

## Running Tests

```bash
make test                      # Go tests + frontend build
make test-e2e                  # Playwright E2E tests (fully containerized)
```

## Toolchain pins

**`mise.toml` at the repository root is the pin.** It names every runtime and CLI tool
the repository builds with — Go, Node, npm, `air` — at an exact version, and `make setup`
installs all of them from it.

### How the pinned tools get on PATH

The root Makefile puts `mise bin-paths` in front of `PATH` for every target, so `go`,
`node`, `npm` and `air` resolve to the pinned versions in a shell where `mise activate`
has never run — which is most shells, and every shell that was already open when setup
ran. Sub-makes inherit the exported `PATH`, so `make build-mcp-linux` builds with the
same Go as `make build-worker`.

Absent mise the block is a no-op. CI installs its runtimes with `setup-go` and
`setup-node`, and the check below is what holds those to the same pins.

### Activating mise in your shell

The Makefile's `PATH` block covers `make` targets and nothing else. Everything you run
yourself inside the checkout — `cd api && go test ./...`, `cd web && npm run lint`,
`npx playwright test`, an editor's language server — still gets whatever Go and Node the
machine carries. `mise activate` closes that gap: it hooks the shell prompt, so entering
this directory selects the pinned versions and leaving it puts the previous ones back.
It renders nothing itself, so it composes with starship or any other prompt.

`make setup` ends by **offering** to set that up, and `make setup-shell` does the same
thing on its own. Both call `scripts/setup-shell.sh`, which is the only part of setup
that writes outside the repository — so it asks first, backs the file up as
`<rc>.taskwondo.bak`, and appends a marked block guarded by `command -v mise`:

```bash
# >>> mise (added by taskwondo scripts/setup-shell.sh) >>>
command -v mise >/dev/null 2>&1 && eval "$(mise activate zsh)"
# <<< mise <<<
```

It picks the startup file from `$SHELL` — `${ZDOTDIR:-$HOME}/.zshrc`, `~/.bashrc`
(`~/.bash_profile` when that is the only one, as on macOS), or
`~/.config/fish/config.fish` — and declining costs nothing but the same lines printed to
paste yourself. Re-running is safe: a startup file that already mentions `mise activate`
is left alone. Without a terminal to ask at (CI, a pipe) it prints the instructions and
exits 0, so `make setup` never blocks on it.

| Flag | What it does |
|---|---|
| `--yes` | append without asking |
| `--check` | report whether the shell activates mise; exit 1 when it does not |
| `--print` | print the block and exit, changing nothing |

### The lockfile

`mise.toml` sets `lockfile = true`, so `mise.lock` — committed — records the download
URL, size and SHA256 of every tool for all seven platforms mise supports. An install
that gets served something other than what was pinned fails on the checksum instead of
building with it. This is the same posture `web/.npmrc` already takes for npm packages.

Go and Node are the two that carry checksums, since mise downloads their tarballs
itself. `npm` and `air` install through the npm registry and `go install`, which verify
their own downloads (registry integrity hashes, and Go's checksum database), so the lock
records only their version.

**Run `mise lock` after changing any version in `mise.toml`** — `make check-toolchain`
fails if you don't. `mise install` refreshes the entry for your own platform only, which
is why the explicit command exists; `mise lock --platform linux-x64` targets one, and
`mise lock --bump` re-resolves fuzzy selectors (this repository pins exactly, so it is
a no-op here).

### The consistency check

A Go, Node or npm version is declared in more places than `mise.toml`, and nothing used
to hold them together. `scripts/check-toolchain.sh` does, and it runs as a prerequisite
(`make check-toolchain`) of every build and test target, so a disagreement is a
sub-second failure at the top of the run rather than a broken build somewhere downstream:

| Where | What must hold |
|---|---|
| `api/go.mod`, `mcp/go.mod` | `toolchain goX.Y.Z` equals the `go` pin; the `go X.Y.Z` directive does not exceed it |
| `.github/workflows/ci.yml` | every `go-version:` and `node-version:` is a prefix of the matching pin |
| `docker/Dockerfile.*` | each `FROM golang:` / `FROM node:` tag is a prefix of the matching pin — `golang:1.25` for a `1.25.x` pin |
| `docker/Dockerfile.web` | `ARG NPM_VERSION` equals the `npm` pin (CI greps that line to install npm) |
| `mise.lock` | every tool's locked version equals its pin, and the file exists at all |
| the running `go`, `node` and `npm` | equal to the pins, checked only where mise is installed, since that is where the pins are meant to be in force |

**Change a version in `mise.toml` and change everything it lists in the same commit.**
The check names each file it disagrees with and what it found there.

One ordering note in `mise.toml`: `"npm:npm"` is declared **before** `node` on purpose.
mise assembles `PATH` in declaration order, and node's own bundled npm (10.x on Node 22)
shadows the pin when it comes first. The npm pin matters because `web/.npmrc` sets
`min-release-age`, the supply-chain cooldown npm silently ignores before 11.10.

## Architecture

See [AGENTS.md](AGENTS.md) for full architecture notes and conventions.
