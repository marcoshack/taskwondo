#!/usr/bin/env bash
# Print a checksum manifest of every object in the Taskwondo object store.
#
# Output: one line per object, "<sha256>  <size-bytes>  <bucket>/<key>", sorted.
# Take one before changing the storage service and one after; `diff` the two.
# See docs/minio-to-silo-migration.md.
#
# Usage (on the host, from the directory with docker-compose.yml):
#   scripts/storage-snapshot.sh                 > storage-before.txt
#   scripts/storage-snapshot.sh <container>     > storage-before.txt
#
# With no argument the container is the `minio` service of the compose project
# in the current directory (set COMPOSE_FILE if you use extra -f files). Works
# against both MinIO and Silo server images: it runs the `mc` bundled in the
# server container, logs in with the container's own MINIO_ROOT_* variables and
# never prints them. Every object is read in full, so expect it to take roughly
# as long as reading the whole bucket once.
set -euo pipefail

container=${1:-$(docker compose ps -q minio)}
if [ -z "$container" ]; then
    echo "storage-snapshot: no running 'minio' service here; pass the container name" >&2
    exit 1
fi

docker exec -i "$container" sh -s <<'EOS'
set -e
mc alias set snap http://localhost:9000 "$MINIO_ROOT_USER" "$MINIO_ROOT_PASSWORD" >/dev/null
trap 'mc alias remove snap >/dev/null 2>&1; rm -f /tmp/snap-obj' EXIT
# `mc ls snap` prints "[date] 0B <bucket>/"; the bucket is the last field.
mc ls snap | while IFS= read -r line; do
    b=${line##* }
    mc find "snap/${b%/}"
done | sort | while IFS= read -r p; do
    mc cat "$p" > /tmp/snap-obj
    printf '%s  %s  %s\n' "$(sha256sum /tmp/snap-obj | cut -c1-64)" "$(wc -c < /tmp/snap-obj | tr -d ' ')" "${p#snap/}"
done
EOS
