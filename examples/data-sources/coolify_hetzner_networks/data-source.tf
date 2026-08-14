data "coolify_hetzner_networks" "all" {
  cloud_provider_token_uuid = coolify_cloud_token.hetzner.uuid
}

data "coolify_hetzner_networks" "private" {
  cloud_provider_token_uuid = coolify_cloud_token.hetzner.uuid

  filter {
    name   = "name"
    values = ["acme-private"]
  }
}

output "hetzner_network_names" {
  value = [for n in data.coolify_hetzner_networks.all.networks : n.name]
}
