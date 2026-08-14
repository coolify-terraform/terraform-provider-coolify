resource "coolify_server_cloudflare_tunnel" "example" {
  server_uuid          = coolify_server.example.uuid
  is_cloudflare_tunnel = true
}
