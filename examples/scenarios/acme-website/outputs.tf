output "project_uuid" {
  description = "UUID of the ACME website project"
  value       = coolify_project.acme.uuid
}

output "database_uuid" {
  description = "UUID of the PostgreSQL content database"
  value       = coolify_database_postgresql.content.uuid
}

output "application_uuid" {
  description = "UUID of the marketing website application"
  value       = coolify_application.website.uuid
}

output "database_url" {
  description = "PostgreSQL connection string for the application"
  value       = "postgresql://${coolify_database_postgresql.content.postgres_user}:${coolify_database_postgresql.content.postgres_password}@${coolify_database_postgresql.content.name}:5432/${coolify_database_postgresql.content.postgres_db}"
  sensitive   = true
}
