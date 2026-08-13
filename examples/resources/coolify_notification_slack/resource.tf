# Team Slack notification settings (Coolify >= v4.3.0).
# Import: terraform import coolify_notification_slack.main current
resource "coolify_notification_slack" "main" {
  enabled     = true
  webhook_url = "https://example.com/coolify-slack-webhook"

  deployment_failure = true
  backup_failure     = true
  server_disk_usage  = true
  server_unreachable = true
}
