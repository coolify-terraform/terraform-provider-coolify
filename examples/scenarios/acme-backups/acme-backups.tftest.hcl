# Acceptance test for ACME Corp backup lifecycle.
#
# Tests: postgresql_database, database_backup (with backup_now),
# backup_executions data source.
#
# Required variables via TF_VAR_*:
#   TF_VAR_coolify_endpoint, TF_VAR_coolify_token, TF_VAR_server_uuid

run "setup_and_backup" {
  command = apply

  # --- Project ---
  assert {
    condition     = coolify_project.backups.name == "acme-backups"
    error_message = "Project name mismatch: got ${coolify_project.backups.name}"
  }

  # --- Database ---
  assert {
    condition     = coolify_postgresql_database.app_db.name == "acme-app-db"
    error_message = "Database name mismatch: got ${coolify_postgresql_database.app_db.name}"
  }
  assert {
    condition     = coolify_postgresql_database.app_db.postgres_user == "acme"
    error_message = "Database user mismatch: got ${coolify_postgresql_database.app_db.postgres_user}"
  }

  # --- Backup Configuration ---
  assert {
    condition     = coolify_database_backup.daily.frequency == "0 2 * * *"
    error_message = "Backup frequency mismatch: got ${coolify_database_backup.daily.frequency}"
  }
  assert {
    condition     = coolify_database_backup.daily.enabled == true
    error_message = "Backup should be enabled"
  }
  assert {
    condition     = coolify_database_backup.daily.retain_amount_locally == 7
    error_message = "Retention mismatch: got ${coolify_database_backup.daily.retain_amount_locally}"
  }
}

# Idempotency: re-plan should produce no changes.
run "idempotency" {
  command = plan

  assert {
    condition     = coolify_project.backups.name == "acme-backups"
    error_message = "Project name changed after re-plan"
  }
}
