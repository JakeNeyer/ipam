# Create an environment with one or more pools, and network blocks (IPv4 and IPv6 ULA) within it.
resource "ipam_environment" "example" {
  name = "prod"
  pools = [
    {
      name = "prod-pool"
      cidr = "10.0.0.0/8"
    }
  ]
}

# IPv4 block (CIDR contained in pool 10.0.0.0/8); first pool ID from environment's pool_ids
resource "ipam_block" "example" {
  name           = "prod-vpc"
  cidr           = "10.0.0.0/16"
  environment_id = ipam_environment.example.id
  pool_id        = ipam_environment.example.pool_ids[0]
}

# IPv6 ULA block (no pool_id — CIDR not in the initial pool range)
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
