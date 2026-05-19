# Set an environment variable on an application
resource "coolify_environment_variable" "database_url" {
  application_uuid = coolify_application.example.uuid
  key              = "DATABASE_URL"
  value            = "postgresql://user:pass@db:5432/myapp"
  is_build         = false
  is_preview       = false
}

# Set a custom environment variable on a database
# Note: do not duplicate built-in credential fields (e.g. POSTGRES_PASSWORD)
# that are already managed by the database resource's own attributes.
# `is_build` is application-only; omit it for database/service variables.
resource "coolify_environment_variable" "db_log_level" {
  database_uuid = coolify_database_postgresql.example.uuid
  key           = "POSTGRES_LOG_MIN_MESSAGES"
  value         = "warning"
}