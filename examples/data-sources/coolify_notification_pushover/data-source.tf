data "coolify_notification_pushover" "current" {}

output "notifications" {
  value = data.coolify_notification_pushover.current.enabled
}
