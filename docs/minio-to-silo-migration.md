# Migrating object storage from MinIO to Silo

Taskwondo stores attachments and avatars in an S3-compatible object store. The bundled
Docker Compose stack used to run `minio/minio` and `minio/mc`; MinIO has stopped
publishing those images, so a host that has to pull them again (a new server, a
`docker image prune`, a rebuilt VM) can no longer start the stack.

The stack now runs [Silo](https://github.com/pgsty/silo) (`pgsty/silo` and `pgsty/mc`),
a community-maintained fork of MinIO. This runbook moves an **existing** Docker Compose
install from MinIO to Silo in place, keeping every stored file. Fresh installs already
get Silo and don't need it.

## What changes and what doesn't

| Stays the same | Changes |
|---|---|
| The data volume (`<project>_miniodata`) — Silo reads MinIO's on-disk format as is | `minio` service image → `pgsty/silo` |
| `MINIO_ROOT_USER` / `MINIO_ROOT_PASSWORD`, ports 9000/9001, the web console | `minio-init` service image → `pgsty/mc` |
| Service names `minio` / `minio-init`, so `STORAGE_ENDPOINT=minio:9000` and every other `.env` value | `minio` healthcheck → `silo healthcheck ready` |
| Bucket name and contents | |

No database change, no API change, no `.env` change.

**Tested path:** MinIO `RELEASE.2025-09-07T16-13-09Z` → Silo `RELEASE.2026-10-26T00-00-00Z`
and back again, with byte-for-byte verification of every object in both directions. If
your MinIO is older or newer than that, the steps are the same, but rehearse on a copy of
the volume first (see [Rehearsing on a copy](#rehearsing-on-a-copy)).

**Downtime:** only the object store restarts. While it is down — the time it takes to tar
the data volume plus a few seconds — attachments and avatars fail to load and uploads
fail. Everything else keeps working. For a volume of a few hundred MB that is well under
a minute.

## Before you start

You need a shell on the Docker host, and free disk space at least the size of the data
volume (for the backup tarball). Run every command from the directory that holds your
`docker-compose.yml`.

1. **Set up the compose command.** If you start the stack with extra files (for example
   `-f docker-compose.ollama.yml`), include them, or later commands will see a different
   stack:

   ```bash
   C="docker compose"                       # or: C="docker compose -f docker-compose.yml -f docker-compose.ollama.yml"
   $C ps minio                              # must show the running minio service
   ```

2. **Record the current version and data volume.**

   ```bash
   docker exec "$($C ps -q minio)" minio --version
   docker inspect "$($C ps -q minio)" --format '{{range .Mounts}}{{.Name}} -> {{.Destination}}{{"\n"}}{{end}}'
   ```

   Note the volume mounted at `/data` (usually `<project>_miniodata`); you need it for
   the backup.

3. **Check which bucket the API uses.** Older installs may use a bucket name other than
   the default `taskwondo-attachments`:

   ```bash
   grep -E '^(STORAGE_BUCKET|MINIO_BUCKET)=' .env
   ```

   The snapshot below covers every bucket, so this is for your own sanity check later.

## 1. Take a checksum baseline

`scripts/storage-snapshot.sh` lists every object in every bucket with its SHA-256 and
size. It runs the `mc` that ships inside the server container and logs in with that
container's own credentials, which it never prints. It reads every object once, so it
takes about as long as reading the whole bucket.

If your install is a git checkout, the script is already in `scripts/`. If you installed
with `install.sh`, download it and read it before running it:

```bash
mkdir -p scripts
curl -fsSL -o scripts/storage-snapshot.sh \
  https://raw.githubusercontent.com/marcoshack/taskwondo/main/scripts/storage-snapshot.sh
less scripts/storage-snapshot.sh
```

Take the baseline while MinIO is still running:

```bash
bash scripts/storage-snapshot.sh "$($C ps -q minio)" > storage-before.txt
wc -l < storage-before.txt            # number of objects
```

The line count should match the object counts the store reports. Keep this file until
the migration is verified.

## 2. Update the compose file

Back it up first:

```bash
cp -p docker-compose.yml docker-compose.yml.bak-pre-silo-$(date -u +%Y%m%d-%H%M%S)
```

**Git checkout:** `git pull` brings in the new `docker-compose.yml`. If you have local
edits to it, resolve them so the `minio` and `minio-init` services end up as on `main`.

**Installed with `install.sh`:** the installer keeps your existing `docker-compose.yml`
on upgrades, so edit it by hand. Copy the two `image:` lines and the healthcheck from the
current
[`docker-compose.yml` on `main`](https://github.com/marcoshack/taskwondo/blob/main/docker-compose.yml)
— they are pinned by tag **and** digest, and the copy on `main` is always the version
that has been tested. At the time of writing the services read:

```yaml
  minio:
    image: pgsty/silo:RELEASE.2026-10-26T00-00-00Z@sha256:7bb2aff6b89efe1cb1303c10fc21d579447434ed06649ac9ec05a5e2dbf30ad7
    command: server /data --console-address ":9001"
    # volumes and environment unchanged
    healthcheck:
      test: ["CMD", "silo", "healthcheck", "ready"]
      # interval / timeout / retries unchanged

  minio-init:
    image: pgsty/mc:RELEASE.2026-10-26T00-00-00Z@sha256:ce62c9053c69ce12a527a322345b5ad9f44b5f8a7b8189f82e516f2e3f30f1d8
    # depends_on, entrypoint and environment unchanged
```

Leave everything else in those services as it is. Use the standard `pgsty/silo` image,
not the `-distroless` variant: the init container and backup commands need the shell and
`mc` that only the standard image has.

Validate the file and pull the images now, so the downtime window doesn't include a
download:

```bash
$C config -q && echo config-ok
$C pull minio minio-init
```

## 3. Back up the data volume

Stop the object store so the files on disk are consistent, then tar the raw volume. This
tarball is your rollback: it restores the volume exactly as MinIO left it.

```bash
VOLUME=taskwondo_miniodata              # the volume from "Before you start", step 2
mkdir -p backup
$C stop minio
docker run --rm -v "$VOLUME":/data:ro -v "$PWD/backup":/backup alpine \
  tar czf "/backup/miniodata-pre-silo-$(date -u +%Y%m%d-%H%M%S).tar.gz" -C /data .
ls -lh backup/
```

Attachments are unavailable from here until step 4 finishes.

## 4. Start Silo on the same volume

```bash
$C up -d --no-deps --wait minio
$C up --no-deps minio-init
```

`--no-deps` restarts only the object store; Postgres, NATS, the API and the web
container keep running. `minio-init` prints `Bucket created successfully` even when the
bucket already exists (`--ignore-existing`) — that is expected and does not touch the
data.

## 5. Verify

1. **Silo is running and healthy.**

   ```bash
   docker exec "$($C ps -q minio)" silo --version
   $C ps minio                           # STATUS shows (healthy)
   ```

2. **Every object survived, byte for byte.**

   ```bash
   bash scripts/storage-snapshot.sh "$($C ps -q minio)" > storage-after.txt
   diff storage-before.txt storage-after.txt && echo "all objects identical"
   ```

   Any output from `diff` means stop and roll back (see below).

3. **The app is ready and the logs are clean.**

   ```bash
   curl -fsS http://localhost:3000/readyz      # or your public URL
   $C logs --since 10m minio | grep -iE 'error|fatal|corrupt|unable' || echo "minio logs clean"
   $C logs --since 10m api | grep -iE 'storage|attachment|avatar' | grep -iE 'err|fail' || echo "api logs clean"
   ```

4. **Reads and writes work from the app.** Open a few work items with attachments
   (images, a PDF, a video if you have one) and check that they preview and download.
   Upload a small file to any work item, open it, then delete it.

5. **Bring the rest of the stack in line with the compose file.** If you pulled other
   changes with git, this also starts the updated services; otherwise it is a no-op:

   ```bash
   $C up -d
   ```

## Rolling back

Silo writes the same on-disk format, so MinIO `RELEASE.2025-09-07` reads a volume Silo
has written to, including objects uploaded after the switch. That makes rollback a plain
image swap:

```bash
cp -p docker-compose.yml.bak-pre-silo-<timestamp> docker-compose.yml
$C up -d --no-deps --wait minio
```

This needs the old MinIO image still present on the host (`docker images minio/minio`):
it can't be pulled again. If it is gone, or the volume itself is damaged, restore the
tarball into the volume first. That discards anything uploaded since the backup:

```bash
$C stop minio
docker run --rm -v "$VOLUME":/data -v "$PWD/backup":/backup alpine \
  sh -c 'rm -rf /data/* /data/.[!.]* && tar xzf /backup/miniodata-pre-silo-<timestamp>.tar.gz -C /data'
$C up -d --no-deps --wait minio
```

## Cleaning up

Once you have run on Silo long enough to be confident, delete the backups:

```bash
rm docker-compose.yml.bak-pre-silo-* storage-before.txt storage-after.txt
rm backup/miniodata-pre-silo-*.tar.gz
```

The old `minio/minio` and `minio/mc` images can be removed with `docker rmi` once no
other stack on the host uses them. Remember they can't be pulled again.

## Rehearsing on a copy

To try the migration without touching the live volume, restore the backup tarball into
a new volume and run Silo against it:

```bash
docker volume create silo-rehearsal
docker run --rm -v silo-rehearsal:/data -v "$PWD/backup":/backup alpine \
  tar xzf /backup/miniodata-pre-silo-<timestamp>.tar.gz -C /data
docker run -d --name silo-rehearsal -v silo-rehearsal:/data \
  -e MINIO_ROOT_USER=rehearsal -e MINIO_ROOT_PASSWORD=rehearsal-secret \
  pgsty/silo:<tag>@<digest> server /data
bash scripts/storage-snapshot.sh silo-rehearsal > storage-rehearsal.txt
diff storage-before.txt storage-rehearsal.txt && echo "rehearsal identical"
docker rm -f silo-rehearsal && docker volume rm silo-rehearsal
```

The rehearsal container doesn't need your real root credentials: any user and password
work, because the snapshot script logs in with whatever the container was started with,
and the stored objects and IAM users read the same either way.

Developers can run the same upgrade and rollback against a throwaway stack, through the
API, with `make test-storage-migration` (see [README_DEV.md](../README_DEV.md)).
