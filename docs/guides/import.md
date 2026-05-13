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

The `coolify_github_app` resource uses a **numeric ID** (not a UUID):

```bash
terraform import coolify_github_app.my_app 42
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
```

~> **Important:** Application imports default to `environment_name = "production"`. If your
application is in a different environment, set `environment_name` in your `.tf` file to match
**before** running `terraform plan`, otherwise Terraform will propose replacing the resource.
This applies to all 5 application types: `coolify_application`, `coolify_docker_image_application`,
`coolify_dockerfile_application`, `coolify_github_app_application`, and `coolify_private_git_application`.

## Known Limitations

The Coolify API does not return all fields in its GET responses. After
importing a resource, the following fields may be missing from state and
must be set in your `.tf` configuration before running `terraform plan`:

| Resource Type | Fields the API may not return |
|---|---|
| All databases | `project_uuid`, `server_uuid`, `environment_name` |
| All applications | `project_uuid`, `server_uuid`, `environment_name` |
| `coolify_github_app` | `client_secret`, `private_key_uuid` (write-only, never returned by the API) |
| Database backups | `database_uuid` |

If these fields are missing, `terraform plan` will either show a diff
or propose replacing the resource. Set them in your config to match
your actual Coolify setup.

Additionally, Coolify normalizes some input values:

| Field | What Coolify does |
|---|---|
| `git_repository` | Strips `https://github.com/` prefix (e.g. `https://github.com/org/repo` becomes `org/repo`) |
| `docker_image` | Strips image tags (e.g. `redis:7-alpine` becomes `redis`, `nginx:latest` becomes `nginx`) |
| `dockerfile_location` | Must be base64-encoded Dockerfile content (not a file path); not returned on GET |
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
