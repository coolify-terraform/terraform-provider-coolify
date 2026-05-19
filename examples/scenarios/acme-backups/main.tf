# ACME Corp Database Backup & Restore
#
# Shows the full backup lifecycle:
# 1. Create a PostgreSQL database
# 2. Configure automated daily backups with local retention
# 3. Read backup execution history to verify status
#
# This scenario answers: "How do I set up backups and check if they ran?"

terraform {
  required_providers {
    coolify = {
      source = "SebTardifLabs/coolify"
    }
  }
}

provider "coolify" {
  endpoint = var.coolify_endpoint
  token    = var.coolify_token
}

# --- Project ---

resource "coolify_project" "backups" {
  name        = "acme-backups"
  description = var.project_description
}

# --- Database ---

resource "coolify_database_postgresql" "app_db" {
  name             = "acme-app-db"
  project_uuid     = coolify_project.backups.uuid
  server_uuid      = var.server_uuid
  environment_name = "production"
  image            = "postgres:16"
  postgres_user    = "acme"
  postgres_db      = "acme_production"
}

# --- Backup Configuration ---
#
# This creates a *scheduled backup configuration*, not an immediate backup.
# Coolify runs the backup on the cron schedule. To trigger one immediately,
# set backup_now = true on create.

resource "coolify_database_backup" "daily" {
  database_uuid = coolify_database_postgresql.app_db.uuid
  frequency     = "0 2 * * *" # Daily at 2 AM
  enabled       = true
  backup_now    = true # Trigger an immediate backup on creation

  # Local retention: keep the last 7 backups
  retain_amount_locally = 7
}

# --- Read Backup Status ---
#
# The backup *configuration* (coolify_database_backup) does not have a
# status field because it represents the schedule, not an individual run.
#
# To check whether backups are succeeding, query the *executions* data
# source. Each execution has a status: "success", "failed", or "running".

data "coolify_backup_executions" "latest" {
  database_uuid = coolify_database_postgresql.app_db.uuid
  backup_uuid   = coolify_database_backup.daily.uuid
}
