data "coolify_private_key" "example" {
  uuid = "550e8400-e29b-41d4-a716-446655440003"
}

output "private_key_public_key" {
  value = data.coolify_private_key.example.public_key
}

output "private_key_fingerprint" {
  value = data.coolify_private_key.example.fingerprint
}
