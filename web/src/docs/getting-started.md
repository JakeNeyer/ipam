# Getting started

Source code: [github.com/JakeNeyer/ipam](https://github.com/JakeNeyer/ipam)

## Quick start

Run the setup script directly with `curl`. It clones the repo, builds the Docker image, starts Postgres, and launches IPAM:

```bash
curl -fsSL https://raw.githubusercontent.com/JakeNeyer/ipam/main/setup.sh | bash
```

Or clone the repo first and run it locally:

```bash
git clone https://github.com/JakeNeyer/ipam.git
cd ipam
bash setup.sh
```

The script builds the IPAM image from the `Dockerfile`, starts a Postgres container, waits for it to be ready, and launches the app. When it finishes, open `http://localhost:8080` and create your admin account.

To skip the setup UI, set credentials before running the script:

```bash
export INITIAL_ADMIN_EMAIL=admin@example.com
export INITIAL_ADMIN_PASSWORD=changeme123
bash setup.sh
```

## Concepts

- **Environments** — Logical groupings (e.g. `Production`, `Staging`) that contain network blocks.
- **Network blocks** — CIDR ranges (e.g. `10.1.0.0/16`) that define a pool of IP addresses.
- **Allocations** — Subnets carved out of a block (e.g. `10.1.0.0/24`) representing actual usage.

## First steps (UI)

1. Create an environment on the **Environments** page.
2. Create a network block on the **Networks** page and assign it to the environment. The CIDR wizard suggests non-overlapping ranges.
3. Create allocations within the block to reserve subnets.

Alternatively, use the **Network Advisor** to plan environments, blocks, and sizing in one guided flow and generate everything at once.

## REST API

Create an API token from **Admin → API tokens**, then use it as a Bearer token. All endpoints are under `/api`.

```bash
# Create an environment
curl -X POST http://localhost:8080/api/environments \
  -H "Authorization: Bearer $IPAM_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name": "production"}'

# Create a network block in that environment
curl -X POST http://localhost:8080/api/blocks \
  -H "Authorization: Bearer $IPAM_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name": "prod-block", "cidr": "10.0.0.0/16", "environment_id": "<env-id>"}'

# Auto-allocate the next available /24 in the block (bin-packing)
curl -X POST http://localhost:8080/api/allocations/auto \
  -H "Authorization: Bearer $IPAM_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name": "vpc-subnet", "block_name": "prod-block", "prefix_length": 24}'
```

The auto-allocate endpoint returns the assigned CIDR in the response. It uses bin-packing to fill gaps in the block before appending to the end. Full API docs are available at `/docs`.

## Terraform provider

The `jakeneyer/ipam` Terraform provider manages environments, blocks, and allocations as infrastructure-as-code.

```hcl
terraform {
  required_providers {
    ipam = {
      source  = "jakeneyer/ipam"
      version = ">= 0.1"
    }
  }
}

provider "ipam" {
  endpoint = "http://localhost:8080"
  token    = var.ipam_token
}

resource "ipam_environment" "prod" {
  name = "production"
}

resource "ipam_block" "main" {
  name           = "prod-block"
  cidr           = "10.0.0.0/16"
  environment_id = ipam_environment.prod.id
}

# Explicit CIDR
resource "ipam_allocation" "explicit" {
  name       = "web-tier"
  block_name = ipam_block.main.name
  cidr       = "10.0.1.0/24"
}

# Auto-allocate: next available /24 via bin-packing
resource "ipam_allocation" "auto" {
  name          = "app-tier"
  block_name    = ipam_block.main.name
  prefix_length = 24
}
```

Set `cidr` for an explicit range, or `prefix_length` to let the API find the next available CIDR automatically. The allocated CIDR is stored in state and available as `ipam_allocation.auto.cidr`.
