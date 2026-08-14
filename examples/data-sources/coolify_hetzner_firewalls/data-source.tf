data "coolify_hetzner_firewalls" "all" {
  cloud_provider_token_uuid = coolify_cloud_token.hetzner.uuid
}

output "hetzner_firewall_names" {
  value = [for fw in data.coolify_hetzner_firewalls.all.firewalls : fw.name]
}
