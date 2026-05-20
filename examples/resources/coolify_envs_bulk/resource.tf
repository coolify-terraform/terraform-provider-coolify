# Set all environment variables for an application in a single atomic operation.
resource "coolify_envs_bulk" "app" {
  resource_type = "application"
  resource_uuid = coolify_application.example.uuid
  variables = {
    APP_ENV   = "production"
    LOG_LEVEL = "info"
    DB_HOST   = coolify_database_postgresql.example.internal_db_url
  }
}
