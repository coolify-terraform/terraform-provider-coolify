resource "coolify_application_private_git" "api" {
  name             = "api-server"
  project_uuid     = coolify_project.example.uuid
  server_uuid      = coolify_server.example.uuid
  git_repository   = "git@github.com:myorg/api-server.git"
  git_branch       = "main"
  private_key_uuid = coolify_private_key.deploy.uuid
  build_pack       = "dockerfile"
  ports_exposes    = "8080"
  fqdn             = "https://api.example.com"
}
