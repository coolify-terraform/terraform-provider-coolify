resource "coolify_server" "example" {
  name             = "my-server"
  description      = "Production server"
  ip               = "192.168.1.100"
  port             = 22
  user             = "root"
  private_key_uuid = coolify_private_key.example.uuid
}