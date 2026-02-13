# Create an IPAM environment with one or more pools.
resource "ipam_environment" "example" {
  name = "prod"
  pools = [
    {
      name = "prod-pool"
      cidr = "10.0.0.0/8"
    }
  ]
}

output "environment_id" {
  value = ipam_environment.example.id
}

output "environment_name" {
  value = ipam_environment.example.name
}
