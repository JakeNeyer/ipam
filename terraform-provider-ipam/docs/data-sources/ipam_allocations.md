# ipam_allocations (Data Source)

Lists allocations with optional filters: allocation name or block name.

## Example Usage

```hcl
data "ipam_allocations" "example" {
  block_name = "prod-vpc"
}

output "allocations" {
  value = data.ipam_allocations.example.allocations
}
```

## Schema

### Optional

- `block_name` (String) Filter by block name.
- `name` (String) Filter by allocation name.

### Read-Only

- `allocations` (List of Object) List of allocations matching the filters.
  - `block_name` (String) Parent block name.
  - `cidr` (String) CIDR range.
  - `id` (String) Allocation UUID.
  - `name` (String) Allocation name.
