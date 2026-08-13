# Team generic webhook notification settings (Coolify >= v4.3.0).
resource "coolify_notification_webhook" "main" {
  enabled     = true
  webhook_url = "https://example.com/coolify-webhook"

  deployment_failure = true
  backup_failure     = true
  server_disk_usage  = true
  server_unreachable = true
}
