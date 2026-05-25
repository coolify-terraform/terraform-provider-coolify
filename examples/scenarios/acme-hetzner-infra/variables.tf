variable "coolify_endpoint" {
  description = "The Coolify API endpoint URL."
  type        = string
}

variable "coolify_token" {
  description = "The Coolify API token."
  type        = string
  sensitive   = true
}

variable "hetzner_api_token" {
  description = "Hetzner Cloud API token for server provisioning."
  type        = string
  sensitive   = true
  default     = "test-hetzner-token-for-plan"
}

variable "deploy_ssh_key" {
  description = "SSH private key for Hetzner Cloud server access."
  type        = string
  sensitive   = true
  default     = "-----BEGIN OPENSSH PRIVATE KEY-----\ntest-key-for-plan-only\n-----END OPENSSH PRIVATE KEY-----"
}
