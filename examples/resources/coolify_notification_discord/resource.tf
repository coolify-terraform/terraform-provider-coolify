# Team Discord notification settings (Coolify >= v4.3.0).
# Import: terraform import coolify_notification_discord.main current
resource "coolify_notification_discord" "main" {
  enabled     = true
  webhook_url = "https://discord.com/api/webhooks/000000000000000000/replace-me"

  deployment_failure = true
  backup_failure     = true
  server_disk_usage  = true
  server_unreachable = true
  # restart_limit_reached = true  # tip after 2026-08-31; older APIs 422
}
