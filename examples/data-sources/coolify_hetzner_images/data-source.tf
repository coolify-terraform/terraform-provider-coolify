data "coolify_hetzner_images" "all" {
  cloud_provider_token_uuid = coolify_cloud_token.hetzner.uuid
}

output "hetzner_image_names" {
  value = [for img in data.coolify_hetzner_images.all.images : img.name]
}
