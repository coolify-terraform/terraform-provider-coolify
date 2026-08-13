data "coolify_notification_email" "current" {}

output "notifications" {
  value = data.coolify_notification_email.current.smtp_enabled
}
