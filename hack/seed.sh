#!/usr/bin/env bash
# Seed the IPAM API with test data. Run after the server is started (e.g. go run .).
#
# Usage: ./hack/seed.sh [BASE_URL] [API_TOKEN]
# Example: ./hack/seed.sh http://localhost:8011
#          ./hack/seed.sh http://localhost:8011 your-api-token
#          ./hack/seed.sh https://ipam.example.com your-api-token
#
# API_TOKEN is required (create one in Admin > API tokens). All seed requests use Bearer auth.
# For remote HTTPS with self-signed certs: SEED_INSECURE=1 ./hack/seed.sh https://... TOKEN
# Requires: curl, python3. Running multiple times will create duplicate records.

set -e

BASE_URL="${1:-http://localhost:8011}"
API_TOKEN="${2:-}"
API="${BASE_URL}/api"

# Curl options: follow redirects (HTTP→HTTPS), timeouts for remote servers
CURL_OPTS=(-s -L --connect-timeout 10 --max-time 30)
if [ -n "${SEED_INSECURE:-}" ]; then
  CURL_OPTS+=(--insecure)
fi

# Optional Bearer auth for curl. Use: curl "${CURL_OPTS[@]}" -f "${CURL_AUTH[@]}" ...
CURL_AUTH=()
if [ -n "$API_TOKEN" ]; then
  CURL_AUTH=(-H "Authorization: Bearer ${API_TOKEN}")
fi

# Extract JSON field (e.g. id) from stdin. Usage: echo '{"id":"x"}' | json_get id
json_get() { python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get(sys.argv[1], ''))" "$1" 2>/dev/null || true; }

echo "Waiting for API at ${API}..."
for i in 1 2 3 4 5 6 7 8 9 10; do
  if curl "${CURL_OPTS[@]}" -f "${API}/setup/status" >/dev/null 2>&1; then
    echo "API is up."
    break
  fi
  if [ "$i" -eq 10 ]; then
    echo "API did not become ready. Is the server running on ${BASE_URL}?"
    exit 1
  fi
  sleep 1
done

if [ -z "$API_TOKEN" ]; then
  echo "Error: API token required. Create one in Admin > API tokens, then run:"
  echo "  ./hack/seed.sh ${BASE_URL} YOUR_TOKEN"
  exit 1
fi

echo "Creating environments (each requires at least one pool)..."
PROD_RESP=$(curl "${CURL_OPTS[@]}" -f "${CURL_AUTH[@]}" -X POST "${API}/environments" -H "Content-Type: application/json" \
  -d '{"name":"Production","pools":[{"name":"Production pool","cidr":"10.0.0.0/8"}]}')
PROD_ID=$(echo "$PROD_RESP" | json_get id)
PROD_POOL_ID=$(echo "$PROD_RESP" | json_get initial_pool_id)
echo "  Production -> ${PROD_ID} (pool ${PROD_POOL_ID})"

STAGING_RESP=$(curl "${CURL_OPTS[@]}" -f "${CURL_AUTH[@]}" -X POST "${API}/environments" -H "Content-Type: application/json" \
  -d '{"name":"Staging","pools":[{"name":"Staging pool","cidr":"10.0.0.0/8"}]}')
STAGING_ID=$(echo "$STAGING_RESP" | json_get id)
STAGING_POOL_ID=$(echo "$STAGING_RESP" | json_get initial_pool_id)
echo "  Staging -> ${STAGING_ID} (pool ${STAGING_POOL_ID})"

echo "Creating blocks (non-contiguous per env to test suggested CIDR ranges)..."
# Production: 10.0.0.0/16, 10.2.0.0/16, 10.4.0.0/16 — gaps at 10.1.x, 10.3.x, 10.5.x, etc.
curl "${CURL_OPTS[@]}" -f "${CURL_AUTH[@]}" -X POST "${API}/blocks" -H "Content-Type: application/json" \
  -d "{\"name\":\"prod-vpc\",\"cidr\":\"10.0.0.0/16\",\"environment_id\":\"${PROD_ID}\",\"pool_id\":\"${PROD_POOL_ID}\"}" >/dev/null
