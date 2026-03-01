#!/usr/bin/env bash
set -euo pipefail

# =============================================================================
# Taskwondo Setup Script
# =============================================================================
# Generates .env from .env.template, manages Docker Compose deployment,
# and handles backup/restore.
#
# Run ./install.sh or ./install.sh -h for usage.

GITHUB_RAW_URL="https://raw.githubusercontent.com/marcoshack/taskwondo/main"
BASE_URL=""
TARGET_DIR="."
NON_INTERACTIVE=false
IMPORT_FILE=""
DO_EXPORT=false
DO_DOCKER=false
DO_MANUAL_SETUP=false
DO_MANUAL_INFO=false

usage() {
    cat <<EOF
Usage: $(basename "$0") <mode> [options]

Modes:
  --docker               Docker Compose setup — generates .env, downloads
                         docker-compose.yml if needed, and starts services
  --manual-setup         Generate .env for standalone deployment (no Docker required)
  --manual-setup-info    Show all configuration variables with descriptions
  --export               Export database and attachments to a backup archive
  --import FILE          Import data from a backup archive

Options:
  --url URL              Base URL for downloading files
                         (default: $GITHUB_RAW_URL)
  --dir DIR              Target directory (default: current directory)
  -y                     Non-interactive mode: auto-generate all values, skip prompts
  -h, --help             Show this help message

Examples:
  $(basename "$0") --docker                  # Interactive Docker setup
  $(basename "$0") --docker -y               # Non-interactive Docker setup
  $(basename "$0") --manual-setup            # Generate .env for manual deployment
  $(basename "$0") --manual-setup-info       # Show configuration reference
  $(basename "$0") --export                  # Backup database and attachments
  $(basename "$0") --import backup.tar.gz    # Restore from backup
EOF
    exit 0
}

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

# --- Argument parsing ---

while [[ $# -gt 0 ]]; do
    case "$1" in
        --docker)          DO_DOCKER=true; shift ;;
        --manual-setup)    DO_MANUAL_SETUP=true; shift ;;
        --manual-setup-info) DO_MANUAL_INFO=true; shift ;;
        --export)          DO_EXPORT=true; shift ;;
        --import)
            if [[ $# -lt 2 || "$2" == -* ]]; then
                error "--import requires a backup archive path. Example: install.sh --import backup/taskwondo-export-20260228-0032.tar.gz"
            fi
            IMPORT_FILE="$2"; shift 2 ;;
        --url)     BASE_URL="$2"; shift 2 ;;
        --dir)     TARGET_DIR="$2"; shift 2 ;;
        -y)        NON_INTERACTIVE=true; shift ;;
        -h|--help) usage ;;
        *)      echo "Unknown option: $1"; echo; usage ;;
    esac
done

BASE_URL="${BASE_URL:-$GITHUB_RAW_URL}"

# If no mode specified, show help
if ! $DO_DOCKER && ! $DO_MANUAL_SETUP && ! $DO_MANUAL_INFO && ! $DO_EXPORT && [[ -z "$IMPORT_FILE" ]]; then
    usage
fi

# =============================================================================
# --manual-setup-info: Show configuration reference (no Docker needed)
# =============================================================================

