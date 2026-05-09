data "coolify_environment_variables" "app_vars" {
  application_uuid = "your-app-uuid"
}

output "env_var_keys" {
  value = [for ev in data.coolify_environment_variables.app_vars.environment_variables : ev.key]
}
