data "coolify_github_apps" "all" {}

output "github_app_names" {
  value = [for a in data.coolify_github_apps.all.github_apps : a.name]
}
