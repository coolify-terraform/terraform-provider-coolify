data "coolify_gitea_apps" "all" {}

output "connected_gitea_apps" {
  value = [for app in data.coolify_gitea_apps.all.gitea_apps : app.uuid if app.is_connected]
}
