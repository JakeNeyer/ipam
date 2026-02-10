# ipam_reserved_block (Data Source)

Fetches a single reserved block by ID. **Admin only.**

## Example Usage

```hcl
data "ipam_reserved_block" "example" {
  id = "550e8400-e29b-41d4-a716-446655440000"
}

output "reserved_block_cidr" {
  value = data.ipam_reserved_block.example.cidr
}

output "reserved_block_reason" {
  value = data.ipam_reserved_block.example.reason
}
```

## Schema

### Required

- `id` (String) Reserved block UUID.

### Read-Only

- `cidr` (String) Reserved CIDR range.
- `created_at` (String) Creation time (RFC3339).
- `name` (String) Optional name for the reserved range.
- `reason` (String) Optional reason for the reservation.
