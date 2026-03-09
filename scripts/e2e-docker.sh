#!/usr/bin/env bash
set -euo pipefail

# Isolated E2E test runner — spins up a full Docker Compose stack,
# runs Playwright tests inside a container, and tears everything down.

COMPOSE_FILE="docker-compose.e2e.yml"
PROJECT_NAME="taskwondo-e2e"
COMPOSE="docker compose -f $COMPOSE_FILE -p $PROJECT_NAME"

cleanup() {
    echo ""
    echo "==> Tearing down E2E stack..."
    $COMPOSE down -v --remove-orphans 2>/dev/null || true
}

trap cleanup EXIT

# Pre-clean any leftover stack from a previous crashed run
echo "==> Cleaning up any previous E2E stack..."
$COMPOSE down -v --remove-orphans 2>/dev/null || true

# Clean previous test results (may be owned by root from Docker)
docker run --rm -v "$(pwd)/test/e2e:/e2e" alpine sh -c "rm -rf /e2e/test-results /e2e/playwright-report" 2>/dev/null || true

# Build all images
echo "==> Building images..."
$COMPOSE build

# Start the stack in detached mode, then wait for playwright to finish.
# We can't use --exit-code-from because --abort-on-container-exit (implied)
# kills everything when minio-init exits as a one-shot container.
echo "==> Starting E2E stack..."
$COMPOSE up -d

echo "==> Waiting for Playwright tests to complete..."
set +e
$COMPOSE wait playwright
EXIT_CODE=$?
set -e

echo ""
echo "==> Playwright logs:"
$COMPOSE logs playwright

echo ""
if [ $EXIT_CODE -eq 0 ]; then
    echo "==> E2E tests passed!"
else
    echo "==> E2E tests failed (exit code: $EXIT_CODE)"
fi

if [ -d "test/e2e/playwright-report" ]; then
    echo ""
    echo "Report: test/e2e/playwright-report/index.html"
    echo "Serve it with: make test-e2e-report"
fi

exit $EXIT_CODE
