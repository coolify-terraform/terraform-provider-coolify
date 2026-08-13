data "coolify_notification_webhook" "current" {}

output "notifications" {
  value = data.coolify_notification_webhook.current.enabled
}
