data "coolify_hetzner_locations" "all" {}

output "hetzner_location_names" {
  value = [for loc in data.coolify_hetzner_locations.all.locations : loc.name]
}
