data "coolify_hetzner_images" "all" {}

output "hetzner_image_names" {
  value = [for img in data.coolify_hetzner_images.all.images : img.name]
}
