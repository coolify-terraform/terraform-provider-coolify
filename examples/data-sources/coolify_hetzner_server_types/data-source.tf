data "coolify_hetzner_server_types" "all" {
  cloud_provider_token_uuid = coolify_cloud_token.hetzner.uuid
}

output "hetzner_server_type_names" {
  value = [for st in data.coolify_hetzner_server_types.all.server_types : st.name]
}
