# Set an environment variable on an application
resource "coolify_environment_variable" "database_url" {
  application_uuid = coolify_application.example.uuid
  key              = "DATABASE_URL"
  value            = "postgresql://user:pass@db:5432/myapp"
  is_build         = false
  is_preview       = false
}

# Set an environment variable on a database
resource "coolify_environment_variable" "db_password" {
  database_uuid = coolify_postgresql_database.example.uuid
  key           = "POSTGRES_PASSWORD"
  value         = "supersecret"
}