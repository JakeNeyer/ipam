# ipam_allocation

Manages an IPAM allocation. An allocation is a subnet within a network block (e.g. a VPC or region).

## Example Usage

```hcl
resource "ipam_environment" "example" {
  name = "prod"
  pools = [
    { name = "prod-pool", cidr = "10.0.0.0/8" }
  ]
}

resource "ipam_block" "example" {
  name           = "prod-vpc"
  cidr           = "10.0.0.0/8"
  environment_id = ipam_environment.example.id
  pool_id        = ipam_environment.example.pool_ids[0]
}

resource "ipam_allocation" "example" {
  name       = "region-us-east-1"
  block_name = ipam_block.example.name
  cidr       = "10.0.0.0/16"
}

output "allocation_cidr" {
  value = ipam_allocation.example.cidr
}
```

## Schema

### Required

- `block_name` (String) Name of the parent network block. Changing this forces replacement.
- `cidr` (String) CIDR for this allocation (must be within the block). Changing this forces replacement.
- `name` (String) Allocation name.

### Optional

- `id` (String) Allocation UUID. Set by the provider; use for import.

### Read-Only

- `id` (String) Allocation UUID.

## Import

Import an existing allocation by UUID:

```bash
terraform import ipam_allocation.example <allocation-uuid>
```

Example:

```bash
terraform import ipam_allocation.example 550e8400-e29b-41d4-a716-446655440000
```
