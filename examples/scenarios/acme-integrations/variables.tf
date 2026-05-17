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
  default     = "ACME Corp external service integrations"
}