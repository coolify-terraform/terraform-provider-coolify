# dockerfile_location expects Dockerfile content (not a file path, despite
# the field name). The provider accepts plain text or pre-encoded base64;
# encoding is handled automatically.
resource "coolify_application_dockerfile" "app" {
  name                = "my-dockerfile-app"
  project_uuid        = coolify_project.example.uuid
  server_uuid         = coolify_server.example.uuid
  dockerfile_location = <<-DOCKERFILE
    FROM nginx:alpine
    EXPOSE 80
  DOCKERFILE
  ports_exposes       = "80"
  domains             = "https://app.example.com"

  # Optional fields (uncomment as needed):
  # dockerfile_target_build = "production"  # Target stage for multi-stage Docker builds

  # Required when the app holds an exclusive lock on a persistent volume
  # (SQLite, DuckDB, LMDB, BoltDB). Without this, Coolify rolling updates
  # report success while the new container never starts.
  # is_consistent_container_name_enabled = true
  # max_restart_count = 3  # Coolify >= v4.3.0; API default 10
}
