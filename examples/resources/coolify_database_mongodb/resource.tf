variable "mongo_root_password" {
  type      = string
  sensitive = true
}

resource "coolify_database_mongodb" "example" {
  name                       = "my-mongo"
  project_uuid               = coolify_project.example.uuid
  server_uuid                = coolify_server.example.uuid
  image                      = "mongo:7"
  mongo_initdb_root_username = "admin"
  mongo_initdb_root_password = var.mongo_root_password
  mongo_initdb_database      = "myapp"
}
