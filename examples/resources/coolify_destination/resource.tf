resource "coolify_destination" "app_net" {
  server_uuid = coolify_server.main.uuid
  network     = "coolify-app"
  name        = "App network"
  type        = "standalone"
}
