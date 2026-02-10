# Reserve a CIDR range so it cannot be used as a block or allocation (admin only).
resource "ipam_reserved_block" "example" {
  name   = "reserved-documentation"
  cidr   = "192.0.2.0/24"
  reason = "Reserved for documentation (RFC 5737)"
}

output "reserved_block_id" {
  value = ipam_reserved_block.example.id
}

output "reserved_block_cidr" {
  value = ipam_reserved_block.example.cidr
}
