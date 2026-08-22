# Instance-wide SMTP / Resend settings (Coolify >= v4.3.10).
# Requires a root-team API token (team 0).
data "coolify_instance_email_settings" "current" {}

output "instance_smtp_enabled" {
  value = data.coolify_instance_email_settings.current.smtp_enabled
}
