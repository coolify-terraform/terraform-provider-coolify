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

variable "github_app_private_key" {
  description = "PEM-encoded private key for the GitHub App"
  type        = string
  sensitive   = true
}

variable "github_app_id" {
  description = "The GitHub App ID"
  type        = number
}

variable "github_app_installation_id" {
  description = "The GitHub App installation ID"
  type        = number
}

variable "github_app_client_id" {
  description = "The GitHub App client ID"
  type        = string
}

variable "github_app_client_secret" {
  description = "The GitHub App client secret"
  type        = string
  sensitive   = true
}

variable "github_app_webhook_secret" {
  description = "The GitHub App webhook secret"
  type        = string
  sensitive   = true
  default     = "scenario-webhook-secret"
}

variable "git_repository" {
  description = "The GitHub repository URL"
  type        = string
}

variable "git_branch" {
  description = "The Git branch to deploy"
  type        = string
  default     = "main"
}

variable "database_url" {
  description = "The database connection URL for the application"
  type        = string
  sensitive   = true
  default     = "postgresql://user:pass@db:5432/acme"
}