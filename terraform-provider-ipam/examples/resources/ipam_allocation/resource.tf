# Create an environment, block, and allocations within the block.
resource "ipam_environment" "example" {
  name = "prod"
}

resource "ipam_block" "example" {
  name           = "prod-vpc"
  cidr           = "10.0.0.0/8"
  environment_id = ipam_environment.example.id
}

# Explicit CIDR
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

output "allocation_id" {
  value = ipam_allocation.example.id
}

output "allocation_cidr" {
  value = ipam_allocation.example.cidr
}

output "allocation_auto_cidr" {
  value = ipam_allocation.auto.cidr
}
