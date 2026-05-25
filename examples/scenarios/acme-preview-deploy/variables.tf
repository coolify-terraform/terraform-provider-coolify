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

variable "github_app_private_key" {
  description = "PEM-encoded private key for the GitHub App."
  type        = string
  sensitive   = true
  default     = "-----BEGIN RSA PRIVATE KEY-----\nfake-key-for-plan-only\n-----END RSA PRIVATE KEY-----"
}

variable "github_app_id" {
  description = "The GitHub App ID."
  type        = number
  default     = 12345
}

variable "github_app_installation_id" {
  description = "The GitHub App installation ID."
  type        = number
  default     = 67890
}

variable "github_app_client_id" {
  description = "The GitHub App client ID."
  type        = string
  default     = "Iv1.abc123def456"
}

variable "github_app_client_secret" {
  description = "The GitHub App client secret."
  type        = string
  sensitive   = true
  default     = "test-client-secret"
}

variable "github_app_webhook_secret" {
  description = "The GitHub App webhook secret."
  type        = string
  sensitive   = true
  default     = "test-webhook-secret"
}

variable "git_repository" {
  description = "The GitHub repository URL for the application."
  type        = string
  default     = "https://github.com/coollabsio/coolify-examples"
}
