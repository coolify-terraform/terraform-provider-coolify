data "coolify_resource_vulnerabilities" "app" {
  application_uuid = "550e8400-e29b-41d4-a716-446655440000"
  severity         = ["critical", "high"]
}

output "finding_ids" {
  value = [for finding in data.coolify_resource_vulnerabilities.app.findings : finding.display_id]
}
