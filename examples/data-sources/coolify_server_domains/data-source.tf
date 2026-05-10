data "coolify_server_domains" "example" {
  server_uuid = "550e8400-e29b-41d4-a716-446655440005"
}

output "domain_names" {
  value = [for d in data.coolify_server_domains.example.domains : d.domain]
}
