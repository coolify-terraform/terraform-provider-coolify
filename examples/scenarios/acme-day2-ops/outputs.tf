output "project_uuid" {
  value = coolify_project.ops.uuid
}

output "db_uuid" {
  value = coolify_database_postgresql.db.uuid
}

output "app_uuid" {
  value = coolify_application_docker_image.web.uuid
}

output "restart_action" {
  value = coolify_resource_action.restart_app.action
}

output "restart_resource_type" {
  value = coolify_resource_action.restart_app.resource_type
}
