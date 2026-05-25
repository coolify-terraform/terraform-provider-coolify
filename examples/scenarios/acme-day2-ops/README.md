# ACME Corp Day-2 Operations

This scenario demonstrates operational lifecycle management for deployed
resources. It creates a database and an application, then uses
`coolify_resource_action` to restart the application with a trigger-based
re-execution pattern.

## Resources Tested

| Resource | Purpose |
|---|---|
| `coolify_project` | Project container |
| `coolify_database_postgresql` | PostgreSQL database |
| `coolify_application_docker_image` | Docker image application (nginx) |
| `coolify_resource_action` | Restart action with triggers map |

## Key Concepts

- **Trigger-based re-execution**: The `triggers` map on `coolify_resource_action`
  forces the action to fire again whenever a value changes. Bumping `app_version`
  from `v1` to `v2` triggers a restart.
- **Fire-and-forget semantics**: The action runs on create. Read and delete are
  no-ops -- Terraform tracks what was last executed but cannot undo an action.
- **RequiresReplace on all attributes**: Changing any attribute (including
  triggers) forces destroy+recreate, which re-fires the action.

## Running

```bash
export TF_VAR_coolify_endpoint="http://localhost:8000"
export TF_VAR_coolify_token="your-token"
export TF_VAR_server_uuid="your-server-uuid"

terraform test -verbose
```
