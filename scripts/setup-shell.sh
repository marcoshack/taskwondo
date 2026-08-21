#!/usr/bin/env bash
#
# Offer to activate mise in the user's interactive shell, so the pinned
# toolchain (mise.toml) lands on PATH automatically on entering the checkout.
#
# The root Makefile puts `mise bin-paths` on PATH for its own targets, which is
# why `make build` works in a shell that has never heard of mise. Nothing else
# does: `go test ./...`, `cd api && make test`, an editor's language server and
# every script run by hand all get whatever Go and Node the machine carries.
# `mise activate` closes that gap — it hooks the shell prompt, so entering this
# directory selects the pinned versions and leaving it puts the old ones back.
#
# This is the one thing setup cannot do for you without touching a file outside
# the repository, so it asks first and does nothing when declined.
#
# Usage: scripts/setup-shell.sh [--yes] [--check] [--print]
#
#   --yes    append the activation block without asking
#   --check  report whether the shell is activated; exit 1 when it is not
#   --print  print the lines that would be added, and nothing else

set -euo pipefail

cd "$(dirname "$0")/.."

CYAN='\033[36m'
GREEN='\033[32m'
YELLOW='\033[33m'
RED='\033[31m'
RESET='\033[0m'

assume_yes=false
check_only=false
print_only=false

for arg in "$@"; do
  case "$arg" in
    -y|--yes) assume_yes=true ;;
    --check) check_only=true ;;
    --print) print_only=true ;;
    -h|--help) sed -n '3,21p' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
    *) printf "${RED}## unknown argument: %s${RESET}\n" "$arg" >&2; exit 2 ;;
  esac
done

if ! command -v mise >/dev/null 2>&1; then
  printf "${RED}## mise is not installed — run 'make setup' first.${RESET}\n" >&2
  exit 1
fi

# The shell to configure is the login shell, not the one running this script:
# make runs its recipes with /bin/sh whatever the user types commands into.
shell_name="$(basename "${SHELL:-}")"

case "$shell_name" in
  zsh)
    rc="${ZDOTDIR:-$HOME}/.zshrc"
    activate_line='eval "$(mise activate zsh)"'
    ;;
  bash)
    # A macOS Terminal window is a login shell and reads .bash_profile only, so
    # append there when it exists and .bashrc does not.
    if [ ! -f "$HOME/.bashrc" ] && [ -f "$HOME/.bash_profile" ]; then
      rc="$HOME/.bash_profile"
    else
      rc="$HOME/.bashrc"
    fi
    activate_line='eval "$(mise activate bash)"'
    ;;
  fish)
    rc="${XDG_CONFIG_HOME:-$HOME/.config}/fish/config.fish"
    activate_line='mise activate fish | source'
    ;;
  *)
    printf "${YELLOW}## Cannot configure '%s' automatically.${RESET}\n" "${shell_name:-unknown shell}"
    echo "   mise supports it if 'mise activate --help' lists it; add the line it"
    echo "   documents to your shell's startup file: https://mise.jdx.dev/getting-started.html"
    exit 0
    ;;
esac

# mise.run installs into ~/.local/bin, which a startup file may not have on PATH
# yet at the point the activation runs.
mise_dir="$(dirname "$(command -v mise)")"
path_line=""
if [ "$mise_dir" = "$HOME/.local/bin" ]; then
  path_line='export PATH="$HOME/.local/bin:$PATH"'
  [ "$shell_name" = "fish" ] && path_line='fish_add_path "$HOME/.local/bin"'
fi

BEGIN_MARKER="# >>> mise (added by taskwondo scripts/setup-shell.sh) >>>"
END_MARKER="# <<< mise <<<"

block() {
  echo "$BEGIN_MARKER"
  echo "# Puts the toolchain pinned by a project's mise.toml on PATH on entering"
  echo "# its directory, and takes it back off on leaving. Hooks the prompt, so it"
  echo "# coexists with starship and any other prompt: neither renders the other."
  [ -n "$path_line" ] && echo "$path_line"
  if [ "$shell_name" = "fish" ]; then
    echo "command -q mise; and $activate_line"
  else
    echo "command -v mise >/dev/null 2>&1 && $activate_line"
  fi
  echo "$END_MARKER"
}

if $print_only; then
  block
  exit 0
fi

# Already done, either in the file we would edit or by the shell that is running.
if [ -f "$rc" ] && grep -q "mise activate" "$rc"; then
  printf "${GREEN}## Shell already activates mise (%s).${RESET}\n" "$rc"
  exit 0
fi
if [ -n "${MISE_SHELL:-}" ]; then
  printf "${GREEN}## mise is already active in this shell (MISE_SHELL=%s).${RESET}\n" "$MISE_SHELL"
  printf "${YELLOW}## It is not in %s, though — add it there to keep it across sessions.${RESET}\n" "$rc"
fi

if $check_only; then
  printf "${YELLOW}## %s does not activate mise.${RESET}\n" "$rc"
  echo "   Run: ./scripts/setup-shell.sh"
  exit 1
fi

manual_instructions() {
  printf "${CYAN}## To do it yourself later, append this to %s:${RESET}\n" "$rc"
  block | sed 's/^/     /'
  echo "   or run: ./scripts/setup-shell.sh"
}

if ! $assume_yes; then
  # Ask on the terminal rather than stdin: make does not guarantee one, and a
  # non-interactive run (CI, a pipe) must fall through instead of blocking.
  if [ ! -t 1 ] || [ ! -r /dev/tty ]; then
    manual_instructions
    exit 0
  fi
  printf "${CYAN}## Activate mise in your shell? It appends 4 lines to %s${RESET}\n" "$rc"
  printf "   so 'go', 'node' and 'air' resolve to the pinned versions inside this\n"
  printf "   directory in any shell, not only under make. [y/N] "
  read -r reply </dev/tty || reply=""
  case "$reply" in
    [yY]|[yY][eE][sS]) ;;
    *) echo; manual_instructions; exit 0 ;;
  esac
fi

mkdir -p "$(dirname "$rc")"
if [ -f "$rc" ]; then
  backup="$rc.taskwondo.bak"
  cp "$rc" "$backup"
  printf "${CYAN}## Backed up %s -> %s${RESET}\n" "$rc" "$backup"
  # An rc file that does not end in a newline would swallow the first marker.
  [ -n "$(tail -c 1 "$rc")" ] && echo >> "$rc"
fi

{
  echo
  block
} >> "$rc"

# Without trust, the activated shell refuses this repository's mise.toml and
# reports it on every prompt. `make setup` trusts it too; running standalone
# must not leave that behind.
mise trust --quiet 2>/dev/null || true

printf "${GREEN}## Added mise activation to %s${RESET}\n" "$rc"
printf "${GREEN}## Start a new shell, or run: %s${RESET}\n" "$activate_line"
