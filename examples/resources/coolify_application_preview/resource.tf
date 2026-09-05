# Track a PR preview deployment so terraform destroy cleans it up.
# domains requires Coolify >= v4.3.15; omit it on older instances.
resource "coolify_application_preview" "pr_42" {
  application_uuid = coolify_application.api.uuid
  pull_request_id  = 42
  domains          = "https://pr.example.com"
}

# Compose preview: write docker_compose_domains as a JSON array (not the GET object map).
# Requires Coolify >= v4.3.15; omit it on older instances.
resource "coolify_application_preview" "pr_43_compose" {
  application_uuid       = coolify_application.compose.uuid
  pull_request_id        = 43
  docker_compose_domains = jsonencode([{ name = "web", domain = "https://pr.example.com" }])
}
