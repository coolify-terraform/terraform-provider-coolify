# Option 1: Deploy from the Coolify service catalog.
# Common types: uptime-kuma, plausible, minio, grafana, n8n, ghost,
# gitea, code-server, nocodb, appwrite, supabase, meilisearch, umami.
# See the full list in the Coolify UI under Services > New Service.
resource "coolify_service" "catalog" {
  name             = "uptime-kuma"
  type             = "uptime-kuma"
  project_uuid     = coolify_project.example.uuid
  server_uuid      = coolify_server.example.uuid
  environment_name = "production"

  # Optional: connect service containers to the Coolify Docker network
  # connect_to_docker_network = true
}

# Option 2: Deploy a custom Docker Compose stack.
# The provider accepts plain YAML; base64 encoding is handled automatically.
resource "coolify_service" "custom" {
  name         = "my-custom-stack"
  project_uuid = coolify_project.example.uuid
  server_uuid  = coolify_server.example.uuid

  docker_compose_raw = file("docker-compose.yml")

  # Assign domains to service containers
  urls = [{
    name = "web"
    url  = "https://app.example.com"
  }]
}