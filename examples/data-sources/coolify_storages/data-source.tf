# List persistent storages for an application
data "coolify_storages" "app_volumes" {
  application_uuid = "your-app-uuid"
}

output "storage_names" {
  value = [for s in data.coolify_storages.app_volumes.storages : s.name]
}

# List persistent storages for a database
data "coolify_storages" "db_volumes" {
  database_uuid = "your-db-uuid"
}

output "db_storage_paths" {
  value = [for s in data.coolify_storages.db_volumes.storages : s.mount_path]
}
