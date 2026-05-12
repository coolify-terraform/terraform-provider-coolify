# docker_compose_raw must be base64-encoded.
resource "coolify_docker_compose_application" "example" {
  name         = "my-compose-stack"
  project_uuid = coolify_project.example.uuid
  server_uuid  = coolify_server.example.uuid
  docker_compose_raw = base64encode(<<-YAML
    services:
      web:
        image: nginx:latest
        ports:
          - "80:80"
  YAML
  )
  fqdn = "https://compose.example.com"
}
