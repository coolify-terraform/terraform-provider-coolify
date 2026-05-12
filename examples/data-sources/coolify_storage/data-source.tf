# Look up a single persistent storage by UUID
data "coolify_storage" "example" {
  uuid             = "your-storage-uuid"
  application_uuid = "your-app-uuid"
}

output "storage_mount_path" {
  value = data.coolify_storage.example.mount_path
}

output "storage_host_path" {
  value = data.coolify_storage.example.host_path
}
