data "coolify_digitalocean_sizes" "all" {
  cloud_provider_token_uuid = coolify_cloud_token.do.uuid
}

output "digitalocean_size_slugs" {
  value = [for s in data.coolify_digitalocean_sizes.all.sizes : s.slug]
}
