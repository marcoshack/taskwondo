#!/usr/bin/env bash
set -euo pipefail

# =============================================================================
# Taskwondo Setup Script
# =============================================================================
# Creates a .env file from .env.template, auto-generating secrets and
# prompting for user-defined values.
#
# Works in two modes:
#   Local:  when .env.template and docker-compose.yml exist in the target dir
#   Remote: downloads both files from GitHub (or a custom URL)

GITHUB_RAW_URL="https://raw.githubusercontent.com/marcoshack/taskwondo/main"
BASE_URL=""
TARGET_DIR="."
NON_INTERACTIVE=false
IMPORT_FILE=""
DO_EXPORT=false

usage() {
    cat <<EOF
Usage: $(basename "$0") [options]

Sets up Taskwondo by generating a .env file from the configuration template.

Options:
  --url URL        Base URL for downloading files
                   (default: $GITHUB_RAW_URL)
  --dir DIR        Target directory (default: current directory)
  --export         Export database and attachments to a timestamped backup archive
  --import FILE    Import data from a backup archive after setup
  -y               Non-interactive mode: auto-generate all values, skip prompts
  -h, --help       Show this help message

Backup:
  --export creates backup/taskwondo-export-YYYYmmdd-HHMM.tar.gz containing
  a full PostgreSQL dump and all MinIO attachment files. Requires a running
  Taskwondo instance (docker compose up).

  --import restores from a backup archive into a fresh instance. The database
  must be empty (no tables). This is typically used during initial setup.
EOF
    exit 0
}

# --- Argument parsing ---

while [[ $# -gt 0 ]]; do
    case "$1" in
        --url)     BASE_URL="$2"; shift 2 ;;
        --dir)     TARGET_DIR="$2"; shift 2 ;;
        --export)  DO_EXPORT=true; shift ;;
        --import)  IMPORT_FILE="$2"; shift 2 ;;
        -y)        NON_INTERACTIVE=true; shift ;;
        -h|--help) usage ;;
        *)      echo "Unknown option: $1"; usage ;;
    esac
done

BASE_URL="${BASE_URL:-$GITHUB_RAW_URL}"

# --- Helpers ---

info()  { printf "\033[36m%s\033[0m\n" "$*"; }
ok()    { printf "\033[32m%s\033[0m\n" "$*"; }
warn()  { printf "\033[33m%s\033[0m\n" "$*" >&2; }
error() { printf "\033[31mError: %s\033[0m\n" "$*" >&2; exit 1; }

check_command() {
    if ! command -v "$1" &>/dev/null; then
        error "$1 is required but not found. Please install it first."
    fi
}

generate_hex_32() {
    openssl rand -hex 32
}

generate_password() {
    openssl rand -base64 24
}

# Prompt user for a value. Args: description, default_value
# In non-interactive mode, returns the default.
prompt_value() {
    local desc="$1"
    local default="$2"

    if $NON_INTERACTIVE; then
        echo "$default"
        return
    fi

    local prompt_text
    if [[ -n "$default" ]]; then
        prompt_text="  $desc [$default]: "
    else
        prompt_text="  $desc: "
    fi

    local value
    read -rp "$prompt_text" value </dev/tty
    echo "${value:-$default}"
}

download_file() {
    local url="$1"
    local dest="$2"

    if command -v curl &>/dev/null; then
        curl -fsSL "$url" -o "$dest"
    elif command -v wget &>/dev/null; then
        wget -q "$url" -O "$dest"
    else
        error "curl or wget is required to download files."
    fi
}

# Load .env variables into the current shell (for docker compose commands).
load_env() {
    local env_file="$TARGET_DIR/.env"
    if [[ ! -f "$env_file" ]]; then
        error ".env file not found. Run install.sh first to generate it."
    fi
    set -a
    # shellcheck disable=SC1090
    source "$env_file"
    set +a
}

# Wait for the postgres container to accept connections.
wait_for_postgres() {
    info "Waiting for PostgreSQL to be ready..."
    local attempts=0
    while ! docker compose -f "$TARGET_DIR/docker-compose.yml" exec -T postgres \
        pg_isready -U "${POSTGRES_USER:-taskwondo}" &>/dev/null; do
        attempts=$((attempts + 1))
        if [[ $attempts -ge 30 ]]; then
            error "Timed out waiting for PostgreSQL to be ready."
        fi
        sleep 1
    done
}

# --- Export mode (runs before setup, exits early) ---

