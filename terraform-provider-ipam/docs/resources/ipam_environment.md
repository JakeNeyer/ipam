# ipam_environment

Manages an IPAM environment. Environments group network blocks (e.g. prod, staging).

## Example Usage

```hcl
resource "ipam_environment" "example" {
  name = "prod"
}

output "environment_id" {
  value = ipam_environment.example.id
}
```

## Schema

### Required

- `name` (String) Environment name.

### Optional

- `id` (String) Environment UUID. Set by the provider; use for import.

### Read-Only

- `id` (String) Environment UUID.

## Import

Import an existing environment by UUID:

```bash
terraform import ipam_environment.example <environment-uuid>
```

Example:

```bash
terraform import ipam_environment.example 550e8400-e29b-41d4-a716-446655440000
```
