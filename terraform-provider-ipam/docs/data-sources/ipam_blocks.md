# ipam_blocks (Data Source)

Lists network blocks with optional filters: name, environment ID, or orphaned only.

## Example Usage

```hcl
data "ipam_blocks" "example" {
  environment_id = "550e8400-e29b-41d4-a716-446655440000"
}

output "blocks" {
  value = data.ipam_blocks.example.blocks
}
```

## Schema

### Optional

- `environment_id` (String) Filter by environment UUID.
- `name` (String) Filter by name.
- `orphaned_only` (Boolean) Only blocks not assigned to an environment.

### Read-Only

- `blocks` (List of Object) List of network blocks matching the filters.
  - `available_ips` (Number) Available IPs.
  - `cidr` (String) CIDR range.
  - `environment_id` (String) Environment UUID, or empty for orphaned blocks.
  - `id` (String) Block UUID.
  - `name` (String) Block name.
  - `total_ips` (Number) Total IP count in the block.
  - `used_ips` (Number) IPs used by allocations.
