# List environment variables for an application
data "coolify_environment_variables" "app_vars" {
  application_uuid = "your-app-uuid"
}

output "env_var_keys" {
  value = [for ev in data.coolify_environment_variables.app_vars.environment_variables : ev.key]
}

# List environment variables for a database
data "coolify_environment_variables" "db_vars" {
  database_uuid = "your-db-uuid"
}

output "db_env_var_keys" {
  value = [for ev in data.coolify_environment_variables.db_vars.environment_variables : ev.key]
}
