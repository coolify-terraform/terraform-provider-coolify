variable "coolify_endpoint" {
  description = "The Coolify API endpoint URL."
  type        = string
}

variable "coolify_token" {
  description = "The Coolify API token."
  type        = string
  sensitive   = true
}

variable "server_uuid" {
  description = "UUID of the Coolify server to deploy on."
  type        = string
}

variable "deploy_ssh_key" {
  description = "SSH private key for Git repository authentication."
  type        = string
  sensitive   = true
}

variable "git_repository" {
  description = "The private Git repository URL (SSH format, e.g., git@github.com:org/repo.git)."
  type        = string
}

variable "git_branch" {
  description = "The Git branch to deploy."
  type        = string
  default     = "main"
}

variable "database_url" {
  description = "The database connection URL for the application."
  type        = string
  sensitive   = true
}

variable "app_secret" {
  description = "Application secret key."
  type        = string
  sensitive   = true
}
