data "coolify_notification_discord" "current" {}

output "notifications" {
  value = data.coolify_notification_discord.current.enabled
}
