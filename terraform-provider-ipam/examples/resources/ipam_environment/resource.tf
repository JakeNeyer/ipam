# Create an IPAM environment (e.g. prod, staging).
resource "ipam_environment" "example" {
  name = "prod"
}

output "environment_id" {
  value = ipam_environment.example.id
}

output "environment_name" {
  value = ipam_environment.example.name
}
