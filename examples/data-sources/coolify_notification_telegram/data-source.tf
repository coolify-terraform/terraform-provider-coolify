data "coolify_notification_telegram" "current" {}

output "notifications" {
  value = data.coolify_notification_telegram.current.enabled
}
