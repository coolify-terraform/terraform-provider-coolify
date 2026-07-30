# Set an environment variable on an application
resource "coolify_environment_variable" "database_url" {
  application_uuid = coolify_application.example.uuid
  key              = "DATABASE_URL"
  value            = "postgresql://user:pass@db:5432/myapp"
  is_build         = false
  is_runtime       = true
  is_preview       = false
  is_literal       = false
  comment          = "Primary database DSN"
}

# Build-time only secret (not injected at runtime)
resource "coolify_environment_variable" "npm_token" {
  application_uuid = coolify_application.example.uuid
  key              = "NPM_TOKEN"
  value            = "change-me-in-production"
  is_build         = true
  is_runtime       = false
  is_literal       = true
}

# Set a custom environment variable on a database
# Note: do not duplicate built-in credential fields (e.g. POSTGRES_PASSWORD)
# that are already managed by the database resource's own attributes.
# is_build / is_runtime are application-only; omit them for database/service variables.
resource "coolify_environment_variable" "db_log_level" {
  database_uuid = coolify_database_postgresql.example.uuid
  key           = "POSTGRES_LOG_MIN_MESSAGES"
  value         = "warning"
  is_literal    = true
  comment       = "Quiet Postgres logs"
}
