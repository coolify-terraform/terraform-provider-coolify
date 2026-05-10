data "coolify_resources" "all" {}

output "resource_names" {
  value = [for r in data.coolify_resources.all.resources : r.name]
}
