resource "coolify_database_keydb" "example" {
  name         = "my-keydb"
  project_uuid = coolify_project.example.uuid
  server_uuid  = coolify_server.example.uuid
  image        = "eqalpha/keydb:6"
}
