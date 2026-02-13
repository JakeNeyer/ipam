# ipam_environment

Manages an IPAM environment. Environments group network blocks (e.g. prod, staging). **Every environment must have at least one pool** — a CIDR range that network blocks in that environment draw from. You can specify multiple pools when creating the environment.

## Example Usage

```hcl
resource "ipam_environment" "example" {
  name = "prod"
  pools = [
    {
      name = "prod-pool"
      cidr = "10.0.0.0/8"
    }
  ]
}

# Multiple pools
resource "ipam_environment" "multi" {
  name = "staging"
  pools = [
    { name = "staging-v4", cidr = "10.0.0.0/8" },
    { name = "staging-v6", cidr = "fd00::/8" },
  ]
}

output "environment_id" {
  value = ipam_environment.example.id
}

output "first_pool_id" {
  value = ipam_environment.example.pool_ids[0]
}
```

## Schema

### Required

- `name` (String) Environment name.
- `pools` (List of Object, Min: 1) At least one pool. Use `pools = [ { name = "...", cidr = "..." } ]`. Each element has:
  - `name` (String) Pool name.
  - `cidr` (String) Pool CIDR (e.g. `10.0.0.0/8`).

### Optional

- `id` (String) Environment UUID. Set by the provider; use for import.

### Read-Only

- `id` (String) Environment UUID.
- `pool_ids` (List of String) UUIDs of pools created with this environment (same order as `pools`).

## Import

Import an existing environment by UUID:

```bash
terraform import ipam_environment.example <environment-uuid>
```

Example:

```bash
terraform import ipam_environment.example 550e8400-e29b-41d4-a716-446655440000
```
