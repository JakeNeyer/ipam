# List all reserved blocks (admin only).
data "ipam_reserved_blocks" "all" {}

output "reserved_blocks" {
  value = data.ipam_reserved_blocks.all.reserved_blocks
}
