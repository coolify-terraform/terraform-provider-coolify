resource "coolify_mongodb_database" "example" {
  name                        = "my-mongo"
  project_uuid                = coolify_project.example.uuid
  server_uuid                 = coolify_server.example.uuid
  image                       = "mongo:7"
  mongo_initdb_root_username  = "admin"
  mongo_initdb_root_password  = "secret"
  mongo_initdb_database       = "myapp"
}