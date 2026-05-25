output "cloud_token_uuid" {
  description = "UUID of the Hetzner cloud token registered with Coolify."
  value       = coolify_cloud_token.hetzner.uuid
}

output "production_server_name" {
  description = "Name of the production Hetzner server."
  value       = coolify_server_hetzner.production.name
}

output "build_server_name" {
  description = "Name of the dedicated build server."
  value       = coolify_server_hetzner.build.name
}

output "ssh_key_uuid" {
  description = "UUID of the SSH key used for server access."
  value       = coolify_private_key.deploy.uuid
}
