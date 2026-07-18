---
page_title: "Importing Existing Resources"
subcategory: ""
description: |-
  Guide for importing existing Coolify resources into Terraform state.
---

# Importing Existing Resources

All stateful Coolify provider resources support `terraform import`. Action and validation resources (`coolify_api_settings`, `coolify_application_preview`, `coolify_cloud_token_validate`, `coolify_server_validate`, `coolify_resource_action`) are lifecycle-only and do not support import.

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
terraform import coolify_database_postgresql.db <db-uuid>
terraform import coolify_service.plausible <service-uuid>
terraform import coolify_private_key.deploy <key-uuid>
terraform import coolify_cloud_token.hetzner <token-uuid>
```

### Compound Import Format (Recommended for Applications, Databases, and Services)

Applications, databases, and services support an extended compound import format
that populates `project_uuid`, `server_uuid`, and `environment_name` automatically:

```bash
# Format: project_uuid:server_uuid:environment_name:resource_uuid
terraform import coolify_application.api \
  <project-uuid>:<server-uuid>:production:<app-uuid>

terraform import coolify_database_postgresql.db \
  <project-uuid>:<server-uuid>:production:<db-uuid>

terraform import coolify_service.plausible \
  <project-uuid>:<server-uuid>:production:<service-uuid>
```

This avoids post-import diffs for `project_uuid`, `server_uuid`, and
`environment_name` (which the API may not return in GET responses). The simple
UUID format still works but may require you to set these fields manually in
your `.tf` configuration.

-> **Warning:** The `server_uuid` segment must be the server that actually hosts
the resource. Coolify application GET responses do not return `server_uuid`, so
a wrong value is not corrected on refresh. On replace, Terraform would recreate
the resource on the wrong server. Application compound import validates that
the application appears in `GET /servers/{server_uuid}/resources` and fails if
it does not.

~> **Note:** Top-level S3 storages are managed in the Coolify web UI. When
`coolify_database_backup` uses `save_s3 = true`, set `s3_storage_uuid` to an
existing storage UUID.


The `coolify_github_app` resource uses the **GitHub App ID** (shown as "App Id" in the Coolify UI under Sources), not a UUID or internal database ID:

```bash
terraform import coolify_github_app.my_app 12345
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

# Deployment: application_uuid:deployment_uuid
terraform import coolify_deployment.web <app-uuid>:<deployment-uuid>

# Environment: project_uuid:environment_name
terraform import coolify_environment.staging <project-uuid>:staging

# Bulk environment variables: resource_type/resource_uuid
terraform import coolify_envs_bulk.app application/<app-uuid>
terraform import coolify_envs_bulk.db database/<db-uuid>

