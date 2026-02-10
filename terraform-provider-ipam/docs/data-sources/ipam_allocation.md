# ipam_allocation (Data Source)

Fetches a single allocation by ID.

## Example Usage

```hcl
data "ipam_allocation" "example" {
  id = "550e8400-e29b-41d4-a716-446655440000"
}

output "allocation_name" {
  value = data.ipam_allocation.example.name
}

output "allocation_cidr" {
  value = data.ipam_allocation.example.cidr
}

output "block_name" {
  value = data.ipam_allocation.example.block_name
}
```

## Schema

### Required

- `id` (String) Allocation UUID.

### Read-Only

- `block_name` (String) Parent block name.
- `cidr` (String) CIDR range.
- `name` (String) Allocation name.
