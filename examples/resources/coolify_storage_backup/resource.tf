# Requires Coolify v4.x with volume backup API (coollabsio/coolify#10946+).
# Not available on git tag v4.2.0 or stable CDN 4.1.2.
resource "coolify_storage_backup" "app_data" {
  application_uuid = coolify_application.api.uuid
  storage_uuid     = coolify_storage.data.uuid
  frequency        = "0 2 * * *"
  enabled          = true
  save_s3          = false
  timeout          = 3600
}
