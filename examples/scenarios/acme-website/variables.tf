# --- Required ---

variable "coolify_endpoint" {
  description = "Coolify API endpoint (e.g. https://coolify.example.com)"
  type        = string
}

variable "coolify_token" {
  description = "Coolify API token"
  type        = string
  sensitive   = true
}

variable "server_uuid" {
  description = "UUID of the destination server registered in Coolify"
  type        = string
}

# --- Update test ---

variable "project_description" {
  description = "Project description (used by update scenario test)"
  type        = string
  default     = "ACME Corp marketing website and content database"
}

# --- Backups ---

variable "enable_backups" {
  description = "Enable daily database backups using an existing UI-managed S3 storage"
  type        = bool
  default     = false
}

variable "existing_s3_storage_uuid" {
  description = "UUID of an existing S3 storage already configured in the Coolify web UI"
  type        = string
  default     = ""
}
