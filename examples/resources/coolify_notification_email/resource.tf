# Team email notification settings (Coolify >= v4.3.0).
# Import: terraform import coolify_notification_email.main current
resource "coolify_notification_email" "main" {
  # To inherit instance-wide SMTP instead, set use_instance_email_settings = true
  # (coolify_instance_email_settings) and omit the smtp_* credentials below.
  smtp_enabled      = true
  smtp_host         = "smtp.example.com"
  smtp_port         = 587
  smtp_encryption   = "starttls"
  smtp_from_address = "alerts@example.com"
  smtp_from_name    = "Coolify"
  smtp_recipients   = "ops@example.com"
  smtp_username     = "smtp-user"
  smtp_password     = "change-me-in-production"
  # smtp_ehlo_domain needs Coolify >= v4.3.10
  # smtp_ehlo_domain = "mail.example.com"

  deployment_failure = true
  backup_failure     = true
  server_disk_usage  = true
  server_unreachable = true
}
