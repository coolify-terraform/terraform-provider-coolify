# Start a database
resource "coolify_resource_action" "start_db" {
  resource_uuid = coolify_database_postgresql.main.uuid
  resource_type = "database"
  action        = "start"
}

# Restart an application whenever the image tag changes
resource "coolify_resource_action" "restart_app" {
  resource_uuid = coolify_application.api.uuid
  resource_type = "application"
  action        = "restart"

  triggers = {
    image_tag = var.image_tag
  }
}

# Stop a service
resource "coolify_resource_action" "stop_svc" {
  resource_uuid = coolify_service.monitoring.uuid
  resource_type = "service"
  action        = "stop"
}
