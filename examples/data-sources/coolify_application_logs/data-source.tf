data "coolify_application_logs" "example" {
  uuid = "existing-application-uuid"
}

output "log_lines" {
  value = [for l in data.coolify_application_logs.example.logs : l.line]
}
