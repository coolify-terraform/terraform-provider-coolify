data "coolify_gitlab_apps" "all" {}

output "gitlab_app_names" {
  value = [for a in data.coolify_gitlab_apps.all.apps : a.name]
}
