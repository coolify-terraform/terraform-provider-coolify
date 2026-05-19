# Requires an existing coolify_github_app backed by a live GitHub App
# installation that can access the target repository.
resource "coolify_application_github_app" "app" {
  name            = "my-github-app"
  project_uuid    = coolify_project.example.uuid
  server_uuid     = coolify_server.example.uuid
  github_app_uuid = coolify_github_app.example.uuid
  git_repository  = "https://github.com/myorg/myrepo"
  git_branch      = "main"
  build_pack      = "nixpacks"
  ports_exposes   = "3000"
  fqdn            = "https://app.example.com"
}
