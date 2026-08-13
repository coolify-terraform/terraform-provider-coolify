output "coolify_version" {
  description = "Coolify instance version"
  value       = data.coolify_version.current.version
}

output "discord_id" {
  description = "Discord notification resource id (always current)"
  value       = coolify_notification_discord.alerts.id
}

output "email_id" {
  description = "Email notification resource id (always current)"
  value       = coolify_notification_email.ops.id
}

output "webhook_id" {
  description = "Webhook notification resource id (always current)"
  value       = coolify_notification_webhook.hooks.id
}
