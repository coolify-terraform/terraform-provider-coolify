resource "coolify_server_docker_cleanup" "example" {
  server_uuid              = coolify_server.example.uuid
  docker_cleanup_frequency = "@daily"
  docker_cleanup_threshold = 80
  force_docker_cleanup     = true
}
