# Attach a persistent storage to an application
resource "coolify_storage" "app_data" {
  application_uuid = coolify_application.example.uuid
  name             = "app-data"
  mount_path       = "/data"
  host_path        = "/opt/app-data"
}

# Attach a persistent storage to a database (Docker volume, no host_path)
resource "coolify_storage" "db_data" {
  database_uuid = coolify_postgresql_database.example.uuid
  name          = "pg-data"
  mount_path    = "/var/lib/postgresql/data"
}
