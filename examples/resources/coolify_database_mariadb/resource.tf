variable "mariadb_password" {
  type      = string
  sensitive = true
}

variable "mariadb_root_password" {
  type      = string
  sensitive = true
}

resource "coolify_database_mariadb" "example" {
  name                  = "my-mariadb"
  project_uuid          = coolify_project.example.uuid
  server_uuid           = coolify_server.example.uuid
  image                 = "mariadb:11"
  mariadb_user          = "app"
  mariadb_password      = var.mariadb_password
  mariadb_database      = "myapp"
  mariadb_root_password = var.mariadb_root_password
}