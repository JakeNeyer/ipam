#!/usr/bin/env bash
# Seed the IPAM API with test data. Run after the server is started (e.g. go run .).
#
# Usage: ./scripts/seed.sh [BASE_URL]
# Example: ./scripts/seed.sh
#          ./scripts/seed.sh http://localhost:8011
#
# Requires: curl, python3. Running multiple times will create duplicate records.

set -e

BASE_URL="${1:-http://localhost:8011}"
API="${BASE_URL}/api"

# Extract JSON field (e.g. id) from stdin. Usage: echo '{"id":"x"}' | json_get id
json_get() { python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get(sys.argv[1], ''))" "$1" 2>/dev/null || true; }

echo "Waiting for API at ${API}..."
for i in 1 2 3 4 5 6 7 8 9 10; do
  if curl -sf "${API}/environments" >/dev/null 2>&1; then
    echo "API is up."
    break
  fi
  if [ "$i" -eq 10 ]; then
    echo "API did not become ready. Is the server running on ${BASE_URL}?"
    exit 1
  fi
  sleep 1
done

echo "Creating environments..."
PROD_RESP=$(curl -sf -X POST "${API}/environments" -H "Content-Type: application/json" -d '{"name":"Production"}')
PROD_ID=$(echo "$PROD_RESP" | json_get id)
echo "  Production -> ${PROD_ID}"

STAGING_RESP=$(curl -sf -X POST "${API}/environments" -H "Content-Type: application/json" -d '{"name":"Staging"}')
STAGING_ID=$(echo "$STAGING_RESP" | json_get id)
echo "  Staging -> ${STAGING_ID}"

echo "Creating blocks (non-contiguous per env to test suggested CIDR ranges)..."
# Production: 10.0.0.0/16, 10.2.0.0/16, 10.4.0.0/16 — gaps at 10.1.x, 10.3.x, 10.5.x, etc.
curl -sf -X POST "${API}/blocks" -H "Content-Type: application/json" \
  -d "{\"name\":\"prod-vpc\",\"cidr\":\"10.0.0.0/16\",\"environment_id\":\"${PROD_ID}\"}" >/dev/null
echo "  prod-vpc (10.0.0.0/16) in Production"

curl -sf -X POST "${API}/blocks" -H "Content-Type: application/json" \
  -d "{\"name\":\"prod-dmz\",\"cidr\":\"10.2.0.0/16\",\"environment_id\":\"${PROD_ID}\"}" >/dev/null
echo "  prod-dmz (10.2.0.0/16) in Production"

curl -sf -X POST "${API}/blocks" -H "Content-Type: application/json" \
  -d "{\"name\":\"prod-data\",\"cidr\":\"10.4.0.0/16\",\"environment_id\":\"${PROD_ID}\"}" >/dev/null
echo "  prod-data (10.4.0.0/16) in Production"

# Staging: 10.1.0.0/16, 10.3.0.0/16, 10.5.0.0/16 — gaps at 10.2.x, 10.4.x, etc.
curl -sf -X POST "${API}/blocks" -H "Content-Type: application/json" \
  -d "{\"name\":\"staging-vpc\",\"cidr\":\"10.1.0.0/16\",\"environment_id\":\"${STAGING_ID}\"}" >/dev/null
echo "  staging-vpc (10.1.0.0/16) in Staging"

curl -sf -X POST "${API}/blocks" -H "Content-Type: application/json" \
  -d "{\"name\":\"staging-test\",\"cidr\":\"10.3.0.0/16\",\"environment_id\":\"${STAGING_ID}\"}" >/dev/null
echo "  staging-test (10.3.0.0/16) in Staging"

curl -sf -X POST "${API}/blocks" -H "Content-Type: application/json" \
  -d "{\"name\":\"staging-dev\",\"cidr\":\"10.5.0.0/16\",\"environment_id\":\"${STAGING_ID}\"}" >/dev/null
echo "  staging-dev (10.5.0.0/16) in Staging"

curl -sf -X POST "${API}/blocks" -H "Content-Type: application/json" \
  -d '{"name":"orphan-block","cidr":"192.168.0.0/24"}' >/dev/null
echo "  orphan-block (192.168.0.0/24) [no environment]"

echo "Creating allocations..."
curl -sf -X POST "${API}/allocations" -H "Content-Type: application/json" \
  -d '{"name":"prod-web","block_name":"prod-vpc","cidr":"10.0.0.0/24"}' >/dev/null
echo "  prod-web 10.0.0.0/24 in prod-vpc"

curl -sf -X POST "${API}/allocations" -H "Content-Type: application/json" \
  -d '{"name":"prod-db","block_name":"prod-vpc","cidr":"10.0.2.0/24"}' >/dev/null
echo "  prod-db 10.0.2.0/24 in prod-vpc (gap 10.0.1.0/24 for bin-pack demo)"

curl -sf -X POST "${API}/allocations" -H "Content-Type: application/json" \
  -d '{"name":"staging-app","block_name":"staging-vpc","cidr":"10.1.0.0/24"}' >/dev/null
echo "  staging-app 10.1.0.0/24 in staging-vpc"

curl -sf -X POST "${API}/allocations" -H "Content-Type: application/json" \
  -d '{"name":"orphan-subnet","block_name":"orphan-block","cidr":"192.168.0.0/26"}' >/dev/null
echo "  orphan-subnet 192.168.0.0/26 in orphan-block"

echo "Done. Seed data created at ${BASE_URL}"
