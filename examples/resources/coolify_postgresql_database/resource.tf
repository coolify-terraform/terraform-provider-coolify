resource "coolify_postgresql_database" "example" {
  name             = "my-postgres"
  project_uuid     = coolify_project.example.uuid
  server_uuid      = coolify_server.example.uuid
  image            = "postgres:16"
  postgres_user    = "app"
  postgres_password = "secret"
  postgres_db      = "myapp"
  is_public        = false
}