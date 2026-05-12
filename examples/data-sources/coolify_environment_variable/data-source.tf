# Look up a single environment variable by UUID
data "coolify_environment_variable" "example" {
  uuid             = "your-env-var-uuid"
  application_uuid = "your-app-uuid"
}

output "env_var_key" {
  value = data.coolify_environment_variable.example.key
}

output "env_var_value" {
  value     = data.coolify_environment_variable.example.value
  sensitive = true
}