echo "  prod-vpc (10.0.0.0/16) in Production"

curl "${CURL_OPTS[@]}" -f "${CURL_AUTH[@]}" -X POST "${API}/blocks" -H "Content-Type: application/json" \
  -d "{\"name\":\"prod-dmz\",\"cidr\":\"10.2.0.0/16\",\"environment_id\":\"${PROD_ID}\",\"pool_id\":\"${PROD_POOL_ID}\"}" >/dev/null
echo "  prod-dmz (10.2.0.0/16) in Production"

curl "${CURL_OPTS[@]}" -f "${CURL_AUTH[@]}" -X POST "${API}/blocks" -H "Content-Type: application/json" \
  -d "{\"name\":\"prod-data\",\"cidr\":\"10.4.0.0/16\",\"environment_id\":\"${PROD_ID}\",\"pool_id\":\"${PROD_POOL_ID}\"}" >/dev/null
echo "  prod-data (10.4.0.0/16) in Production"

# Staging: 10.1.0.0/16, 10.3.0.0/16, 10.5.0.0/16 — gaps at 10.2.x, 10.4.x, etc.
curl "${CURL_OPTS[@]}" -f "${CURL_AUTH[@]}" -X POST "${API}/blocks" -H "Content-Type: application/json" \
  -d "{\"name\":\"staging-vpc\",\"cidr\":\"10.1.0.0/16\",\"environment_id\":\"${STAGING_ID}\",\"pool_id\":\"${STAGING_POOL_ID}\"}" >/dev/null
echo "  staging-vpc (10.1.0.0/16) in Staging"

curl "${CURL_OPTS[@]}" -f "${CURL_AUTH[@]}" -X POST "${API}/blocks" -H "Content-Type: application/json" \
  -d "{\"name\":\"staging-test\",\"cidr\":\"10.3.0.0/16\",\"environment_id\":\"${STAGING_ID}\",\"pool_id\":\"${STAGING_POOL_ID}\"}" >/dev/null
echo "  staging-test (10.3.0.0/16) in Staging"

curl "${CURL_OPTS[@]}" -f "${CURL_AUTH[@]}" -X POST "${API}/blocks" -H "Content-Type: application/json" \
  -d "{\"name\":\"staging-dev\",\"cidr\":\"10.5.0.0/16\",\"environment_id\":\"${STAGING_ID}\",\"pool_id\":\"${STAGING_POOL_ID}\"}" >/dev/null
echo "  staging-dev (10.5.0.0/16) in Staging"

curl "${CURL_OPTS[@]}" -f "${CURL_AUTH[@]}" -X POST "${API}/blocks" -H "Content-Type: application/json" \
  -d '{"name":"orphan-block","cidr":"192.168.0.0/24"}' >/dev/null
echo "  orphan-block (192.168.0.0/24) [no environment]"

# Full utilization: entire block allocated (100%)
curl "${CURL_OPTS[@]}" -f "${CURL_AUTH[@]}" -X POST "${API}/blocks" -H "Content-Type: application/json" \
  -d "{\"name\":\"full-block\",\"cidr\":\"10.7.0.0/26\",\"environment_id\":\"${PROD_ID}\",\"pool_id\":\"${PROD_POOL_ID}\"}" >/dev/null
echo "  full-block (10.7.0.0/26) in Production [for 100% utilization]"

# Nearly full utilization: ~94% allocated (240/256)
curl "${CURL_OPTS[@]}" -f "${CURL_AUTH[@]}" -X POST "${API}/blocks" -H "Content-Type: application/json" \
  -d "{\"name\":\"nearly-full-block\",\"cidr\":\"10.8.0.0/24\",\"environment_id\":\"${PROD_ID}\",\"pool_id\":\"${PROD_POOL_ID}\"}" >/dev/null
