output "dev_project_uuid" {
  description = "UUID of the dev project"
  value       = module.dev.project_uuid
}

output "dev_db_uuid" {
  description = "UUID of the dev database"
  value       = module.dev.database_uuid
}

output "staging_project_uuid" {
  description = "UUID of the staging project"
  value       = module.staging.project_uuid
}

output "staging_db_uuid" {
  description = "UUID of the staging database"
  value       = module.staging.database_uuid
}
