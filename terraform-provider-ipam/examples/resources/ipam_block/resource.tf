# Create an environment and a network block within it.
resource "ipam_environment" "example" {
  name = "prod"
}

resource "ipam_block" "example" {
  name           = "prod-vpc"
  cidr           = "10.0.0.0/8"
  environment_id = ipam_environment.example.id
}

output "block_id" {
  value = ipam_block.example.id
}

output "block_cidr" {
  value = ipam_block.example.cidr
}

output "total_ips" {
  value = ipam_block.example.total_ips
}
