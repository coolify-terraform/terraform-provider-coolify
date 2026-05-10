data "coolify_health" "current" {}

output "coolify_health_status" {
  value = data.coolify_health.current.status
}
