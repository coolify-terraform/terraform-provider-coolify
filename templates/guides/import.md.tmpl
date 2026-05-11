---
page_title: "Importing Existing Resources"
subcategory: ""
description: |-
  Guide for importing existing Coolify resources into Terraform state.
---

# Importing Existing Resources

All Coolify provider resources support `terraform import`.

## Finding Resource UUIDs

Use the provider's data sources or the Coolify API:

```bash
curl -s -H "Authorization: Bearer $COOLIFY_TOKEN" \
  "$COOLIFY_ENDPOINT/api/v1/projects" | jq '.[].uuid'
```

## Import Commands

Most resources use a single UUID:

```bash
terraform import coolify_project.main <project-uuid>
terraform import coolify_server.web <server-uuid>
terraform import coolify_application.api <app-uuid>
terraform import coolify_postgresql_database.db <db-uuid>
terraform import coolify_service.plausible <service-uuid>
terraform import coolify_s3_storage.backups <storage-uuid>
```

Resources with composite IDs:

```bash
# Database backup: database_uuid:backup_id
terraform import coolify_database_backup.daily <db-uuid>:<backup-id>

# Environment variable: type:parent_uuid:env_var_uuid
terraform import coolify_environment_variable.db_url application:<app-uuid>:<env-uuid>
terraform import coolify_environment_variable.redis service:<svc-uuid>:<env-uuid>
terraform import coolify_environment_variable.pg_pass database:<db-uuid>:<env-uuid>

# Scheduled task: type:parent_uuid:task_uuid
terraform import coolify_scheduled_task.backup application:<app-uuid>:<task-uuid>
terraform import coolify_scheduled_task.check service:<svc-uuid>:<task-uuid>

# Persistent storage: type:parent_uuid:storage_uuid
terraform import coolify_storage.data application:<app-uuid>:<storage-uuid>
terraform import coolify_storage.vol service:<svc-uuid>:<storage-uuid>
terraform import coolify_storage.pgdata database:<db-uuid>:<storage-uuid>
```

## Workflow

1. Write the resource block in your `.tf` file
2. Run `terraform import <resource>.<name> <id>`
3. Run `terraform plan` to see what Terraform detects
4. Fill in required attributes not stored in state
5. Run `terraform plan` again to confirm no changes
