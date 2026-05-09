data "coolify_private_keys" "all" {}

output "key_names" {
  value = [for k in data.coolify_private_keys.all.private_keys : k.name]
}