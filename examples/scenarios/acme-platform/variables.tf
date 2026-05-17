variable "coolify_endpoint" {
  description = "Coolify API endpoint URL"
  type        = string
}

variable "coolify_token" {
  description = "Coolify API token"
  type        = string
  sensitive   = true
}

variable "server_uuid" {
  description = "UUID of the target Coolify server"
  type        = string
}

variable "project_description" {
  description = "Project description (used by update scenario test)"
  type        = string
  default     = "ACME Corp platform infrastructure"
}

variable "deploy_key" {
  description = "SSH private key for deployments (PEM-encoded)"
  type        = string
  sensitive   = true
  default     = "-----BEGIN OPENSSH PRIVATE KEY-----\ngenerate-your-own-key\n-----END OPENSSH PRIVATE KEY-----"
}

