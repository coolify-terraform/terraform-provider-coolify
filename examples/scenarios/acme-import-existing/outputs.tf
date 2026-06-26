output "project_uuid" {
  description = "UUID of the imported project"
  value       = coolify_project.existing.uuid
}

output "app_uuid" {
  description = "UUID of the imported application"
  value       = coolify_application_docker_image.web.uuid
}

output "server_name" {
  description = "Name of the target server (from data source)"
  value       = data.coolify_server.target.name
}
