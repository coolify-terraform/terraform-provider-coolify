output "database_uuid" {
  description = "UUID of the PostgreSQL database"
  value       = coolify_postgresql_database.app_db.uuid
}

output "backup_uuid" {
  description = "UUID of the backup configuration"
  value       = coolify_database_backup.daily.uuid
}

output "backup_frequency" {
  description = "Cron schedule for the backup"
  value       = coolify_database_backup.daily.frequency
}

output "backup_enabled" {
  description = "Whether the backup schedule is active"
  value       = coolify_database_backup.daily.enabled
}

output "execution_count" {
  description = "Number of backup executions recorded"
  value       = length(data.coolify_backup_executions.latest.executions)
}

output "latest_execution_status" {
  description = "Status of the most recent backup execution (empty if none yet)"
  value       = length(data.coolify_backup_executions.latest.executions) > 0 ? data.coolify_backup_executions.latest.executions[0].status : "no executions yet"
}
