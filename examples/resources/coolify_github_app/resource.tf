variable "github_app_client_secret" {
  type      = string
  sensitive = true
}

resource "coolify_github_app" "example" {
  name             = "my-github-app"
  app_id           = 12345
  installation_id  = 67890
  client_id        = "Iv1.abc123def456"
  client_secret    = var.github_app_client_secret
  private_key_uuid = coolify_private_key.github.uuid
}
