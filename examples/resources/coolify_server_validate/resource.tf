# Validate that a server is reachable before deploying applications.
resource "coolify_server_validate" "web" {
  server_uuid = coolify_server.web.uuid
}

# Force re-validation by changing the triggers map.
resource "coolify_server_validate" "web_periodic" {
  server_uuid = coolify_server.web.uuid
  triggers = {
    run = timestamp()
  }
}
