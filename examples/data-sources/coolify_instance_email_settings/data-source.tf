data "coolify_instance_email_settings" "current" {}

output "instance_smtp_enabled" {
  value = data.coolify_instance_email_settings.current.smtp_enabled
}
