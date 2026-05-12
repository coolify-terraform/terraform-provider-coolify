# ACME Corp Docker Deployments

This scenario tests Docker-based resources including the `:latest` tag
normalization quirk, Docker Compose applications, scheduled tasks, and
persistent storage volumes.

## Resources Tested

| Resource | Purpose |
|---|---|
| `coolify_docker_image_application` | Docker image app with `:latest` tag (normalization test) |
| `coolify_docker_image_application` | Docker image app with specific tag |
| `coolify_docker_compose_application` | Compose-based multi-container app |
| `coolify_scheduled_task` | Cron task attached to app |
| `coolify_storage` | Persistent volume mount |

## API Quirks Verified

- **Quirk #2**: Coolify strips `:latest` from Docker image names. The provider
  must preserve the user's original `nginx:latest` value.
- **ports_exposes**: Verify the configured port survives read-back.

## Running

```bash
export TF_VAR_coolify_endpoint="http://localhost:8000"
export TF_VAR_coolify_token="your-token"
export TF_VAR_server_uuid="your-server-uuid"

terraform test -verbose
```