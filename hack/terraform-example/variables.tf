variable "ipam_endpoint" {
  type        = string
  description = "Base URL of the IPAM API (e.g. http://localhost:8011)"
}

variable "ipam_token" {
  type        = string
  sensitive   = true
  description = "API token for IPAM (create in Admin > API tokens)"
}
