data "coolify_applications" "all" {}

output "app_names" {
  value = [for a in data.coolify_applications.all.applications : a.name]
}