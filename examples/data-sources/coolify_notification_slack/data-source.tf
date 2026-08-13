data "coolify_notification_slack" "current" {}

output "notifications" {
  value = data.coolify_notification_slack.current.enabled
}
