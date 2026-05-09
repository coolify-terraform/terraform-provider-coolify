data "coolify_server_domains" "example" {
  server_uuid = "existing-server-uuid"
}

output "domain_names" {
  value = [for d in data.coolify_server_domains.example.domains : d.domain]
}
