#!/usr/bin/env bash
# Storage migration test: MinIO -> Silo -> rollback to MinIO, through the real API.
#
#   1. Start the stack on the OLD object store and seed it through the HTTP API:
#      attachments of several sizes (one large enough to force a multipart
#      upload), a unicode filename, a deleted attachment, and a user avatar.
#      Record the SHA-256 of every download.
#   2. Recreate only the object-store container with the NEW image on the same
#      data volume (what `docker compose pull && up -d` does on an upgrade),
#      restart the API, verify every download byte-for-byte, then write and
#      delete through the new store.
#   3. Roll back to the OLD image on the volume the new store wrote, and verify
#      everything again, including the objects the new store created.
#
# Usage: scripts/storage-migration-test.sh   (or: make test-storage-migration)
# Override images with OLD_STORAGE_IMAGE / OLD_MC_IMAGE / NEW_STORAGE_IMAGE /
# NEW_MC_IMAGE. KEEP_STACK=1 leaves the stack up afterwards for inspection.
set -euo pipefail

OLD_STORAGE_IMAGE=${OLD_STORAGE_IMAGE:-ghcr.io/marcoshack/taskwondo/minio:RELEASE.2025-09-07T16-13-09Z}
OLD_MC_IMAGE=${OLD_MC_IMAGE:-ghcr.io/marcoshack/taskwondo/mc:RELEASE.2025-08-13T08-35-41Z}
# Default the new images to whatever docker-compose.yml pins, so the test
# always exercises the release that ships.
NEW_STORAGE_IMAGE=${NEW_STORAGE_IMAGE:-$(awk '/^  minio:/{f=1} f&&/image:/{print $2; exit}' docker-compose.yml)}
NEW_MC_IMAGE=${NEW_MC_IMAGE:-$(awk '/^  minio-init:/{f=1} f&&/image:/{print $2; exit}' docker-compose.yml)}

PORT=${MIGRATION_API_PORT:-18089}
API="http://127.0.0.1:$PORT/api/v1"
NS=default
PROJECT=MIG
COMPOSE="docker compose --progress quiet -f docker-compose.storage-migration.yml -p taskwondo-storage-migration"
WORK=$(mktemp -d)
FAILURES=0

cleanup() {
    if [ "${KEEP_STACK:-0}" = 1 ]; then
        echo "==> KEEP_STACK=1: stack left running ($COMPOSE)"
    else
        echo "==> Tearing down"
        STORAGE_IMAGE=x MC_IMAGE=x $COMPOSE down -v --remove-orphans >/dev/null 2>&1 || true
    fi
    rm -rf "$WORK"
}
trap cleanup EXIT

ok()   { echo "  ok   $*"; }
fail() { echo "  FAIL $*"; FAILURES=$((FAILURES + 1)); }
expect_eq() { if [ "$1" = "$2" ]; then ok "$3"; else fail "$3 (got '$1', want '$2')"; fi; }

# --- stack control -----------------------------------------------------------

storage_up() { # $1 storage image, $2 mc image, $3 label
    echo "==> Object store: $3"
    echo "    $1"
    export STORAGE_IMAGE=$1 MC_IMAGE=$2
    $COMPOSE up -d --wait --build postgres nats minio >/dev/null
    $COMPOSE up minio-init --exit-code-from minio-init >/dev/null
    $COMPOSE up -d --wait --force-recreate api >/dev/null
    echo "    running: $($COMPOSE exec -T minio sh -c 'silo --version 2>/dev/null || minio --version' | head -1)"
}

storage_down() {
    $COMPOSE stop minio >/dev/null
    # Anything alarming the store logged about the volume it just served.
    local bad
    bad=$($COMPOSE logs minio 2>&1 | grep -iE "corrupt|unformatted|inconsistent|xl.meta|fatal|panic|unable to" || true)
    if [ -n "$bad" ]; then fail "object store logged problems:"; echo "$bad" | head -20 | sed 's/^/       /'; else ok "object store logs clean"; fi
    $COMPOSE rm -f minio >/dev/null
}

# --- API helpers -------------------------------------------------------------

