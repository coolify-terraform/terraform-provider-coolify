data "coolify_vulnerabilities" "update_now" {
  status = ["update_now", "update_available"]
}

output "resource_names" {
  value = [for resource in data.coolify_vulnerabilities.update_now.resources : resource.name]
}
