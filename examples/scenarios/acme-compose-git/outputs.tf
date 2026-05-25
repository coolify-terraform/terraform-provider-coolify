output "project_uuid" {
  description = "UUID of the project"
  value       = coolify_project.compose.uuid
}

output "service_uuid" {
  description = "UUID of the compose service"
  value       = coolify_service.stack.uuid
}

output "service_name" {
  description = "Name of the service from data source read-back"
  value       = data.coolify_service.stack.name
}
