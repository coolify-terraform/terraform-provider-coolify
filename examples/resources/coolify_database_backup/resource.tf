# Create the S3 storage in the Coolify web UI first, then pass its UUID here.
# Current Coolify v4 does not expose a public top-level S3 storage CRUD API.
variable "existing_s3_storage_uuid" {
  type = string
}

resource "coolify_database_backup" "daily" {
  database_uuid         = coolify_postgresql_database.example.uuid
  frequency             = "0 2 * * *"
  enabled               = true
  save_s3               = true
  retain_amount_locally = 7 # Number of backup copies to keep (not days)
  s3_storage_uuid       = var.existing_s3_storage_uuid
}
