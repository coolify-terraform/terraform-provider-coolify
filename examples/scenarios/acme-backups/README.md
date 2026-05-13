# ACME Corp: Database Backup & Restore

This scenario walks through the full database backup lifecycle:

1. **Provision** a PostgreSQL database
2. **Configure** automated daily backups with local retention
3. **Trigger** an immediate backup on creation
4. **Monitor** backup status via the executions data source

## Key concepts

### Backup configuration vs. execution

A `coolify_database_backup` resource is a **schedule** (cron frequency,
retention policy, S3 destination). It does not have a "status" field
because the schedule itself is always either enabled or disabled.

The actual results of each backup run are **executions**. Use the
`coolify_backup_executions` data source to list them:

```hcl
data "coolify_backup_executions" "latest" {
  database_uuid = coolify_postgresql_database.app_db.uuid
  backup_uuid   = coolify_database_backup.daily.uuid
}

# Check the most recent execution
output "latest_status" {
  value = data.coolify_backup_executions.latest.executions[0].status
}
```

Each execution has:
- `uuid` - unique identifier
- `status` - `"success"`, `"failed"`, or `"running"`
- `created_at` - when the backup started
- `size` - backup file size in bytes

### Adding S3 off-site storage

To store backups in S3 (in addition to local), add an S3 storage
resource and enable `save_s3`:

```hcl
resource "coolify_s3_storage" "backups" {
  name       = "backup-storage"
  endpoint   = "https://s3.amazonaws.com"
  bucket     = "acme-backups"
  region     = "us-east-1"
  access_key = var.s3_access_key
  secret_key = var.s3_secret_key
}

resource "coolify_database_backup" "daily" {
  database_uuid         = coolify_postgresql_database.app_db.uuid
  frequency             = "0 2 * * *"
  enabled               = true
  save_s3               = true
  s3_storage_id         = coolify_s3_storage.backups.uuid
  retain_amount_locally = 7
  retain_amount_s3      = 30
}
```

## Running

```bash
export TF_VAR_coolify_endpoint="http://localhost:8000"
export TF_VAR_coolify_token="your-token"
export TF_VAR_server_uuid="your-server-uuid"

terraform init
terraform apply

# After apply, check backup status:
terraform output latest_execution_status
```

## Testing

```bash
terraform test -verbose
```
