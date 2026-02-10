# ipam_reserved_blocks (Data Source)

Lists all reserved blocks. **Admin only.**

## Example Usage

```hcl
data "ipam_reserved_blocks" "all" {}

output "reserved_blocks" {
  value = data.ipam_reserved_blocks.all.reserved_blocks
}
```

## Schema

### Read-Only

- `reserved_blocks` (List of Object) List of reserved CIDR blocks.
  - `cidr` (String) Reserved CIDR range.
  - `created_at` (String) Creation time (RFC3339).
  - `id` (String) Reserved block UUID.
  - `name` (String) Optional name for the reserved range.
  - `reason` (String) Optional reason for the reservation.
