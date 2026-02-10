# Fetch a single reserved block by ID (admin only).
data "ipam_reserved_block" "example" {
  id = "550e8400-e29b-41d4-a716-446655440000"
}

output "reserved_block_cidr" {
  value = data.ipam_reserved_block.example.cidr
}

output "reserved_block_reason" {
  value = data.ipam_reserved_block.example.reason
}