do_export() {
    load_env

    local timestamp
    timestamp="$(date +%Y%m%d-%H%M)"
    local backup_dir="$TARGET_DIR/backup"
    local tmp_dir
    tmp_dir="$(mktemp -d)"
    local archive="$backup_dir/taskwondo-export-${timestamp}.tar.gz"

    mkdir -p "$backup_dir" "$tmp_dir/db" "$tmp_dir/attachments"

    # 1. Dump database
    info "Dumping database..."
    docker compose -f "$TARGET_DIR/docker-compose.yml" exec -T postgres \
        pg_dump -U "${POSTGRES_USER:-taskwondo}" \
                -d "${POSTGRES_DB:-taskwondo}" \
                --format=custom \
        > "$tmp_dir/db/taskwondo.dump"

    if [[ ! -s "$tmp_dir/db/taskwondo.dump" ]]; then
        rm -rf "$tmp_dir"
        error "Database dump is empty. Is the database running?"
    fi

    # 2. Mirror MinIO bucket
    info "Copying attachment files from MinIO..."
    docker compose -f "$TARGET_DIR/docker-compose.yml" run --rm \
        -v "$tmp_dir/attachments:/backup" \
        --entrypoint /bin/sh \
        minio-init -c "
            mc alias set local http://minio:9000 \$MINIO_ROOT_USER \$MINIO_ROOT_PASSWORD;
            mc mirror --quiet local/\${MINIO_BUCKET:-taskwondo-attachments} /backup/ || true;
        "

    # 3. Create tar.gz archive
    info "Creating backup archive..."
    tar -czf "$archive" -C "$tmp_dir" db attachments

    # Cleanup
    rm -rf "$tmp_dir"

    echo
    ok "=== Export Complete ==="
    echo
    printf "  Archive: %s\n" "$archive"
    printf "  Size:    %s\n" "$(du -h "$archive" | cut -f1)"
    echo
}

# --- Import mode ---

do_import() {
    local import_file="$1"

    if [[ ! -f "$import_file" ]]; then
        error "Import file not found: $import_file"
    fi

    info "Importing data from: $import_file"
    echo

    local tmp_dir
    tmp_dir="$(mktemp -d)"

    # Extract archive
    info "Extracting backup archive..."
    tar -xzf "$import_file" -C "$tmp_dir"

    if [[ ! -f "$tmp_dir/db/taskwondo.dump" ]]; then
        rm -rf "$tmp_dir"
        error "Invalid backup archive: missing db/taskwondo.dump"
    fi

    # Start database and storage services.
    info "Starting database and storage services..."
    cd "$TARGET_DIR"
    docker compose -f docker-compose.yml up -d postgres minio minio-init

    wait_for_postgres

    # Restore database
    info "Restoring database..."
    docker compose -f docker-compose.yml exec -T postgres \
        pg_restore -U "${POSTGRES_USER:-taskwondo}" \
                   -d "${POSTGRES_DB:-taskwondo}" \
                   --no-owner --no-privileges \
        < "$tmp_dir/db/taskwondo.dump"

    # Restore MinIO attachments (if any were exported)
    if [[ -d "$tmp_dir/attachments" ]] && [[ -n "$(ls -A "$tmp_dir/attachments" 2>/dev/null)" ]]; then
        info "Restoring attachment files to MinIO..."
        docker compose -f docker-compose.yml run --rm \
            -v "$tmp_dir/attachments:/backup:ro" \
            --entrypoint /bin/sh \
            minio-init -c "
                mc alias set local http://minio:9000 \$MINIO_ROOT_USER \$MINIO_ROOT_PASSWORD;
                mc mb --ignore-existing local/\${MINIO_BUCKET:-taskwondo-attachments};
                mc mirror --quiet /backup/ local/\${MINIO_BUCKET:-taskwondo-attachments};
            "
    fi

    # Cleanup
    rm -rf "$tmp_dir"

    # Start all services.
    info "Starting all services..."
    docker compose -f docker-compose.yml up -d

    echo
    ok "=== Taskwondo restored from backup and running ==="
    echo
    printf "  Web UI:  http://localhost:%s\n" "${WEB_PORT:-3000}"
    printf "  API:     http://localhost:%s\n" "${API_PORT:-8080}"
    echo
}

# --- Preflight checks ---

info "Checking prerequisites..."

check_command docker

if ! docker compose version &>/dev/null; then
    error "docker compose (v2) is required but not found. Install Docker Compose v2."
fi

printf "  docker:          %s\n" "$(docker --version 2>&1)"
printf "  docker compose:  %s\n" "$(docker compose version 2>&1)"
echo

# --- Export: run and exit ---

if $DO_EXPORT; then
    do_export
    exit 0
fi

# --- Setup requires openssl ---

check_command openssl
printf "  openssl:         %s\n" "$(openssl version 2>&1 | head -1)"
echo

# --- Ensure target directory exists ---

mkdir -p "$TARGET_DIR"

# --- Determine mode: local or remote ---

TEMPLATE_FILE="$TARGET_DIR/.env.template"
COMPOSE_FILE="$TARGET_DIR/docker-compose.yml"

