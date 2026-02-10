# ipam_reserved_block

Reserves a CIDR block so it cannot be used as a network block or allocation. **Admin only.** Changing `cidr` forces replacement; name and reason cannot be updated in place.

## Example Usage

```hcl
resource "ipam_reserved_block" "example" {
  name   = "reserved-documentation"
  cidr   = "192.0.2.0/24"
  reason = "Reserved for documentation (RFC 5737)"
}

output "reserved_block_cidr" {
  value = ipam_reserved_block.example.cidr
}
```

## Schema

### Required

- `cidr` (String) CIDR range to reserve (e.g. `10.0.0.0/8`). Changing this forces replacement.

### Optional

- `name` (String) Optional name for the reserved range.
- `reason` (String) Optional reason for the reservation.

### Read-Only

- `created_at` (String) Creation time (RFC3339).
- `id` (String) Reserved block UUID.

## Import

Import an existing reserved block by UUID (admin only):

```bash
terraform import ipam_reserved_block.example <reserved-block-uuid>
```

Example:

```bash
terraform import ipam_reserved_block.example 550e8400-e29b-41d4-a716-446655440000
```
