# IPAM Provider

The IPAM provider manages resources in an [IPAM](https://github.com/JakeNeyer/ipam) instance: environments, network blocks, allocations, and reserved blocks. It uses the IPAM HTTP API with Bearer token authentication.

## Authentication

The provider uses an **API token** (Bearer token). Create a token in the IPAM web UI: **Admin** → **API tokens** → **Create token**. Use the token value in the provider block (keep it secret; use environment variables or a secret store).

```hcl
provider "ipam" {
  endpoint = "https://ipam.example.com"
  token   = var.ipam_token
}
```

## Example Usage

```hcl
terraform {
  required_providers {
    ipam = {
      source  = "jakeneyer/ipam"
      version = "~> 0.1"
    }
  }
}

provider "ipam" {
  endpoint = "https://ipam.example.com"
  token   = var.ipam_api_token
}

resource "ipam_environment" "prod" {
  name = "prod"
}

resource "ipam_block" "prod_vpc" {
  name           = "prod-vpc"
  cidr           = "10.0.0.0/8"
  environment_id = ipam_environment.prod.id
}

resource "ipam_allocation" "region_a" {
  name       = "region-us-east-1"
  block_name = ipam_block.prod_vpc.name
  cidr       = "10.0.0.0/16"
}
```

## Schema

### Provider Arguments

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| `endpoint` | Base URL of the IPAM API (e.g. `https://ipam.example.com`). | `string` | n/a | yes |
| `token` | API token for authentication (Bearer token). Create tokens in the IPAM UI under Admin. | `string` | n/a | yes (sensitive) |

## Resources

- [ipam_environment](resources/ipam_environment.md) – Manage an IPAM environment.
- [ipam_block](resources/ipam_block.md) – Manage a network block.
- [ipam_allocation](resources/ipam_allocation.md) – Manage an allocation (subnet within a block).
- [ipam_reserved_block](resources/ipam_reserved_block.md) – Reserve a CIDR range (admin only).

## Data Sources

- [ipam_environment](data-sources/ipam_environment.md) – Fetch a single environment by ID.
- [ipam_environments](data-sources/ipam_environments.md) – List environments with optional name filter.
- [ipam_block](data-sources/ipam_block.md) – Fetch a single network block by ID.
- [ipam_blocks](data-sources/ipam_blocks.md) – List network blocks with optional filters.
- [ipam_allocation](data-sources/ipam_allocation.md) – Fetch a single allocation by ID.
- [ipam_allocations](data-sources/ipam_allocations.md) – List allocations with optional filters.
- [ipam_reserved_block](data-sources/ipam_reserved_block.md) – Fetch a single reserved block by ID (admin only).
- [ipam_reserved_blocks](data-sources/ipam_reserved_blocks.md) – List all reserved blocks (admin only).
