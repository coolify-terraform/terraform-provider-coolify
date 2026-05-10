data "coolify_server_validation" "example" {
  uuid = "550e8400-e29b-41d4-a716-446655440005"
}

output "server_valid" {
  value = data.coolify_server_validation.example.valid
}

output "server_message" {
  value = data.coolify_server_validation.example.message
}
