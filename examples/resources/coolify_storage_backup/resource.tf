resource "coolify_storage_backup" "app_data" {
  application_uuid = coolify_application.api.uuid
  storage_uuid     = coolify_storage.data.uuid
  frequency        = "0 2 * * *"
  enabled          = true
  save_s3          = false
  timeout          = 3600
}
