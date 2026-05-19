variable "postgres_password" {
  type      = string
  sensitive = true
}

resource "coolify_database_postgresql" "example" {
  name              = "my-postgres"
  project_uuid      = coolify_project.example.uuid
  server_uuid       = coolify_server.example.uuid
  image             = "postgres:16"
  postgres_user     = "app"
  postgres_password = var.postgres_password
  postgres_db       = "myapp"
  is_public         = false
}