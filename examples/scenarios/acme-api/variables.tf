variable "coolify_endpoint" {
  description = "Coolify API endpoint (e.g. https://coolify.example.com)"
  type        = string
}

variable "coolify_token" {
  description = "Coolify API token"
  type        = string
  sensitive   = true
}

variable "server_uuid" {
  description = "UUID of the Coolify server to deploy to"
  type        = string
}

variable "project_description" {
  description = "Project description (used by update scenario test)"
  type        = string
  default     = "ACME Corp order processing microservices"
}
