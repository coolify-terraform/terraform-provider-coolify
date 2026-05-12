variable "s3_access_key" {
  type      = string
  sensitive = true
}

variable "s3_secret_key" {
  type      = string
  sensitive = true
}

resource "coolify_s3_storage" "backups" {
  name       = "backup-storage"
  endpoint   = "https://s3.amazonaws.com"
  bucket     = "coolify-backups"
  region     = "us-east-1"
  access_key = var.s3_access_key
  secret_key = var.s3_secret_key
}

resource "coolify_database_backup" "daily" {
  database_uuid = coolify_postgresql_database.db.uuid
  frequency     = "0 2 * * *"
  enabled       = true
  retain_days   = 7 # Number of backup copies to keep (not days)
  s3_storage_id = coolify_s3_storage.backups.uuid
}
