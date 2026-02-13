# ipam_pools (Data Source)

List pools for an environment. Use this to discover pool IDs for use in `ipam_block.pool_id` or other resources.

## Example Usage

```hcl
data "ipam_pools" "prod" {
  environment_id = ipam_environment.prod.id
}

output "pool_ids" {
  value = [for p in data.ipam_pools.prod.pools : p.id]
}
```

## Schema

### Required

- `environment_id` (String) Environment UUID.

### Read-Only

- `pools` (List of Object) Pools in the environment. Each element has:
  - `id` (String) Pool UUID.
  - `environment_id` (String) Environment UUID.
  - `name` (String) Pool name.
  - `cidr` (String) Pool CIDR range.
