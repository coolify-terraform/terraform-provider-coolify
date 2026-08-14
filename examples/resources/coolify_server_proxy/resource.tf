resource "coolify_server_proxy" "example" {
  server_uuid      = coolify_server.example.uuid
  proxy_type       = "traefik"
  redirect_enabled = true
  redirect_url     = "https://example.com"
}
