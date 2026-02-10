# ipam_block

Manages an IPAM network block. A block is a CIDR range assigned to an environment; allocations are subnets within a block.

## Example Usage

```hcl
resource "ipam_environment" "example" {
  name = "prod"
}

resource "ipam_block" "example" {
  name           = "prod-vpc"
  cidr           = "10.0.0.0/8"
  environment_id = ipam_environment.example.id
}

output "block_id" {
  value = ipam_block.example.id
}

output "total_ips" {
  value = ipam_block.example.total_ips
}
```

## Schema

### Required

- `cidr` (String) CIDR range (e.g. `10.0.0.0/8`). Changing this forces replacement.
- `name` (String) Block name.

### Optional

- `environment_id` (String) Environment UUID. Omit for orphaned blocks.
- `id` (String) Block UUID. Set by the provider; use for import.

### Read-Only

- `available_ips` (Number) Available IPs.
- `id` (String) Block UUID.
- `total_ips` (Number) Total IP count in the block.
- `used_ips` (Number) IPs used by allocations.

## Import

Import an existing block by UUID:

```bash
terraform import ipam_block.example <block-uuid>
```

Example:

```bash
terraform import ipam_block.example 550e8400-e29b-41d4-a716-446655440000
```
