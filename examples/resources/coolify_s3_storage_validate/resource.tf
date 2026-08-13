# Re-test S3 connectivity after credential rotation (Coolify >= v4.3.0).
resource "coolify_s3_storage_validate" "backups" {
  s3_storage_uuid = coolify_s3_storage.backups.uuid

  # Change this map to force Coolify to re-run validation.
  triggers = {
    rotated_at = "2026-08-01"
  }
}
