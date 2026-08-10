# Requires Coolify >= v4.2.0. Pass uuid into application, database, or service
# create as destination_uuid when the server has more than one network.
resource "coolify_destination" "app_net" {
  server_uuid = coolify_server.main.uuid
  network     = "coolify-app"
  name        = "App network"
  type        = "standalone"
}
