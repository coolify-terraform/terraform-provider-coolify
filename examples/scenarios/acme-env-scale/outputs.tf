output "project_uuid" {
  value = coolify_project.env_scale.uuid
}

output "database_uuid" {
  value = coolify_database_postgresql.shared.uuid
}

output "api_uuid" {
  value = coolify_application_docker_image.api.uuid
}

output "worker_uuid" {
  value = coolify_application_docker_image.worker.uuid
}

output "api_env_count" {
  value = length(coolify_envs_bulk.api.variables)
}

output "worker_env_count" {
  value = length(coolify_envs_bulk.worker.variables)
}
