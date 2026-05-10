data "coolify_hetzner_ssh_keys" "all" {}

output "hetzner_ssh_key_names" {
  value = [for k in data.coolify_hetzner_ssh_keys.all.ssh_keys : k.name]
}
