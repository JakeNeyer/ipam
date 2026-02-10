# ipam_block (Data Source)

Fetches a single network block by ID.

## Example Usage

```hcl
data "ipam_block" "example" {
  id = "550e8400-e29b-41d4-a716-446655440000"
}

output "block_name" {
  value = data.ipam_block.example.name
}

output "block_cidr" {
  value = data.ipam_block.example.cidr
}

output "available_ips" {
  value = data.ipam_block.example.available_ips
}
```

## Schema

### Required

- `id` (String) Block UUID.

### Read-Only

- `available_ips` (Number) Available IPs.
- `cidr` (String) CIDR range.
- `environment_id` (String) Environment UUID, or empty for orphaned blocks.
- `name` (String) Block name.
- `total_ips` (Number) Total IP count in the block.
- `used_ips` (Number) IPs used by allocations.
