# ACME Corp team notification channels
#
# Configures Coolify team-scoped notification settings for Discord, email
# (SMTP), and a generic webhook. Channels start disabled so apply does not
# send real traffic to external endpoints. Use terraform import with id
# "current" to adopt an existing team configuration.
#
# Requires Coolify >= v4.3.0.
#
# Caution: apply overwrites the API token's team notification settings
# (including webhook URLs and SMTP fields). Do not point this at a
# production team with live channels unless that is intentional.
#
# Destroy disables the channels (enabled = false / smtp_enabled = false)
# without deleting Coolify-side webhook URLs or SMTP credentials.

terraform {
  required_providers {
    coolify = {
      source = "coolify-terraform/coolify"
    }
  }
}

provider "coolify" {
  endpoint = var.coolify_endpoint
  token    = var.coolify_token
}

data "coolify_version" "current" {}

# Discord: placeholder webhook; keep enabled=false for safe demos.
resource "coolify_notification_discord" "alerts" {
  enabled     = var.discord_enabled
  webhook_url = var.discord_webhook_url

  deployment_failure = true
  backup_failure     = true
  status_change      = var.discord_status_change
  server_disk_usage  = true
  server_unreachable = true
}

# Email: SMTP placeholders; smtp_enabled=false by default.
resource "coolify_notification_email" "ops" {
  smtp_enabled      = var.smtp_enabled
  smtp_host         = var.smtp_host
  smtp_port         = var.smtp_port
  smtp_encryption   = var.smtp_encryption
  smtp_from_address = var.smtp_from_address
  smtp_from_name    = var.smtp_from_name
  smtp_recipients   = var.smtp_recipients
  smtp_username     = var.smtp_username
  smtp_password     = var.smtp_password

  deployment_failure = true
  backup_failure     = true
  server_disk_usage  = true
  server_unreachable = true
}

# Generic webhook as a third channel example (also disabled by default).
resource "coolify_notification_webhook" "hooks" {
  enabled     = var.webhook_enabled
  webhook_url = var.webhook_url

  deployment_failure = true
  status_change      = true
}
