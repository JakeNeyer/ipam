# ipam_environment (Data Source)

Fetches a single IPAM environment by ID.

## Example Usage

```hcl
data "ipam_environment" "example" {
  id = "550e8400-e29b-41d4-a716-446655440000"
}

output "environment_name" {
  value = data.ipam_environment.example.name
}
```

## Schema

### Required

- `id` (String) Environment UUID.

### Read-Only

- `name` (String) Environment name.
