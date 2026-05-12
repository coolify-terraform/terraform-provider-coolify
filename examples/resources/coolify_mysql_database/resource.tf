variable "mysql_password" {
  type      = string
  sensitive = true
}

variable "mysql_root_password" {
  type      = string
  sensitive = true
}

resource "coolify_mysql_database" "example" {
  name                = "my-mysql"
  project_uuid        = coolify_project.example.uuid
  server_uuid         = coolify_server.example.uuid
  image               = "mysql:8"
  mysql_user          = "app"
  mysql_password      = var.mysql_password
  mysql_database      = "myapp"
  mysql_root_password = var.mysql_root_password
}