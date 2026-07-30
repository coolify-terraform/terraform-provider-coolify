variable "github_app_client_secret" {
  type      = string
  sensitive = true
}

variable "github_app_webhook_secret" {
  type      = string
  sensitive = true
}

# Requires a coolify_private_key resource created from the GitHub App
# private key PEM before creating the integration.
resource "coolify_github_app" "example" {
  name             = "my-github-app"
  app_id           = 12345
  installation_id  = 67890
  client_id        = "Iv1.abc123def456"
  client_secret    = var.github_app_client_secret
  webhook_secret   = var.github_app_webhook_secret
  private_key_uuid = coolify_private_key.example.uuid
  # Optional SSH clone defaults (Coolify defaults: user git, port 22)
  # custom_user = "git"
  # custom_port = 22
  # Self-hosted only: share app across teams
  # is_system_wide = false
}
