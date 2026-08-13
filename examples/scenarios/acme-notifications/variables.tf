variable "coolify_endpoint" {
  description = "Coolify API endpoint URL"
  type        = string
}

variable "coolify_token" {
  description = "Coolify API token"
  type        = string
  sensitive   = true
}

variable "discord_enabled" {
  description = "Whether Discord notifications are enabled (default false for safe demos)"
  type        = bool
  default     = false
}

variable "discord_webhook_url" {
  description = "Discord incoming webhook URL (placeholder OK for demos)"
  type        = string
  sensitive   = true
  default     = "https://example.com/coolify-acme-discord-webhook"
}

variable "smtp_enabled" {
  description = "Whether SMTP delivery is enabled (default false for safe demos)"
  type        = bool
  default     = false
}

variable "smtp_host" {
  description = "SMTP host"
  type        = string
  default     = "smtp.example.com"
}

variable "smtp_port" {
  description = "SMTP port"
  type        = number
  default     = 587
}

variable "smtp_encryption" {
  description = "SMTP encryption: starttls, tls, or none"
  type        = string
  default     = "starttls"
}

variable "smtp_from_address" {
  description = "SMTP From address"
  type        = string
  default     = "alerts@example.com"
}

variable "smtp_from_name" {
  description = "SMTP From display name"
  type        = string
  default     = "ACME Coolify"
}

variable "smtp_recipients" {
  description = "Comma-separated recipient addresses"
  type        = string
  default     = "ops@example.com"
}

variable "smtp_username" {
  description = "SMTP username"
  type        = string
  default     = "smtp-user"
}

variable "smtp_password" {
  description = "SMTP password (placeholder)"
  type        = string
  sensitive   = true
  default     = "change-me-in-production"
}

variable "webhook_enabled" {
  description = "Whether generic webhook notifications are enabled"
  type        = bool
  default     = false
}

variable "webhook_url" {
  description = "Generic webhook URL (placeholder OK for demos)"
  type        = string
  sensitive   = true
  default     = "https://example.com/coolify-acme-webhook"
}