echo "  nearly-full-block (10.8.0.0/24) in Production [for ~94% utilization]"

echo "Creating allocations..."
curl "${CURL_OPTS[@]}" -f "${CURL_AUTH[@]}" -X POST "${API}/allocations" -H "Content-Type: application/json" \
  -d '{"name":"prod-web","block_name":"prod-vpc","cidr":"10.0.0.0/24"}' >/dev/null
echo "  prod-web 10.0.0.0/24 in prod-vpc"

curl "${CURL_OPTS[@]}" -f "${CURL_AUTH[@]}" -X POST "${API}/allocations" -H "Content-Type: application/json" \
  -d '{"name":"prod-db","block_name":"prod-vpc","cidr":"10.0.2.0/24"}' >/dev/null
echo "  prod-db 10.0.2.0/24 in prod-vpc (gap 10.0.1.0/24 for bin-pack demo)"

curl "${CURL_OPTS[@]}" -f "${CURL_AUTH[@]}" -X POST "${API}/allocations" -H "Content-Type: application/json" \
  -d '{"name":"staging-app","block_name":"staging-vpc","cidr":"10.1.0.0/24"}' >/dev/null
echo "  staging-app 10.1.0.0/24 in staging-vpc"

curl "${CURL_OPTS[@]}" -f "${CURL_AUTH[@]}" -X POST "${API}/allocations" -H "Content-Type: application/json" \
  -d '{"name":"orphan-subnet","block_name":"orphan-block","cidr":"192.168.0.0/26"}' >/dev/null
echo "  orphan-subnet 192.168.0.0/26 in orphan-block"

# Full utilization: entire block
curl "${CURL_OPTS[@]}" -f "${CURL_AUTH[@]}" -X POST "${API}/allocations" -H "Content-Type: application/json" \
  -d '{"name":"full-alloc","block_name":"full-block","cidr":"10.7.0.0/26"}' >/dev/null
echo "  full-alloc 10.7.0.0/26 in full-block (100% utilization)"

# Nearly full: 240/256 IPs (~94%)
curl "${CURL_OPTS[@]}" -f "${CURL_AUTH[@]}" -X POST "${API}/allocations" -H "Content-Type: application/json" \
  -d '{"name":"nearly-a","block_name":"nearly-full-block","cidr":"10.8.0.0/25"}' >/dev/null
curl "${CURL_OPTS[@]}" -f "${CURL_AUTH[@]}" -X POST "${API}/allocations" -H "Content-Type: application/json" \
  -d '{"name":"nearly-b","block_name":"nearly-full-block","cidr":"10.8.0.128/26"}' >/dev/null
curl "${CURL_OPTS[@]}" -f "${CURL_AUTH[@]}" -X POST "${API}/allocations" -H "Content-Type: application/json" \
  -d '{"name":"nearly-c","block_name":"nearly-full-block","cidr":"10.8.0.192/27"}' >/dev/null
curl "${CURL_OPTS[@]}" -f "${CURL_AUTH[@]}" -X POST "${API}/allocations" -H "Content-Type: application/json" \
  -d '{"name":"nearly-d","block_name":"nearly-full-block","cidr":"10.8.0.224/28"}' >/dev/null
echo "  nearly-a/b/c/d in nearly-full-block (240/256 = ~94% utilization)"

echo "Creating reserved blocks (admin-only; blacklisted CIDRs)..."
curl "${CURL_OPTS[@]}" -f "${CURL_AUTH[@]}" -X POST "${API}/reserved-blocks" -H "Content-Type: application/json" \
  -d '{"name":"Future use","cidr":"10.6.0.0/16","reason":"Reserved for future use"}' >/dev/null
echo "  Future use 10.6.0.0/16 (Reserved for future use)"

curl "${CURL_OPTS[@]}" -f "${CURL_AUTH[@]}" -X POST "${API}/reserved-blocks" -H "Content-Type: application/json" \
  -d '{"name":"Prod gap","cidr":"10.0.1.0/24","reason":"Reserved gap in prod-vpc"}' >/dev/null
