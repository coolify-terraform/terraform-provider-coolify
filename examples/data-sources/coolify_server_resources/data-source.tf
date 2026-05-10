data "coolify_server_resources" "example" {
  server_uuid = "550e8400-e29b-41d4-a716-446655440005"
}

output "resource_names" {
  value = [for r in data.coolify_server_resources.example.resources : r.name]
}
