# Example Terraform config using the IPAM provider against a running instance.
# Run from repo root: ./hack/terraform-example.sh [plan|apply|destroy]
# Requires: IPAM_ENDPOINT and IPAM_TOKEN set (or -var ipam_endpoint=... -var ipam_token=...)

terraform {
  required_providers {
    ipam = {
      source  = "jakeneyer/ipam"
      version = ">= 0.1"
    }
  }
}

provider "ipam" {
  endpoint = var.ipam_endpoint
  token    = var.ipam_token
}

# Create an environment with one or more pools
resource "ipam_environment" "hack" {
  name = "tf-hack-env"
  pools = [
    {
      name = "tf-hack-pool"
      cidr = "10.200.0.0/16"
    }
  ]
}

# First pool ID comes from the environment's computed pool_ids (same order as pools)
# Optional: use data "ipam_pools" to look up pools by name or list all pool details
data "ipam_pools" "hack" {
  environment_id = ipam_environment.hack.id
}

# IPv4 block in the pool (CIDR contained in pool 10.200.0.0/16)
resource "ipam_block" "hack" {
  name           = "tf-hack-block"
  cidr           = "10.200.0.0/24"
  environment_id = ipam_environment.hack.id
  pool_id        = ipam_environment.hack.pool_ids[0]
}

resource "ipam_allocation" "hack" {
  name       = "tf-hack-alloc"
  block_name = ipam_block.hack.name
  cidr       = "10.200.0.0/26"
}

# Auto-allocate: next available /26 in the block (uses POST /api/allocations/auto)
resource "ipam_allocation" "hack_auto" {
  name           = "tf-hack-alloc-auto"
  block_name     = ipam_block.hack.name
  prefix_length  = 26
}

# IPv6 ULA block and allocation
resource "ipam_block" "hack_ula" {
  name           = "tf-hack-ula"
  cidr           = "fd00:200::/48"
  environment_id = ipam_environment.hack.id
}

resource "ipam_allocation" "hack_ula_subnet" {
  name       = "tf-hack-ula-subnet"
  block_name = ipam_block.hack_ula.name
  cidr       = "fd00:200::/64"
}

output "environment_id" {
  value = ipam_environment.hack.id
}

output "pool_id" {
  value       = ipam_environment.hack.pool_ids[0]
  description = "Initial pool created with the environment (used by ipam_block.hack)"
}

output "block_id" {
  value = ipam_block.hack.id
}

output "allocation_id" {
  value = ipam_allocation.hack.id
}

output "allocation_cidr" {
  value = ipam_allocation.hack.cidr
}

output "allocation_auto_id" {
  value = ipam_allocation.hack_auto.id
}

output "allocation_auto_cidr" {
  value = ipam_allocation.hack_auto.cidr
}

output "block_ula_id" {
  value = ipam_block.hack_ula.id
}

output "allocation_ula_cidr" {
  value = ipam_allocation.hack_ula_subnet.cidr
}
