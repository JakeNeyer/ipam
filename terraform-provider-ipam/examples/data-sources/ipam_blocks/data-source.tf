# List network blocks with optional filters.
data "ipam_blocks" "example" {
  environment_id = "550e8400-e29b-41d4-a716-446655440000"
}

output "blocks" {
  value = data.ipam_blocks.example.blocks
}
