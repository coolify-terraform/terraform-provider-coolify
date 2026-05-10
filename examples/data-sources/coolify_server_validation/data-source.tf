data "coolify_server_validation" "example" {
  uuid = "existing-server-uuid"
}

output "server_valid" {
  value = data.coolify_server_validation.example.valid
}

output "server_message" {
  value = data.coolify_server_validation.example.message
}