echo "  Prod gap 10.0.1.0/24 (Reserved gap in prod-vpc)"

curl "${CURL_OPTS[@]}" -f "${CURL_AUTH[@]}" -X POST "${API}/reserved-blocks" -H "Content-Type: application/json" \
  -d '{"name":"DMZ","cidr":"172.16.0.0/24","reason":"DMZ reserve"}' >/dev/null
echo "  DMZ 172.16.0.0/24 (DMZ reserve)"

# Adversarial: attempt invalid requests and expect API to reject (4xx)
echo "Verifying API rejects invalid data..."
expect_4xx() {
  local desc="$1"
  shift
  local code
  code=$(curl "${CURL_OPTS[@]}" -o /tmp/seed_resp.json -w "%{http_code}" "${CURL_AUTH[@]}" "$@")
  if [ "$code" -lt 400 ] || [ "$code" -gt 499 ]; then
    echo "  FAIL: $desc (expected 4xx, got $code)"
    cat /tmp/seed_resp.json 2>/dev/null | head -3
    exit 1
  fi
  echo "  OK: $desc -> $code"
}

# Environment: empty name
expect_4xx "environment with empty name" -X POST "${API}/environments" -H "Content-Type: application/json" -d '{"name":""}'
# Environment: missing required pools
expect_4xx "environment without pools" -X POST "${API}/environments" -H "Content-Type: application/json" -d '{"name":"BadEnv"}'
# Environment: invalid pool CIDR
expect_4xx "environment with invalid pool CIDR" -X POST "${API}/environments" -H "Content-Type: application/json" \
  -d '{"name":"BadEnv","pools":[{"name":"x","cidr":"not-a-cidr"}]}'

# Block: missing name
expect_4xx "block with missing name" -X POST "${API}/blocks" -H "Content-Type: application/json" \
  -d "{\"name\":\"\",\"cidr\":\"10.99.0.0/24\",\"environment_id\":\"${PROD_ID}\"}"
# Block: invalid CIDR
expect_4xx "block with invalid CIDR" -X POST "${API}/blocks" -H "Content-Type: application/json" \
  -d "{\"name\":\"bad-block\",\"cidr\":\"256.0.0.0/8\",\"environment_id\":\"${PROD_ID}\"}"
# Block: overlaps reserved (10.6.0.0/16 is reserved)
expect_4xx "block overlapping reserved range" -X POST "${API}/blocks" -H "Content-Type: application/json" \
  -d "{\"name\":\"overlap-reserved\",\"cidr\":\"10.6.0.0/24\",\"environment_id\":\"${PROD_ID}\"}"

# Allocation: missing block_name
expect_4xx "allocation with missing block_name" -X POST "${API}/allocations" -H "Content-Type: application/json" \
  -d '{"name":"x","block_name":"","cidr":"10.0.0.0/24"}'
# Allocation: block not found
expect_4xx "allocation with non-existent block" -X POST "${API}/allocations" -H "Content-Type: application/json" \
  -d '{"name":"x","block_name":"no-such-block","cidr":"10.0.0.0/24"}'
# Allocation: CIDR outside parent block (10.0.0.0/24 is in prod-vpc; 10.1.0.0/24 is in staging-vpc)
expect_4xx "allocation CIDR outside parent block" -X POST "${API}/allocations" -H "Content-Type: application/json" \
  -d '{"name":"x","block_name":"prod-vpc","cidr":"10.1.0.0/24"}'
# Allocation: overlaps reserved (10.0.1.0/24 is reserved)
expect_4xx "allocation overlapping reserved range" -X POST "${API}/allocations" -H "Content-Type: application/json" \
  -d '{"name":"x","block_name":"prod-vpc","cidr":"10.0.1.0/24"}'

echo "Done. Seed data and adversarial checks completed at ${BASE_URL}"
