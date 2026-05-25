output "project_uuid" {
  description = "UUID of the project"
  value       = coolify_project.env_scale.uuid
}

output "database_uuid" {
  description = "UUID of the shared PostgreSQL database"
  value       = coolify_database_postgresql.shared.uuid
}

output "api_uuid" {
  description = "UUID of the API application"
  value       = coolify_application_docker_image.api.uuid
}

output "worker_uuid" {
  description = "UUID of the Worker application"
  value       = coolify_application_docker_image.worker.uuid
}

output "api_env_count" {
  description = "Number of environment variables set on the API application"
  value       = length(coolify_envs_bulk.api.variables)
  sensitive   = true
}

output "worker_env_count" {
  description = "Number of environment variables set on the Worker application"
  value       = length(coolify_envs_bulk.worker.variables)
  sensitive   = true
}
