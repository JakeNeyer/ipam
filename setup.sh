#!/usr/bin/env bash
# IPAM local setup — clones the repo (if needed), builds the image, starts
# Postgres + IPAM via Docker.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/JakeNeyer/ipam/main/setup.sh | bash
#   — or —
#   git clone https://github.com/JakeNeyer/ipam.git && cd ipam && bash setup.sh

set -euo pipefail

REPO_URL="https://github.com/JakeNeyer/ipam.git"

# ── Configurable defaults (override with env vars) ──────────────────────────
IPAM_DB_NAME="${IPAM_DB_NAME:-ipam-db}"
IPAM_APP_NAME="${IPAM_APP_NAME:-ipam}"
IPAM_DB_USER="${IPAM_DB_USER:-ipam}"
IPAM_DB_PASS="${IPAM_DB_PASS:-ipam}"
IPAM_DB_PORT="${IPAM_DB_PORT:-5432}"
IPAM_PORT="${IPAM_PORT:-8080}"
IPAM_IMAGE="${IPAM_IMAGE:-ipam:local}"
POSTGRES_IMAGE="${POSTGRES_IMAGE:-postgres:16-alpine}"

# ── Helpers ──────────────────────────────────────────────────────────────────
info()  { printf '\033[1;34m▸ %s\033[0m\n' "$*"; }
ok()    { printf '\033[1;32m✓ %s\033[0m\n' "$*"; }
err()   { printf '\033[1;31m✗ %s\033[0m\n' "$*" >&2; }

# ── Preflight ────────────────────────────────────────────────────────────────
if ! command -v docker &>/dev/null; then
  err "Docker is not installed. Install it from https://docs.docker.com/get-docker/ and try again."
  exit 1
fi

if ! docker info &>/dev/null 2>&1; then
  err "Docker daemon is not running. Start Docker and try again."
  exit 1
fi

if ! command -v git &>/dev/null; then
  err "Git is not installed. Install it and try again."
  exit 1
fi

# ── Ensure we have the repo source ──────────────────────────────────────────
# When run from an existing checkout, use it. Otherwise clone into a temp dir.
CLEANUP_REPO=""
if [ -f "Dockerfile" ]; then
  REPO_DIR="$(pwd)"
elif [ -n "${BASH_SOURCE[0]:-}" ] && [ -f "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/Dockerfile" ]; then
  REPO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
else
  info "Cloning $REPO_URL …"
  REPO_DIR="$(mktemp -d)"
  CLEANUP_REPO="$REPO_DIR"
  git clone --depth 1 "$REPO_URL" "$REPO_DIR"
  ok "Cloned."
fi
cd "$REPO_DIR"

# Clean up temp clone on exit (if we created one)
cleanup() { [ -n "$CLEANUP_REPO" ] && rm -rf "$CLEANUP_REPO"; }
trap cleanup EXIT

# ── Clean up any previous run ────────────────────────────────────────────────
for name in "$IPAM_APP_NAME" "$IPAM_DB_NAME"; do
  if docker ps -a --format '{{.Names}}' | grep -qx "$name"; then
    info "Removing existing container: $name"
    docker rm -f "$name" >/dev/null
  fi
done

# ── Build IPAM image ─────────────────────────────────────────────────────────
info "Building IPAM image ($IPAM_IMAGE) …"
docker build -t "$IPAM_IMAGE" .
ok "Image built."

# ── Start Postgres ───────────────────────────────────────────────────────────
info "Starting Postgres ($POSTGRES_IMAGE) on port $IPAM_DB_PORT …"
docker run -d --name "$IPAM_DB_NAME" \
  -e POSTGRES_USER="$IPAM_DB_USER" \
  -e POSTGRES_PASSWORD="$IPAM_DB_PASS" \
  -e POSTGRES_DB="$IPAM_DB_USER" \
  -p "$IPAM_DB_PORT":5432 \
  "$POSTGRES_IMAGE" >/dev/null

# Wait for Postgres to accept connections
info "Waiting for Postgres to be ready …"
retries=0
until docker exec "$IPAM_DB_NAME" pg_isready -U "$IPAM_DB_USER" -q 2>/dev/null; do
  retries=$((retries + 1))
  if [ "$retries" -ge 30 ]; then
    err "Postgres did not become ready in time."
    exit 1
  fi
  sleep 1
done
ok "Postgres is ready."

# ── Start IPAM ───────────────────────────────────────────────────────────────
DATABASE_URL="postgres://${IPAM_DB_USER}:${IPAM_DB_PASS}@host.docker.internal:${IPAM_DB_PORT}/${IPAM_DB_USER}?sslmode=disable"

info "Starting IPAM on port $IPAM_PORT …"
docker run -d --name "$IPAM_APP_NAME" \
  -e DATABASE_URL="$DATABASE_URL" \
  -e PORT=8080 \
  -p "$IPAM_PORT":8080 \
  "$IPAM_IMAGE" >/dev/null

# Brief pause for the app to start
sleep 2

if docker ps --format '{{.Names}}' | grep -qx "$IPAM_APP_NAME"; then
  ok "IPAM is running."
else
  err "IPAM container failed to start. Check logs with: docker logs $IPAM_APP_NAME"
  exit 1
fi

# ── Done ─────────────────────────────────────────────────────────────────────
echo ""
ok "Setup complete!"
echo ""
echo "  Open http://localhost:${IPAM_PORT} to get started."
echo "  Complete the initial setup to create your admin account."
echo ""
echo "  Useful commands:"
echo "    docker logs -f $IPAM_APP_NAME      # follow app logs"
echo "    docker logs -f $IPAM_DB_NAME       # follow database logs"
echo "    docker stop $IPAM_APP_NAME $IPAM_DB_NAME   # stop both containers"
echo "    docker rm -f $IPAM_APP_NAME $IPAM_DB_NAME  # remove both containers"
echo ""