TOKEN=""
login() {
    TOKEN=$(curl -fsS "$API/auth/login" -H 'Content-Type: application/json' \
        -d '{"email":"admin@migration.test","password":"MigAdminPass123!"}' | jq -r .data.token)
    [ -n "$TOKEN" ] && [ "$TOKEN" != null ] || { echo "login failed"; exit 1; }
}
api() { curl -fsS -H "Authorization: Bearer $TOKEN" "$@"; }

upload() { # $1 item number, $2 file path, $3 content type, $4 filename -> attachment id
    api "$API/$NS/projects/$PROJECT/items/$1/attachments" \
        -F "file=@$2;type=$3;filename=$4" | jq -r .data.id
}

download_sha() { # $1 item number, $2 attachment id -> sha256 or HTTP status on failure
    local out="$WORK/dl" code
    code=$(curl -sS -o "$out" -w '%{http_code}' -H "Authorization: Bearer $TOKEN" \
        "$API/$NS/projects/$PROJECT/items/$1/attachments/$2")
    if [ "$code" = 200 ]; then shasum -a 256 "$out" | cut -c1-64; else echo "HTTP $code"; fi
}

avatar_sha() { # $1 size ("" or large)
    local out="$WORK/av" code
    code=$(curl -sS -o "$out" -w '%{http_code}' -H "Authorization: Bearer $TOKEN" \
        "$API/users/$USER_ID/avatar${1:+?size=$1}")
    if [ "$code" = 200 ]; then shasum -a 256 "$out" | cut -c1-64; else echo "HTTP $code"; fi
}

# MANIFEST lines: "<item> <attachment id> <sha256> <label>"
MANIFEST="$WORK/manifest"

verify_all() { # $1 label
    echo "-- verify attachments ($1)"
    while read -r item id sha label; do
        expect_eq "$(download_sha "$item" "$id")" "$sha" "$label"
    done < "$MANIFEST"
    expect_eq "$(download_sha "$DELETED_ITEM" "$DELETED_ID")" "HTTP 404" "deleted attachment stays gone"
    for item in 1 2 3; do
        expect_eq "$(api "$API/$NS/projects/$PROJECT/items/$item/attachments" | jq '.data | length')" \
            "$(awk -v i="$item" '$1==i' "$MANIFEST" | wc -l | tr -d ' ')" "item $item attachment count"
    done
    echo "-- verify avatar ($1)"
    expect_eq "$(avatar_sha "")" "$AVATAR_SHA" "avatar (default size)"
    expect_eq "$(avatar_sha large)" "$AVATAR_LARGE_SHA" "avatar (large)"
}

# --- fixtures ----------------------------------------------------------------

printf 'Plain text attachment written before the storage migration.\n' > "$WORK/notes.txt"
head -c 3145728 /dev/urandom > "$WORK/medium.bin"      # 3 MiB, single PUT
head -c 41943040 /dev/urandom > "$WORK/large.bin"      # 40 MiB, multipart in minio-go
printf "%%PDF-1.4\n%%migration\n" > "$WORK/doc.pdf"   # small typed file (the API rejects empty files)
printf '{"migrated": true}\n' > "$WORK/data.json"
# A valid 96x96 PNG so the avatar pipeline decodes and re-encodes it.
python3 - "$WORK/avatar.png" <<'EOF'
import struct, sys, zlib
w = h = 96
raw = b"".join(b"\x00" + b"".join(bytes((x * 2 % 256, y * 2 % 256, 128)) for x in range(w)) for y in range(h))
chunk = lambda t, d: struct.pack(">I", len(d)) + t + d + struct.pack(">I", zlib.crc32(t + d) & 0xffffffff)
png = b"\x89PNG\r\n\x1a\n" + chunk(b"IHDR", struct.pack(">IIBBBBB", w, h, 8, 2, 0, 0, 0)) \
      + chunk(b"IDAT", zlib.compress(raw)) + chunk(b"IEND", b"")
open(sys.argv[1], "wb").write(png)
EOF
sha() { shasum -a 256 "$1" | cut -c1-64; }

# --- 0. clean slate ----------------------------------------------------------

