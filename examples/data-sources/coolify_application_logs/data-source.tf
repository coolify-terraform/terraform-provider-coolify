data "coolify_application_logs" "example" {
  uuid = "550e8400-e29b-41d4-a716-446655440000"
}

output "log_lines" {
  value = [for l in data.coolify_application_logs.example.logs : l.line]
}
