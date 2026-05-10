data "coolify_environments" "all" {
  project_uuid = "550e8400-e29b-41d4-a716-446655440006"
}

output "environment_names" {
  value = [for e in data.coolify_environments.all.environments : e.name]
}
