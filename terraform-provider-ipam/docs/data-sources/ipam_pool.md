# ipam_pool (Data Source)

Get an IPAM pool by ID. Pools are CIDR ranges that network blocks in an environment draw from.

## Example Usage

```hcl
data "ipam_pool" "prod_pool" {
  id = "550e8400-e29b-41d4-a716-446655440000"
}

output "pool_cidr" {
  value = data.ipam_pool.prod_pool.cidr
}
```

## Schema

### Required

- `id` (String) Pool UUID.

### Read-Only

- `environment_id` (String) Environment UUID.
- `name` (String) Pool name.
- `cidr` (String) Pool CIDR range.
