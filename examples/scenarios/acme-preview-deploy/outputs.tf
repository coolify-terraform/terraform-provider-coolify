output "project_uuid" {
  description = "UUID of the project."
  value       = coolify_project.acme.uuid
}

output "github_app_uuid" {
  description = "UUID of the GitHub App integration."
  value       = coolify_github_app.acme.uuid
}

output "application_uuid" {
  description = "UUID of the deployed application."
  value       = coolify_application_github_app.web.uuid
}

output "preview_pr_1_id" {
  description = "Pull request ID for preview environment 1."
  value       = coolify_application_preview.pr_1.pull_request_id
}

output "preview_pr_2_id" {
  description = "Pull request ID for preview environment 2."
  value       = coolify_application_preview.pr_2.pull_request_id
}
