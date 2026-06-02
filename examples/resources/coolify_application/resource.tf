resource "coolify_application" "example" {
  name           = "my-web-app"
  project_uuid   = coolify_project.example.uuid
  server_uuid    = coolify_server.example.uuid
  git_repository = "https://github.com/example/app"
  git_branch     = "main"
  build_pack     = "nixpacks"
  ports_exposes  = "3000"
  domains        = "https://app.example.com"

  # Optional fields (uncomment as needed):
  # redirect                 = "both"                # WWW redirect: "www", "non-www", or "both" (default: "both")
  # base_directory           = "/app"                 # Base directory for the application source code (default: "/")
  # watch_paths              = "/src:/lib"            # Paths to watch for changes (triggers auto-deploy)
  # is_static                = false                  # Whether the application is a static site
  # is_force_https_enabled   = true                   # Whether to force HTTPS (default: true)
  # health_check_type        = "http"                 # Health check type: "http" or "cmd" (default: "http")
  # pre_deployment_command   = "npm run migrate"      # Command to run before deployment
  # post_deployment_command  = "npm run seed"         # Command to run after deployment
  # custom_docker_run_options = "--memory=512m"       # Custom Docker run options passed to the container
}
