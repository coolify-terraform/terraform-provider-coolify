resource "coolify_docker_image_application" "nginx" {
  name          = "nginx-proxy"
  project_uuid  = coolify_project.example.uuid
  server_uuid   = coolify_server.example.uuid
  docker_image  = "nginx:latest"
  ports_exposes = "80"
  fqdn          = "https://proxy.example.com"

  # Optional fields (uncomment as needed):
  # docker_registry_image_tag = "1.27-alpine"  # Pin to a specific image tag instead of :latest
}
