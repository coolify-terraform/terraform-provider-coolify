data "coolify_environment" "example" {
  project_uuid = "550e8400-e29b-41d4-a716-446655440006"
  name         = "production"
}

output "environment_uuid" {
  value = data.coolify_environment.example.uuid
}
