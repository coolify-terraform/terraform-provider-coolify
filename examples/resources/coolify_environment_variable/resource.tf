resource "coolify_environment_variable" "database_url" {
  application_uuid = coolify_application.example.uuid
  key              = "DATABASE_URL"
  value            = "postgresql://user:pass@db:5432/myapp"
  is_build         = false
  is_preview       = false
}