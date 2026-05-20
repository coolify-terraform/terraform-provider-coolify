# Track a backup execution so terraform destroy purges the record.
resource "coolify_backup_execution" "old_daily" {
  database_uuid  = coolify_database_postgresql.example.uuid
  backup_uuid    = coolify_database_backup.daily.uuid
  execution_uuid = "550e8400-e29b-41d4-a716-446655440099"
}
