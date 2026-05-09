resource "coolify_mysql_database" "example" {
  name               = "my-mysql"
  project_uuid       = coolify_project.example.uuid
  server_uuid        = coolify_server.example.uuid
  image              = "mysql:8"
  mysql_user         = "app"
  mysql_password     = "secret"
  mysql_database     = "myapp"
  mysql_root_password = "rootsecret"
}