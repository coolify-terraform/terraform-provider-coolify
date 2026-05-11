variable "coolify_endpoint" {
  description = "Coolify API endpoint"
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
