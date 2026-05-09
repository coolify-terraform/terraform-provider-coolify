resource "coolify_dragonfly_database" "example" {
  name         = "my-dragonfly"
  project_uuid = coolify_project.example.uuid
  server_uuid  = coolify_server.example.uuid
  image        = "docker.dragonflydb.io/dragonflydb/dragonfly:latest"
}
