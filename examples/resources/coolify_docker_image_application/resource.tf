resource "coolify_docker_image_application" "nginx" {
  name          = "nginx-proxy"
  project_uuid  = coolify_project.example.uuid
  server_uuid   = coolify_server.example.uuid
  docker_image  = "nginx:latest"
  ports_exposes = "80"
  fqdn          = "https://proxy.example.com"
}
