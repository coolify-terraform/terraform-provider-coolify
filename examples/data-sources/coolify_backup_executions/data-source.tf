data "coolify_backup_executions" "db_backup" {
  database_uuid = "your-database-uuid"
  backup_uuid   = "your-backup-uuid"
}

output "backup_execution_statuses" {
  value = [for e in data.coolify_backup_executions.db_backup.executions : e.status]
}
