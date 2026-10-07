resource "coolify_server_sentinel" "example" {
  server_uuid        = coolify_server.example.uuid
  is_metrics_enabled = true

  # Other Sentinel settings are left as Coolify stored them.
  # An omitted value keeps the last applied value.
}
