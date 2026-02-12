# Create an environment and network blocks (IPv4 and IPv6 ULA) within it.
resource "ipam_environment" "example" {
  name = "prod"
}

# IPv4 block
resource "ipam_block" "example" {
  name           = "prod-vpc"
  cidr           = "10.0.0.0/8"
  environment_id = ipam_environment.example.id
}

# IPv6 ULA block
resource "ipam_block" "example_ula" {
  name           = "prod-ula"
  cidr           = "fd00::/48"
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

output "block_ula_id" {
  value = ipam_block.example_ula.id
}

output "block_ula_cidr" {
  value = ipam_block.example_ula.cidr
}

output "block_ula_total_ips" {
  value = ipam_block.example_ula.total_ips
}
