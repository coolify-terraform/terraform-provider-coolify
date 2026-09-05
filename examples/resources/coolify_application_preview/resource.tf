# Track a PR preview so terraform destroy can delete it. This resource
# does not create the preview; Coolify creates it (webhook or UI).
# domains PATCH an existing preview (the PR must already have a Coolify
# preview deploy). Requires Coolify >= v4.3.15.
resource "coolify_application_preview" "pr_42" {
  application_uuid = coolify_application.api.uuid
  pull_request_id  = 42
}

# PATCH domains on an existing preview. PR 42 must already be deployed.
resource "coolify_application_preview" "pr_42_domains" {
  application_uuid = coolify_application.api.uuid
  pull_request_id  = 42
  domains          = "https://pr.example.com"
}

# Compose preview: write docker_compose_domains as a JSON array (not the GET object map).
# PR 43 must already be deployed. Requires Coolify >= v4.3.15.
resource "coolify_application_preview" "pr_43_compose" {
  application_uuid       = coolify_application.compose.uuid
  pull_request_id        = 43
  docker_compose_domains = jsonencode([{ name = "web", domain = "https://pr.example.com" }])
}

# Set force_domain_override = true when the domain is already in use.
resource "coolify_application_preview" "pr_44_override" {
  application_uuid      = coolify_application.api.uuid
  pull_request_id       = 44
  domains               = "https://pr.example.com"
  force_domain_override = true
}
