# Reserve CIDR ranges so they cannot be used as blocks or allocations (admin only).
# IPv4 (documentation range, RFC 5737)
resource "ipam_reserved_block" "example" {
  name   = "reserved-documentation"
  cidr   = "192.0.2.0/24"
  reason = "Reserved for documentation (RFC 5737)"
}

# IPv6 ULA
resource "ipam_reserved_block" "ula" {
  name   = "reserved-ula"
  cidr   = "fd00:0:0:ff00::/56"
  reason = "Reserved IPv6 ULA range"
}

output "reserved_block_id" {
  value = ipam_reserved_block.example.id
}

output "reserved_block_cidr" {
  value = ipam_reserved_block.example.cidr
}

output "reserved_block_ula_cidr" {
  value = ipam_reserved_block.ula.cidr
}
