# Create an environment with one or more pools, blocks (IPv4 and IPv6), and allocations within them.
resource "ipam_environment" "example" {
  name = "prod"
  pools = [
    {
      name = "prod-pool"
      cidr = "10.0.0.0/8"
    }
  ]
}

# IPv4 block (in pool); first pool ID from environment's pool_ids
resource "ipam_block" "example" {
  name           = "prod-vpc"
  cidr           = "10.0.0.0/8"
  environment_id = ipam_environment.example.id
  pool_id        = ipam_environment.example.pool_ids[0]
}

# IPv6 ULA block (no pool — CIDR not in initial pool range)
resource "ipam_block" "example_ula" {
  name           = "prod-ula"
  cidr           = "fd00::/48"
  environment_id = ipam_environment.example.id
}

# Explicit CIDR (IPv4)
resource "ipam_allocation" "example" {
  name       = "region-us-east-1"
  block_name = ipam_block.example.name
  cidr       = "10.0.0.0/16"
}

# Auto-allocate: next available /24 in the block (uses POST /api/allocations/auto)
resource "ipam_allocation" "auto" {
  name           = "region-us-west-1"
  block_name     = ipam_block.example.name
  prefix_length  = 24
}

# IPv6 ULA allocation
resource "ipam_allocation" "ula_subnet" {
  name       = "prod-ula-subnet"
  block_name = ipam_block.example_ula.name
  cidr       = "fd00::/64"
}

output "allocation_id" {
  value = ipam_allocation.example.id
}

output "allocation_cidr" {
  value = ipam_allocation.example.cidr
}

output "allocation_auto_cidr" {
  value = ipam_allocation.auto.cidr
}

output "allocation_ula_cidr" {
  value = ipam_allocation.ula_subnet.cidr
}
