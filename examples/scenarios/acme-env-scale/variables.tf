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
  description = "Project description"
  type        = string
  default     = "Environment variable management at scale demo"
}

variable "app_environment" {
  description = "Application environment name shared across all apps"
  type        = string
  default     = "production"
}

variable "log_level" {
  description = "Log level shared across all apps"
  type        = string
  default     = "info"
}
