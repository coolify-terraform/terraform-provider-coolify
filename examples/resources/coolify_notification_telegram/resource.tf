# Team Telegram notification settings (Coolify >= v4.3.0).
# Import: terraform import coolify_notification_telegram.main current
resource "coolify_notification_telegram" "main" {
  enabled = true
  token   = "0000000000:replace-with-bot-token"
  chat_id = "-1000000000000"

  deployment_failure = true
  backup_failure     = true
  server_disk_usage  = true
  server_unreachable = true

  # Optional forum topic thread IDs (sensitive).
  # thread_deployment_failure = "123"
}
