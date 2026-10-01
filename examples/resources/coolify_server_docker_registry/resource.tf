resource "coolify_server_docker_registry" "example" {
  server_uuid = var.server_uuid
  registry    = "ghcr.io"
  username    = "octocat"
  password    = "change-me-in-production"
}
