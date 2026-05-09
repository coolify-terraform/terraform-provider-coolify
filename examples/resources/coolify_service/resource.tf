resource "coolify_service" "monitoring" {
  name             = "uptime-kuma"
  type             = "uptime-kuma"
  project_uuid     = coolify_project.example.uuid
  server_uuid      = coolify_server.example.uuid
  environment_name = "production"
}