echo "==> Old store:  $OLD_STORAGE_IMAGE"
echo "==> New store:  $NEW_STORAGE_IMAGE"
STORAGE_IMAGE=x MC_IMAGE=x $COMPOSE down -v --remove-orphans >/dev/null 2>&1 || true

# --- 1. seed on the old store ------------------------------------------------

storage_up "$OLD_STORAGE_IMAGE" "$OLD_MC_IMAGE" "OLD (seed)"
login
USER_ID=$(api "$API/auth/me" | jq -r '.data.id // .data.user.id')
api "$API/$NS/projects" -H 'Content-Type: application/json' -d "{\"key\":\"$PROJECT\",\"name\":\"Storage migration\"}" >/dev/null
for n in 1 2 3; do
    api "$API/$NS/projects/$PROJECT/items" -H 'Content-Type: application/json' \
        -d "{\"title\":\"Migration item $n\",\"type\":\"task\"}" >/dev/null
done

echo "-- seeding"
add() { # $1 item, $2 file, $3 type, $4 filename, $5 label
    local id; id=$(upload "$1" "$2" "$3" "$4")
    echo "$1 $id $(sha "$2") $5" >> "$MANIFEST"
    ok "uploaded $5"
}
add 1 "$WORK/notes.txt"  text/plain               "notes.txt"              "item1 notes.txt"
add 1 "$WORK/medium.bin" application/octet-stream "medium.bin"             "item1 medium.bin (3 MiB)"
add 2 "$WORK/large.bin"  application/octet-stream "large.bin"              "item2 large.bin (40 MiB multipart)"
add 2 "$WORK/data.json"  application/json         "relatório final ü.json" "item2 unicode filename"
add 3 "$WORK/doc.pdf"    application/pdf          "doc.pdf"                "item3 doc.pdf"
DELETED_ITEM=3
DELETED_ID=$(upload 3 "$WORK/notes.txt" text/plain "to-delete.txt")
api -X DELETE "$API/$NS/projects/$PROJECT/items/3/attachments/$DELETED_ID" >/dev/null
ok "uploaded and deleted to-delete.txt"
api "$API/user/avatar" -F "file=@$WORK/avatar.png;type=image/png" >/dev/null
AVATAR_SHA=$(avatar_sha "")
AVATAR_LARGE_SHA=$(avatar_sha large)
if [[ $AVATAR_SHA == HTTP* || $AVATAR_LARGE_SHA == HTTP* ]]; then
    fail "avatar not readable after upload ($AVATAR_SHA / $AVATAR_LARGE_SHA)"
else
    ok "uploaded avatar"
fi

verify_all "old store, baseline"
storage_down

# --- 2. upgrade: new store on the old volume ---------------------------------

storage_up "$NEW_STORAGE_IMAGE" "$NEW_MC_IMAGE" "NEW (upgrade in place)"
login
verify_all "new store reading old data"

echo "-- writing through the new store"
add 1 "$WORK/data.json"  application/json         "written-by-new.json"    "item1 written-by-new.json"
add 3 "$WORK/large.bin"  application/octet-stream "large-again.bin"        "item3 large-again.bin (multipart, new store)"
# Delete an object the OLD store wrote.
OLD_NOTES_ID=$(awk '$4=="item1" && $5=="notes.txt" {print $2}' "$MANIFEST")
api -X DELETE "$API/$NS/projects/$PROJECT/items/1/attachments/$OLD_NOTES_ID" >/dev/null
expect_eq "$(download_sha 1 "$OLD_NOTES_ID")" "HTTP 404" "delete of an old-store object"
grep -v " $OLD_NOTES_ID " "$MANIFEST" > "$MANIFEST.tmp" && mv "$MANIFEST.tmp" "$MANIFEST"
verify_all "new store after writes"
storage_down

# --- 3. rollback: old store on the volume the new store wrote ----------------

storage_up "$OLD_STORAGE_IMAGE" "$OLD_MC_IMAGE" "OLD (rollback)"
login
verify_all "old store reading new-store data"
storage_down

echo
if [ "$FAILURES" = 0 ]; then
    echo "==> STORAGE MIGRATION TEST PASSED"
else
    echo "==> STORAGE MIGRATION TEST FAILED ($FAILURES failures)"
fi
exit $((FAILURES > 0))
