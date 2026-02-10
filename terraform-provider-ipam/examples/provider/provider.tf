# Configure the IPAM provider with endpoint and API token.
# Create API tokens in the IPAM web UI under Admin.
provider "ipam" {
  endpoint = "https://ipam.example.com"
  token   = var.ipam_token
}

variable "ipam_token" {
  type      = string
  sensitive = true
  default   = ""
}
