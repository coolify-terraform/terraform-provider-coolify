# Acceptance test for ACME Corp team notification channels.
#
# Tests: coolify_notification_discord, coolify_notification_email,
# coolify_notification_webhook (team singletons; import id "current").
#
# Requires Coolify >= v4.3.0. Required variables via TF_VAR_*:
#   TF_VAR_coolify_endpoint, TF_VAR_coolify_token
#
# Channels default to disabled so the scenario does not send real alerts.

run "configure_channels" {
  command = apply

  assert {
    condition     = data.coolify_version.current.version != ""
    error_message = "Coolify version is empty"
  }

  assert {
    condition     = coolify_notification_discord.alerts.id == "current"
    error_message = "Discord id should be current, got ${coolify_notification_discord.alerts.id}"
  }

  assert {
    condition     = coolify_notification_discord.alerts.enabled == false
    error_message = "Discord should stay disabled by default for safe demos"
  }

  assert {
    condition     = coolify_notification_discord.alerts.deployment_failure == true
    error_message = "Discord deployment_failure should be true"
  }

  assert {
    condition     = coolify_notification_email.ops.id == "current"
    error_message = "Email id should be current, got ${coolify_notification_email.ops.id}"
  }

  assert {
    condition     = coolify_notification_email.ops.smtp_enabled == false
    error_message = "SMTP should stay disabled by default for safe demos"
  }

  assert {
    condition     = coolify_notification_email.ops.smtp_encryption == "starttls"
    error_message = "SMTP encryption mismatch: got ${coolify_notification_email.ops.smtp_encryption}"
  }

  assert {
    condition     = coolify_notification_webhook.hooks.id == "current"
    error_message = "Webhook id should be current"
  }

  assert {
    condition     = coolify_notification_webhook.hooks.enabled == false
    error_message = "Webhook should stay disabled by default"
  }
}

run "update_discord_events" {
  command = apply

  variables {
    discord_status_change = true
  }

  assert {
    condition     = coolify_notification_discord.alerts.status_change == true
    error_message = "Discord status_change should become true after update"
  }

  assert {
    condition     = coolify_notification_discord.alerts.backup_failure == true
    error_message = "Discord backup_failure should remain true after re-apply"
  }
}

run "idempotency" {
  command = plan

  assert {
    condition     = coolify_notification_discord.alerts.id == "current"
    error_message = "Discord id changed after re-plan"
  }
}
