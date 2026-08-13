# Team Pushover notification settings (Coolify >= v4.3.0).
# Import: terraform import coolify_notification_pushover.main current
resource "coolify_notification_pushover" "main" {
  enabled   = true
  user_key  = "replace-with-pushover-user-key"
  api_token = "replace-with-pushover-api-token"

  deployment_failure = true
  backup_failure     = true
  server_disk_usage  = true
}
