data "coolify_hetzner_ssh_keys" "all" {
  cloud_provider_token_uuid = coolify_cloud_token.hetzner.uuid
}

output "hetzner_ssh_key_names" {
  value = [for k in data.coolify_hetzner_ssh_keys.all.ssh_keys : k.name]
}
