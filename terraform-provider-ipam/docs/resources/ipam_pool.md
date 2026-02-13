# ipam_pool

Manages an IPAM environment pool. A **pool** is a CIDR range that network blocks in an environment can draw from. Hierarchy: **Environment → Pools → Blocks → Allocations**. Creating an environment requires at least one pool via the `pools` argument in `ipam_environment` (e.g. `pools = [ { name = "...", cidr = "..." } ]`); use this resource to add more pools to an environment or manage existing pools.

## Example Usage

```hcl
resource "ipam_pool" "prod_main" {
  environment_id = ipam_environment.prod.id
  name           = "prod-pool"
  cidr           = "10.0.0.0/8"
}

resource "ipam_block" "prod_vpc" {
  name           = "prod-vpc"
  cidr           = "10.0.0.0/16"
  environment_id = ipam_environment.prod.id
  pool_id        = ipam_pool.prod_main.id
}
```

## Schema

### Required

- `environment_id` (String) Environment UUID.
- `name` (String) Pool name.
- `cidr` (String) CIDR range that blocks in this environment can draw from (e.g. `10.0.0.0/8`).

### Optional

- `id` (String) Pool UUID. Set by the provider; use for import.

### Read-Only

- `id` (String) Pool UUID.

## Import

Import an existing pool by UUID:

```bash
terraform import ipam_pool.example <pool-uuid>
```
