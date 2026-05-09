resource "coolify_mariadb_database" "example" {
  name                  = "my-mariadb"
  project_uuid          = coolify_project.example.uuid
  server_uuid           = coolify_server.example.uuid
  image                 = "mariadb:11"
  mariadb_user          = "app"
  mariadb_password      = "secret"
  mariadb_database      = "myapp"
  mariadb_root_password = "rootsecret"
}