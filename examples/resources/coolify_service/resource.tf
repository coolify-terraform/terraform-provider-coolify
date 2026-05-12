# Common service types from the Coolify catalog:
#   uptime-kuma, plausible, minio, grafana, n8n, ghost,
#   gitea, code-server, nocodb, appwrite, supabase,
#   meilisearch, umami, fider, appsmith, directus
#
# See the full list in the Coolify UI under Services > New Service.
resource "coolify_service" "monitoring" {
  name             = "uptime-kuma"
  type             = "uptime-kuma"
  project_uuid     = coolify_project.example.uuid
  server_uuid      = coolify_server.example.uuid
  environment_name = "production"
}