# Track a PR preview deployment so terraform destroy cleans it up.
resource "coolify_application_preview" "pr_42" {
  application_uuid = coolify_application.api.uuid
  pull_request_id  = 42
}
