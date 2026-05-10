# 03 - Database with Backup: provision a database and schedule backups
#
# Creates a PostgreSQL database, an S3 storage destination, and a
# scheduled backup that runs daily at 2 AM UTC.
#
# Prerequisites:
#   - Coolify with a registered server
#   - S3-compatible storage (AWS S3, MinIO, Backblaze B2, etc.)
#
# Run:
#   terraform init
#   terraform plan -var-file=secrets.tfvars
#   terraform apply -var-file=secrets.tfvars

terraform {
  required_providers {
    coolify = {
      source = "SebTardif/coolify"
    }
  }
}

provider "coolify" {}

variable "server_uuid" {
  type = string
}

variable "s3_access_key" {
  type      = string
  sensitive = true
}

variable "s3_secret_key" {
  type      = string
  sensitive = true
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

resource "coolify_s3_storage" "backups" {
  name       = "db-backups"
  access_key = var.s3_access_key
  secret_key = var.s3_secret_key
  bucket     = "coolify-backups"
  endpoint   = "https://s3.amazonaws.com"
  region     = "us-east-1"
}

resource "coolify_database_backup" "daily" {
  database_uuid = coolify_postgresql_database.main.uuid
  s3_storage_id = coolify_s3_storage.backups.uuid
  frequency     = "0 2 * * *"
  retain_days   = 7 # Number of backup copies to keep (not days)
  enabled       = true
}

output "database_uuid" {
  value = coolify_postgresql_database.main.uuid
}
