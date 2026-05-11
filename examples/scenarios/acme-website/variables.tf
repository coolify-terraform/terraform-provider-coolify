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

# --- Backups ---

variable "enable_backups" {
  description = "Enable daily database backups to S3-compatible storage"
  type        = bool
  default     = false
}

variable "s3_endpoint" {
  description = "S3-compatible storage endpoint (e.g. https://s3.amazonaws.com)"
  type        = string
  default     = ""
}

variable "s3_bucket" {
  description = "S3 bucket name for database backups"
  type        = string
  default     = ""
}

variable "s3_region" {
  description = "S3 bucket region (e.g. us-east-1)"
  type        = string
  default     = ""
}

variable "s3_access_key" {
  description = "S3 access key"
  type        = string
  default     = ""
  sensitive   = true
}

variable "s3_secret_key" {
  description = "S3 secret key"
  type        = string
  default     = ""
  sensitive   = true
}
