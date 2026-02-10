# ipam_environments (Data Source)

Lists IPAM environments with an optional name filter (substring match).

## Example Usage

```hcl
data "ipam_environments" "all" {
  name = "prod"
}

output "environment_ids" {
  value = [for e in data.ipam_environments.all.environments : e.id]
}

output "environment_names" {
  value = [for e in data.ipam_environments.all.environments : e.name]
}
```

## Schema

### Optional

- `name` (String) Filter by name (substring).

### Read-Only

- `environments` (List of Object) List of environments matching the filter.
  - `id` (String) Environment UUID.
  - `name` (String) Environment name.
