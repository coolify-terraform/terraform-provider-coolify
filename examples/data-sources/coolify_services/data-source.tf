data "coolify_services" "all" {}

output "service_names" {
  value = [for s in data.coolify_services.all.services : s.name]
}
