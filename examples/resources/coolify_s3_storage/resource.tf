# Note: Current Coolify (v4) does not expose a public API for S3 storage CRUD.
# S3 storages are managed through the Coolify web UI.
# This resource targets an API surface that may not be available in your Coolify version.

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
