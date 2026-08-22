# Instance-wide SMTP / Resend settings (Coolify >= v4.3.10).
# Requires a root-team API token. Import: terraform import coolify_instance_email_settings.main current
resource "coolify_instance_email_settings" "main" {
  smtp_enabled      = true
  smtp_host         = "smtp.example.com"
  smtp_port         = 587
  smtp_encryption   = "starttls"
  smtp_from_address = "alerts@example.com"
  smtp_from_name    = "Coolify"
  smtp_username     = "smtp-user"
  smtp_password     = "change-me-in-production"
  smtp_ehlo_domain  = "mail.example.com"
}
