output "application_uuid" {
  description = "UUID of the deployed application."
  value       = coolify_application_private_git.backend.uuid
}

output "deployment_uuid" {
  description = "UUID of the deployment."
  value       = coolify_deployment.backend.uuid
}

output "deployment_status" {
  description = "Final status of the deployment."
  value       = coolify_deployment.backend.status
}
