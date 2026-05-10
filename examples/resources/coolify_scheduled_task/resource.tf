# Create a scheduled task on an application
resource "coolify_scheduled_task" "backup" {
  application_uuid = coolify_application.example.uuid
  name             = "backup-database"
  command          = "pg_dump mydb > /backups/mydb.sql"
  frequency        = "0 2 * * *"
  enabled          = true
}

# Create a scheduled task on a service
resource "coolify_scheduled_task" "health_check" {
  service_uuid = coolify_service.example.uuid
  name         = "health-check"
  command      = "curl -f http://localhost/health || exit 1"
  frequency    = "*/5 * * * *"
}
