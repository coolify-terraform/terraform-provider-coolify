data "coolify_hetzner_locations" "all" {
  cloud_provider_token_uuid = coolify_cloud_token.hetzner.uuid
}

output "hetzner_location_names" {
  value = [for loc in data.coolify_hetzner_locations.all.locations : loc.name]
}