do_manual_info() {
    local template_file="$TARGET_DIR/.env.template"

    if [[ ! -f "$template_file" ]]; then
        info "Downloading .env.template..."
        download_file "$BASE_URL/.env.template" "$template_file"
    fi

    local required_vars=()
    local optional_vars=()
    local pending_level=""
    local pending_desc=""

    # Parse @manual annotations followed by KEY=value lines
    while IFS= read -r line || [[ -n "$line" ]]; do
        if [[ "$line" =~ ^#\ @manual\ (required|optional)\ (.+)$ ]]; then
            pending_level="${BASH_REMATCH[1]}"
            pending_desc="${BASH_REMATCH[2]}"
            continue
        fi

        if [[ -n "$pending_level" && "$line" =~ ^([A-Z_][A-Z0-9_]*)=(.*)$ ]]; then
            local key="${BASH_REMATCH[1]}"
            local value="${BASH_REMATCH[2]}"
            local default=""

            # Determine default value (skip markers)
            case "$value" in
                __GENERATE_*__|__PROMPT:*__|__DERIVE:*__|__COPY:*__|__OPTIONAL__)
                    default=""
                    ;;
                *)
                    default="$value"
                    ;;
            esac

            local entry="${key}|${pending_desc}|${default}"
            if [[ "$pending_level" == "required" ]]; then
                required_vars+=("$entry")
            else
                optional_vars+=("$entry")
            fi
        fi

        pending_level=""
        pending_desc=""
    done < "$template_file"

    # Print aligned tables
    echo
    ok "=== Taskwondo Configuration Reference ==="
    echo

    if [[ ${#required_vars[@]} -gt 0 ]]; then
        info "Required:"
        echo
        printf "  \033[1m%-25s %s\033[0m\n" "Variable" "Description"
        printf "  %-25s %s\n" "--------" "-----------"
        for entry in "${required_vars[@]}"; do
            IFS='|' read -r key desc default <<< "$entry"
            printf "  %-25s %s\n" "$key" "$desc"
        done
        echo
    fi

    if [[ ${#optional_vars[@]} -gt 0 ]]; then
        # Compute max default width for alignment
        local max_default=7  # minimum: length of "Default"
        for entry in "${optional_vars[@]}"; do
            IFS='|' read -r _key _desc default <<< "$entry"
            local len=${#default}
            if [[ $len -gt $max_default ]]; then
                max_default=$len
            fi
        done
        local dw=$((max_default + 2))  # padding

        info "Optional:"
        echo
        printf "  \033[1m%-25s %-${dw}s %s\033[0m\n" "Variable" "Default" "Description"
        printf "  %-25s %-${dw}s %s\n" "--------" "-------" "-----------"
        for entry in "${optional_vars[@]}"; do
            IFS='|' read -r key desc default <<< "$entry"
            printf "  %-25s %-${dw}s %s\n" "$key" "${default:--}" "$desc"
        done
        echo
    fi
}

if $DO_MANUAL_INFO; then
    do_manual_info
    exit 0
fi

# =============================================================================
# Shared: .env generation from template
# =============================================================================

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

# Generate .env from template. Processes all markers.
generate_env() {
    local template_file="$TARGET_DIR/.env.template"

    if [[ ! -f "$template_file" ]]; then
        info "Downloading .env.template..."
        download_file "$BASE_URL/.env.template" "$template_file"
    fi

    # Check for existing .env
    local env_file="$TARGET_DIR/.env"
    if [[ -f "$env_file" ]]; then
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

    check_command openssl

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

        # Skip comment lines that document template markers or @manual annotations
        if [[ "$line" =~ ^#.*__ ]] || [[ "$line" =~ ^#\ @manual ]]; then
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

    done < "$template_file"

    # Write .env
    printf '%s' "$output" > "$env_file"
}

# =============================================================================
# --manual-setup: Generate .env without Docker
# =============================================================================

if $DO_MANUAL_SETUP; then
    mkdir -p "$TARGET_DIR"
    generate_env

    echo
    ok "=== .env Generated ==="
    echo
    info "Next steps:"
    echo "  1. Edit .env with your database and storage settings"
    echo "  2. See MANUAL_INSTALL.md for deployment instructions"
    echo "  3. Run: ./install.sh --manual-setup-info  for configuration reference"
    echo
    exit 0
fi

# =============================================================================
# Docker-dependent modes below — check prerequisites
# =============================================================================

info "Checking prerequisites..."

check_command docker

if ! docker compose version &>/dev/null; then
    error "docker compose (v2) is required but not found. Install Docker Compose v2."
fi

printf "  docker:          %s\n" "$(docker --version 2>&1)"
printf "  docker compose:  %s\n" "$(docker compose version 2>&1)"
echo

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

# =============================================================================
# --export
# =============================================================================

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

if $DO_EXPORT; then
    do_export
    exit 0
fi

# =============================================================================
# --import
# =============================================================================

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

if [[ -n "$IMPORT_FILE" ]]; then
    ENV_FILE="$TARGET_DIR/.env"
    COMPOSE_FILE="$TARGET_DIR/docker-compose.yml"

    if [[ -f "$ENV_FILE" && -f "$COMPOSE_FILE" ]]; then
        info "Found existing .env and docker-compose.yml — skipping setup."
        echo
        load_env
        do_import "$IMPORT_FILE"
        exit 0
    elif [[ -f "$ENV_FILE" && ! -f "$COMPOSE_FILE" ]]; then
        info "Found .env but docker-compose.yml is missing — downloading it."
        download_file "$BASE_URL/docker-compose.yml" "$COMPOSE_FILE"
        load_env
        do_import "$IMPORT_FILE"
        exit 0
    else
        info "No .env found — running full setup before import."
        echo
    fi
fi

# =============================================================================
# --docker: Full Docker Compose setup
# =============================================================================

check_command openssl
printf "  openssl:         %s\n" "$(openssl version 2>&1 | head -1)"
echo

mkdir -p "$TARGET_DIR"

# Ensure docker-compose.yml exists
COMPOSE_FILE="$TARGET_DIR/docker-compose.yml"
TEMPLATE_FILE="$TARGET_DIR/.env.template"

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

generate_env

# Summary
echo
ok "=== Taskwondo Setup Complete ==="
echo
printf "  Web UI:  http://localhost:%s\n" "${vars[WEB_PORT]:-3000}"
printf "  API:     http://localhost:%s\n" "${vars[API_PORT]:-8080}"
echo

# Import from backup or show next steps
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
