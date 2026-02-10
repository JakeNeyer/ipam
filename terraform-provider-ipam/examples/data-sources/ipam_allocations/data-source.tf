# List allocations with optional filters.
data "ipam_allocations" "example" {
  block_name = "prod-vpc"
}

output "allocations" {
  value = data.ipam_allocations.example.allocations
}
