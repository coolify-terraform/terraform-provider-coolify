output "project_uuid" {
  description = "UUID of the project."
  value       = coolify_project.acme.uuid
}

output "github_app_uuid" {
  description = "UUID of the registered GitHub App."
  value       = coolify_github_app.acme.uuid
}

output "application_uuid" {
  description = "UUID of the deployed application."
  value       = coolify_application_github_app.api.uuid
}

output "deployment_uuid" {
  description = "UUID of the deployment."
  value       = coolify_deployment.api.uuid
}

output "deployment_status" {
  description = "Final status of the deployment."
  value       = coolify_deployment.api.status
}