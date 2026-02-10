# Fetch a single network block by ID.
data "ipam_block" "example" {
  id = "550e8400-e29b-41d4-a716-446655440000"
}

output "block_name" {
  value = data.ipam_block.example.name
}

output "block_cidr" {
  value = data.ipam_block.example.cidr
}

output "available_ips" {
  value = data.ipam_block.example.available_ips
}