if [[ -f "$TEMPLATE_FILE" && -f "$COMPOSE_FILE" ]]; then
    info "Local mode: using files from $TARGET_DIR"
else
    info "Remote mode: downloading files from $BASE_URL"

    if [[ ! -f "$TEMPLATE_FILE" ]]; then
        info "  Downloading .env.template..."
        download_file "$BASE_URL/.env.template" "$TEMPLATE_FILE"
    fi

    if [[ ! -f "$COMPOSE_FILE" ]]; then
        info "  Downloading docker-compose.yml..."
        download_file "$BASE_URL/docker-compose.yml" "$COMPOSE_FILE"
    fi

    ok "  Files downloaded."
    echo
fi

# --- Check for existing .env ---

ENV_FILE="$TARGET_DIR/.env"

if [[ -f "$ENV_FILE" ]]; then
    if $NON_INTERACTIVE; then
        warn ".env already exists, overwriting (non-interactive mode)."
    else
        read -rp "$(warn '.env already exists. Overwrite? [y/N] ')" confirm </dev/tty
        if [[ ! "$confirm" =~ ^[Yy]$ ]]; then
            info "Aborted. Existing .env was not modified."
            exit 0
        fi
    fi
fi

# --- Parse template and generate .env ---

info "Generating .env..."
echo

# Associative array to store generated values (for __COPY__ and __DERIVE__)
declare -A vars
output="# Taskwondo Configuration"$'\n'
output+="# Generated by install.sh on $(date -u +%Y-%m-%dT%H:%M:%SZ)"$'\n'
in_header=true

while IFS= read -r line || [[ -n "$line" ]]; do
    # Skip the template header block (top comments before first blank line)
    if $in_header; then
        if [[ -z "$line" ]]; then
            in_header=false
            output+=$'\n'
        fi
        continue
    fi

    # Skip comment lines that document template markers
    if [[ "$line" =~ ^#.*__ ]]; then
        continue
    fi

    # Blank lines and comments: pass through
    if [[ -z "$line" || "$line" =~ ^# ]]; then
        output+="$line"$'\n'
        continue
    fi

    # Split KEY=VALUE
    if [[ "$line" =~ ^([A-Z_][A-Z0-9_]*)=(.*) ]]; then
        key="${BASH_REMATCH[1]}"
        value="${BASH_REMATCH[2]}"
    else
        # Not a KEY=VALUE line, pass through
        output+="$line"$'\n'
        continue
    fi

    # Process marker values
    case "$value" in
        __GENERATE_HEX_32__)
            value="$(generate_hex_32)"
            ;;
        __GENERATE_PASSWORD__)
            value="$(generate_password)"
            ;;
        __PROMPT:*__)
            # Extract description and optional default from __PROMPT:desc [default]__
            local_desc="${value#__PROMPT:}"
            local_desc="${local_desc%__}"
            local_default=""
            if [[ "$local_desc" =~ \[([^]]*)\]$ ]]; then
                local_default="${BASH_REMATCH[1]}"
                local_desc="${local_desc% \[*\]}"
            fi
            value="$(prompt_value "$local_desc" "$local_default")"
            ;;
        __COPY:*__)
            # Copy value from another variable
            local_ref="${value#__COPY:}"
            local_ref="${local_ref%__}"
            value="${vars[$local_ref]:-}"
            ;;
        __DERIVE:*__)
            # Evaluate shell expression with current vars
            local_expr="${value#__DERIVE:}"
            local_expr="${local_expr%__}"
            # Substitute ${VAR} references with actual values
            for var_name in "${!vars[@]}"; do
                local_expr="${local_expr//\$\{$var_name\}/${vars[$var_name]}}"
            done
            value="$local_expr"
            ;;
        __OPTIONAL__)
            value=""
            ;;
        # Default: keep value as-is
    esac

    vars["$key"]="$value"
    output+="$key=$value"$'\n'

done < "$TEMPLATE_FILE"

# --- Write .env ---

printf '%s' "$output" > "$ENV_FILE"

# --- Summary ---

echo
ok "=== Taskwondo Setup Complete ==="
echo
printf "  Web UI:  http://localhost:%s\n" "${vars[WEB_PORT]:-3000}"
printf "  API:     http://localhost:%s\n" "${vars[API_PORT]:-8080}"
echo

# --- Import from backup or show next steps ---

if [[ -n "$IMPORT_FILE" ]]; then
    load_env
    do_import "$IMPORT_FILE"
else
    printf "  Admin email:     %s\n" "${vars[ADMIN_EMAIL]:-not set}"
    printf "  Admin password:  %s\n" "${vars[ADMIN_PASSWORD]:-not set}"
    echo
    info "To start Taskwondo:"
    echo "  docker compose -f docker-compose.yml up -d"
    echo
    info "The API will automatically run database migrations and"
    info "seed the admin user on first start."
    echo
fi
