variable "service_resource_uuid" {
  type = string
}

# Attach a persistent storage to an application
resource "coolify_storage" "app_data" {
  application_uuid = coolify_application.example.uuid
  name             = "app-data"
  mount_path       = "/data"
  host_path        = "/opt/app-data"
}

# Attach a persistent storage to a database (Docker volume, no host_path)
resource "coolify_storage" "db_data" {
  database_uuid = coolify_database_postgresql.example.uuid
  name          = "pg-data"
  mount_path    = "/var/lib/postgresql/data"
}

# Attach a persistent storage to a specific sub-resource inside a service.
# resource_uuid must be the nested application or database UUID from the service.
resource "coolify_storage" "service_data" {
  service_uuid  = coolify_service.example.uuid
  resource_uuid = var.service_resource_uuid
  name          = "service-data"
  mount_path    = "/data"
}
