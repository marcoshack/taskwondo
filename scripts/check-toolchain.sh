#!/usr/bin/env bash
#
# Fail the build when the versions the repository declares for its runtimes
# disagree with each other, or with the runtime actually about to compile it.
#
# mise.toml is the pin: `make setup` installs from it and the root Makefile puts
# it first on PATH. But Go, Node and npm versions are also declared in each
# go.mod, in the CI workflow and in the Dockerfiles, and nothing used to hold
# those together — a `go 1.26` directive with mise still pinned at 1.25 builds
# happily on the machine whose shell has its own Go and fails everywhere else,
# hours later. This runs before every build and test target, so the disagreement
# is a five-second failure at the top instead.
#
# Change a version in mise.toml and you change it in the same commit here:
#
#   */go.mod                  `toolchain goX.Y.Z` must equal the mise pin,
#                             `go X.Y.Z` must not exceed it
#   .github/workflows/ci.yml  `go-version:` / `node-version:` must be a prefix
#                             of the matching pin
#   docker/Dockerfile.*       `FROM golang:` / `FROM node:` tags must be a
#                             prefix of the pin (golang:1.25 for a 1.25.x pin);
#                             `ARG NPM_VERSION` must equal the npm pin
#
# Usage: scripts/check-toolchain.sh

set -euo pipefail

cd "$(dirname "$0")/.."

RED='\033[31m'
YELLOW='\033[33m'
RESET='\033[0m'

problems=()
problem_count=0

problem() {
  problems+=("$1")
  problem_count=$((problem_count + 1))
}

# The pins. Everything below is compared against these three.
mise_go="$(awk -F'"' '/^go[[:space:]]*=/ { print $2; exit }' mise.toml)"
mise_node="$(awk -F'"' '/^node[[:space:]]*=/ { print $2; exit }' mise.toml)"
mise_npm="$(awk -F'"' '/^"npm:npm"[[:space:]]*=/ { print $4; exit }' mise.toml)"

if [ -z "$mise_go" ] || [ -z "$mise_node" ] || [ -z "$mise_npm" ]; then
  printf "${RED}## mise.toml does not pin go, node and npm${RESET}\n" >&2
  exit 1
fi

# True when $1 is $2 or a prefix of it on a dot boundary: 1.25 covers 1.25.14,
# 1.2 does not cover 1.25.14.
covers() {
  [ "$2" = "$1" ] || [ "${2#"$1".}" != "$2" ]
}

# True when $1 <= $2 as dotted versions.
not_newer_than() {
  [ "$1" = "$2" ] || [ "$(printf '%s\n%s\n' "$1" "$2" | sort -V | head -1)" = "$1" ]
}

# Every `<key>: <version>` a YAML file declares, quotes stripped, deduplicated.
yaml_versions() {
  grep -oE "$2:[[:space:]]*\"?[0-9][0-9.]*" "$1" | grep -oE '[0-9][0-9.]*$' | sort -u
}

# --- What the Go modules declare ---

for f in $(ls -d ./*/go.mod 2>/dev/null); do
  [ -f "$f" ] || continue

  toolchain="$(awk '/^toolchain[[:space:]]+go/ { sub(/^go/, "", $2); print $2; exit }' "$f")"
  if [ -n "$toolchain" ] && [ "$toolchain" != "$mise_go" ]; then
    problem "$f declares toolchain go$toolchain, mise.toml pins go $mise_go"
  fi

  godirective="$(awk '/^go[[:space:]]+[0-9]/ { print $2; exit }' "$f")"
  if [ -n "$godirective" ] && ! not_newer_than "$godirective" "$mise_go"; then
    problem "$f requires go $godirective, which is newer than the go $mise_go mise.toml pins"
  fi
done

# --- What CI installs ---

ci=".github/workflows/ci.yml"
if [ -f "$ci" ]; then
  while read -r v; do
    [ -n "$v" ] || continue
    covers "$v" "$mise_go" ||
      problem "$ci installs go-version $v, mise.toml pins go $mise_go"
  done < <(yaml_versions "$ci" go-version)

  while read -r v; do
    [ -n "$v" ] || continue
    covers "$v" "$mise_node" ||
      problem "$ci installs node-version $v, mise.toml pins node $mise_node"
  done < <(yaml_versions "$ci" node-version)
fi

# --- What the images build with ---
#
# CI installs npm by grepping ARG NPM_VERSION out of Dockerfile.web, so that one
# line is what both the image and the CI job use; this keeps it on the pin.

for f in docker/Dockerfile.*; do
  [ -f "$f" ] || continue

  while read -r tag; do
    covers "$tag" "$mise_go" ||
      problem "$f builds on golang:$tag, mise.toml pins go $mise_go"
  done < <(grep -oE '^FROM golang:[0-9][0-9.]*' "$f" | cut -d: -f2 | sort -u)

  while read -r tag; do
    covers "$tag" "$mise_node" ||
      problem "$f builds on node:$tag, mise.toml pins node $mise_node"
  done < <(grep -oE '^FROM node:[0-9][0-9.]*' "$f" | cut -d: -f2 | sort -u)

  while read -r v; do
    [ "$v" = "$mise_npm" ] ||
      problem "$f sets ARG NPM_VERSION=$v, mise.toml pins npm $mise_npm"
  done < <(grep -oE '^ARG NPM_VERSION=[0-9][0-9.]*' "$f" | cut -d= -f2 | sort -u)
done

# --- What is actually on PATH ---
#
# Only where mise is installed, which is where the pins are meant to be in
# force. CI installs its runtimes with setup-go and setup-node instead, and the
# checks above are what hold those to the pin.
if command -v mise >/dev/null 2>&1; then
  stale="run 'mise install' (or 'make setup-shell', if this shell is not using mise)"

  if ! command -v go >/dev/null 2>&1; then
    problem "go is not on PATH — run 'make setup'"
  else
    active_go="$(go env GOVERSION 2>/dev/null | sed 's/^go//')"
    [ "$active_go" = "$mise_go" ] ||
      problem "go on PATH is $active_go, mise.toml pins $mise_go — $stale"
  fi

  if ! command -v node >/dev/null 2>&1; then
    problem "node is not on PATH — run 'make setup'"
  else
    active_node="$(node --version | sed 's/^v//')"
    [ "$active_node" = "$mise_node" ] ||
      problem "node on PATH is $active_node, mise.toml pins $mise_node — $stale"
  fi

  if ! command -v npm >/dev/null 2>&1; then
    problem "npm is not on PATH — run 'make setup'"
  else
    active_npm="$(npm --version)"
    [ "$active_npm" = "$mise_npm" ] ||
      problem "npm on PATH is $active_npm, mise.toml pins $mise_npm — $stale"
  fi
fi

if [ "$problem_count" -gt 0 ]; then
  printf "${RED}## Toolchain mismatch (%d):${RESET}\n" "$problem_count" >&2
  for p in "${problems[@]}"; do
    printf "${RED}  - %s${RESET}\n" "$p" >&2
  done
  printf "\n${YELLOW}mise.toml is the pin: go %s, node %s, npm %s. Bring each line above in step with it, or change the pin.${RESET}\n" \
    "$mise_go" "$mise_node" "$mise_npm" >&2
  exit 1
fi

# Silent on success: this runs in front of every build and test target.
exit 0
