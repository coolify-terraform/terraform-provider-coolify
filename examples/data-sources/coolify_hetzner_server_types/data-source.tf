data "coolify_hetzner_server_types" "all" {}

output "hetzner_server_type_names" {
  value = [for st in data.coolify_hetzner_server_types.all.server_types : st.name]
}
