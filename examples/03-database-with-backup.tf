# 03 - Database with Backup: provision a database and schedule backups
#
# Creates a PostgreSQL database and a scheduled backup that targets an
# existing UI-managed S3 storage, then runs daily at 2 AM UTC.
#
# Prerequisites:
#   - Coolify with a registered server
#   - An existing S3-compatible storage configured in the Coolify web UI
#     (AWS S3, MinIO, Backblaze B2, etc.)
#
# Run:
#   terraform init
#   terraform plan -var-file=secrets.tfvars
#   terraform apply -var-file=secrets.tfvars

terraform {
  required_providers {
    coolify = {
      source = "SebTardifLabs/coolify"
    }
  }
}

provider "coolify" {}

variable "server_uuid" {
  type = string
}

variable "existing_s3_storage_uuid" {
  type = string
}

resource "coolify_project" "data" {
  name = "data-tier"
}

resource "coolify_postgresql_database" "main" {
  name         = "app-database"
  project_uuid = coolify_project.data.uuid
  server_uuid  = var.server_uuid
  image        = "postgres:16"
}

resource "coolify_database_backup" "daily" {
  database_uuid         = coolify_postgresql_database.main.uuid
  s3_storage_uuid       = var.existing_s3_storage_uuid
  frequency             = "0 2 * * *"
  retain_amount_locally = 7 # Number of backup copies to keep (not days)
  enabled               = true
}

output "database_uuid" {
  value = coolify_postgresql_database.main.uuid
}
