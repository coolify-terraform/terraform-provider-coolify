data "coolify_digitalocean_images" "all" {
  cloud_provider_token_uuid = coolify_cloud_token.do.uuid
}

output "digitalocean_image_names" {
  value = [for img in data.coolify_digitalocean_images.all.images : img.name]
}
