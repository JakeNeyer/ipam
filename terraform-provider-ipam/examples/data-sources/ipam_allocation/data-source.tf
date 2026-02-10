# Fetch a single allocation by ID.
data "ipam_allocation" "example" {
  id = "550e8400-e29b-41d4-a716-446655440000"
}

output "allocation_name" {
  value = data.ipam_allocation.example.name
}

output "allocation_cidr" {
  value = data.ipam_allocation.example.cidr
}

output "block_name" {
  value = data.ipam_allocation.example.block_name
}
