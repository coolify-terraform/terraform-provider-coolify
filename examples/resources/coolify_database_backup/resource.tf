# Warning: on current Coolify v4, create the S3 storage in the web UI first
# and pass its UUID here. The top-level `coolify_s3_storage` API surface may
# not be available.
variable "existing_s3_storage_uuid" {
  type = string
}

resource "coolify_database_backup" "daily" {
  database_uuid         = coolify_postgresql_database.db.uuid
  frequency             = "0 2 * * *"
  enabled               = true
  save_s3               = true
  retain_amount_locally = 7 # Number of backup copies to keep (not days)
  s3_storage_uuid       = var.existing_s3_storage_uuid
}
