# List environments with optional name filter.
data "ipam_environments" "all" {
  name = "prod"
}

output "environment_ids" {
  value = [for e in data.ipam_environments.all.environments : e.id]
}

output "environment_names" {
  value = [for e in data.ipam_environments.all.environments : e.name]
}
