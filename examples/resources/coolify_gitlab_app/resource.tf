variable "gitlab_app_client_secret" {
  type      = string
  sensitive = true
}

resource "coolify_gitlab_app" "example" {
  name          = "corp-gitlab"
  html_url      = "https://gitlab.example.com"
  client_id     = "gitlab-app-id"
  client_secret = var.gitlab_app_client_secret
}