# Backup execution: database_uuid:backup_uuid:execution_uuid
terraform import coolify_backup_execution.old <db-uuid>:<backup-uuid>:<exec-uuid>
```

~> **Important:** Application imports default to `environment_name = "production"`. If your
application is in a different environment, set `environment_name` in your `.tf` file to match
**before** running `terraform plan`, otherwise Terraform will propose replacing the resource.
This applies to all 5 application types: `coolify_application`, `coolify_application_docker_image`,
`coolify_application_dockerfile`, `coolify_application_github_app`, and `coolify_application_private_git`.

## Known Limitations

The Coolify API does not return all fields in its GET responses. After
importing a resource, the following fields may be missing from state and
must be set in your `.tf` configuration before running `terraform plan`:

| Resource Type | Fields the API may not return |
|---|---|
| `coolify_database_postgresql` | `postgres_password`, `project_uuid`, `server_uuid`, `environment_name` |
| `coolify_database_mysql` | `mysql_password`, `mysql_root_password`, `project_uuid`, `server_uuid`, `environment_name` |
| `coolify_database_mariadb` | `mariadb_password`, `mariadb_root_password`, `project_uuid`, `server_uuid`, `environment_name` |
| `coolify_database_mongodb` | `mongo_initdb_root_password`, `project_uuid`, `server_uuid`, `environment_name` |
| `coolify_database_clickhouse` | `clickhouse_admin_password`, `project_uuid`, `server_uuid`, `environment_name` |
| `coolify_database_redis`, `coolify_database_keydb`, `coolify_database_dragonfly` | `project_uuid`, `server_uuid`, `environment_name` |
| All applications | `project_uuid`, `server_uuid`, `environment_name` |
| `coolify_application_private_git` | `private_key_uuid` (Coolify stores only the linked `private_key_id`, so import cannot reconstruct the original UUID) |
| `coolify_application_github_app` | `github_app_uuid` (Coolify stores the linked GitHub App as `source_id`/`source_type`, so import cannot recover the original UUID) |
| `coolify_service` | `project_uuid`, `server_uuid`, `environment_name`, `type` |
| `coolify_server` | `private_key_uuid` (Coolify stores only the linked `private_key_id`, so import cannot reconstruct the original UUID) |
| `coolify_server_hetzner` | `cloud_provider_token_uuid`, `server_type`, `location`, `image`, `private_key_uuid`, `hetzner_ssh_key_ids`, `cloud_init_script`, `enable_ipv4`, `enable_ipv6`, `instant_validate` (Hetzner-specific fields are only sent at creation time and not returned by the server GET endpoint) |
| `coolify_environment_variable` | `value` (sensitive value is not reliably returned on import; keep it in configuration before the first `terraform plan`) |
| `coolify_deployment` | `triggers`, `wait_for_completion` (Terraform-only behavior controls, not persisted by the Coolify API) |
| `coolify_environment` | `description` (stored in Terraform state only; not returned by the API) |
| `coolify_github_app` | `client_secret`, `webhook_secret`, `private_key_uuid` (`client_secret` and `private_key_uuid` are write-only, and `webhook_secret` is not reliably returned after create/import, including provider-generated values when omitted on create) |
| `coolify_cloud_token` | `token` (write-only, may not be returned by the API) |
| `coolify_private_key` | `private_key` (requires API token with `root` or `read:sensitive` permission; hidden otherwise) |
| Database backups | `database_uuid`, `s3_storage_uuid` when `save_s3 = true` |

If these fields are missing, `terraform plan` will either show a diff
or propose replacing the resource. Set them in your config to match
your actual Coolify setup.

For `coolify_application_github_app`, import also cannot reconstruct
`github_app_uuid` from the API. Keep that field in your configuration
before the first `terraform plan`, or expect a post-import diff.

Additionally, Coolify normalizes some input values:

| Field | What Coolify does |
|---|---|
| `git_repository` | Strips `https://github.com/` prefix (e.g. `https://github.com/org/repo` becomes `org/repo`). The provider reconstructs the full URL on import. |
| `docker_image` | Strips image tags (e.g. `redis:7-alpine` becomes `redis`, `nginx:latest` becomes `nginx`). The provider cannot reconstruct the original tag; expect a one-time diff after import. |
| `dockerfile_location` | For `coolify_application_dockerfile`: base64-encoded Dockerfile content (not a file path); not returned on GET. For other app types: a file path relative to the repository root. |
| `ports_exposes` | May be overridden by Coolify for Dockerfile apps (e.g. returns `80` instead of configured `3000`) |
| Storage `name` | Coolify prepends the application UUID (e.g. `my-vol` becomes `{app-uuid}-my-vol`) |

The provider handles most of these normalizations automatically by
preserving your configured values when Coolify returns a normalized
form. After `terraform import` you may see a one-time diff on the
first `terraform plan` because the imported state uses the API values.

## Workflow

1. Write the resource block in your `.tf` file
2. Run `terraform import <resource>.<name> <id>`
3. Run `terraform plan` to see what Terraform detects
4. Fill in required attributes not stored in state (see above)
5. Run `terraform plan` again to confirm no changes
