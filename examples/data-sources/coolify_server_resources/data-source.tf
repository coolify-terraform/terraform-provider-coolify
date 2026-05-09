data "coolify_server_resources" "example" {
  server_uuid = "existing-server-uuid"
}

output "resource_names" {
  value = [for r in data.coolify_server_resources.example.resources : r.name]
}
