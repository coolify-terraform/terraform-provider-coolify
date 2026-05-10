data "coolify_environments" "all" {
  project_uuid = "existing-project-uuid"
}

output "environment_names" {
  value = [for e in data.coolify_environments.all.environments : e.name]
}
