output "coolify_version" {
  value = data.coolify_version.this.version
}

output "server_is_usable" {
  value = data.coolify_server.target.is_usable
}

output "private_key_uuid" {
  value = coolify_private_key.deploy.uuid
}

output "environment_name" {
  value = coolify_environment.staging.name
}

output "storage_uuid" {
  value = coolify_storage.uploads.uuid
}

