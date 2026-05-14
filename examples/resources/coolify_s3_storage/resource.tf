# Warning: Current Coolify (v4) may not expose the public API used by this resource.
# Prefer creating S3 storage in the Coolify web UI unless you have confirmed
# your target version still exposes the top-level storage CRUD endpoints.

variable "s3_access_key" {
  type      = string
  sensitive = true
}

variable "s3_secret_key" {
  type      = string
  sensitive = true
}

resource "coolify_s3_storage" "example" {
  name        = "my-backup-storage"
  description = "S3 storage for database backups"
  endpoint    = "https://s3.amazonaws.com"
  bucket      = "coolify-backups"
  region      = "us-east-1"
  access_key  = var.s3_access_key
  secret_key  = var.s3_secret_key
}
