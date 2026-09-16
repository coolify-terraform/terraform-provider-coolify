resource "coolify_server_sentinel" "example" {
  server_uuid        = coolify_server.example.uuid
  is_metrics_enabled = true
}
