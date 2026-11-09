# Requires a connected Gitea app that can read the repository.
resource "coolify_application_gitea_app" "app" {
  name           = "my-gitea-app"
  project_uuid   = coolify_project.example.uuid
  server_uuid    = coolify_server.example.uuid
  gitea_app_uuid = "00000000-0000-0000-0000-000000000000"
  git_repository = "https://gitea.example.com/org/repo"
  git_branch     = "main"
  build_pack     = "nixpacks"
  ports_exposes  = "3000"

  # Requires Coolify >= v4.7.0. Older instances keep these values and do not send them.
  is_image_scan_enabled                       = true
  include_network_hosts_in_build              = true
  is_auto_cancel_queued_deployments_enabled   = true
  is_auto_cancel_building_deployments_enabled = false
}
