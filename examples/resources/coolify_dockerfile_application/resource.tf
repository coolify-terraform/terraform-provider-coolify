resource "coolify_dockerfile_application" "app" {
  name                = "my-dockerfile-app"
  project_uuid        = coolify_project.example.uuid
  server_uuid         = coolify_server.example.uuid
  dockerfile_location = "/Dockerfile"
  ports_exposes       = "80"
  fqdn                = "https://app.example.com"
}
