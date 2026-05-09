resource "coolify_database_backup" "daily" {
  database_uuid = coolify_postgresql_database.db.uuid
  frequency     = "0 2 * * *"
  enabled       = true
  retain_days   = 7
}
