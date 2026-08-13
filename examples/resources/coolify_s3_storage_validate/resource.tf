# Re-test S3 connectivity after credential rotation (Coolify >= v4.3.0).
resource "coolify_s3_storage" "backups" {
  name     = "backups"
  endpoint = "https://s3.us-east-1.amazonaws.com"
  bucket   = "my-coolify-backups"
  region   = "us-east-1"
  key      = "AKIA..."
  secret   = "change-me-in-production"
}

resource "coolify_s3_storage_validate" "backups" {
  s3_storage_uuid = coolify_s3_storage.backups.uuid

  # Change this map to force Coolify to re-run validation.
  triggers = {
    rotated_at = "2026-08-01"
  }
}
