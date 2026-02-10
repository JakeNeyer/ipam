# Create an environment, block, and allocation (subnet) within the block.
resource "ipam_environment" "example" {
  name = "prod"
}

resource "ipam_block" "example" {
  name           = "prod-vpc"
  cidr           = "10.0.0.0/8"
  environment_id = ipam_environment.example.id
}

resource "ipam_allocation" "example" {
  name       = "region-us-east-1"
  block_name = ipam_block.example.name
  cidr       = "10.0.0.0/16"
}

output "allocation_id" {
  value = ipam_allocation.example.id
}

output "allocation_cidr" {
  value = ipam_allocation.example.cidr
}
