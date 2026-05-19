resource "coolify_database_dragonfly" "example" {
  name         = "my-dragonfly"
  project_uuid = coolify_project.example.uuid
  server_uuid  = coolify_server.example.uuid
  image        = "docker.dragonflydb.io/dragonflydb/dragonfly:v1.25"
}
