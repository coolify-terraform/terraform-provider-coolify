output "application_uuid" {
  description = "UUID of the deployed application."
  value       = coolify_private_git_application.backend.uuid
}

output "deployment_uuid" {
  description = "UUID of the deployment."
  value       = coolify_deployment.backend.uuid
}

output "deployment_status" {
  description = "Final status of the deployment."
  value       = coolify_deployment.backend.status
}
