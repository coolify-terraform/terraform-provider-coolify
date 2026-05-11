output "project_uuid" {
  description = "UUID of the created Coolify project"
  value       = coolify_project.this.uuid
}

output "database_uuid" {
  description = "UUID of the PostgreSQL database"
  value       = coolify_postgresql_database.app.uuid
}

output "application_uuid" {
  description = "UUID of the deployed application"
  value       = coolify_application.app.uuid
}
