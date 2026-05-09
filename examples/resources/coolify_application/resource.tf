resource "coolify_application" "example" {
  name           = "my-web-app"
  project_uuid   = coolify_project.example.uuid
  server_uuid    = coolify_server.example.uuid
  git_repository = "https://github.com/example/app"
  git_branch     = "main"
  build_pack     = "nixpacks"
  ports_exposes  = "3000"
  fqdn           = "https://app.example.com"
}