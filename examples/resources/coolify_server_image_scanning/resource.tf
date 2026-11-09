resource "coolify_server_image_scanning" "example" {
  server_uuid           = coolify_server.example.uuid
  is_image_scan_enabled = true
}
