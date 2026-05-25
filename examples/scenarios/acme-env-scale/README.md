# ACME Corp Environment Variables at Scale

This scenario demonstrates managing environment variables across multiple
applications using `coolify_envs_bulk` for atomic bulk operations. Shared
variables (database URL, log level, app environment) are defined once in
`locals` and merged with per-app overrides via `merge()`.

## Resources Used

| Resource | Purpose |
|---|---|
| `coolify_project` | Project container |
| `coolify_database_postgresql` | Shared database (source of connection string) |
| `coolify_application_docker_image` | API application (nginx) |
| `coolify_application_docker_image` | Worker application (redis) |
| `coolify_envs_bulk` | Atomic bulk env vars for API app |
| `coolify_envs_bulk` | Atomic bulk env vars for Worker app |

## Pattern Highlights

- **Shared variables via locals**: Define common env vars once, reuse everywhere.
- **`merge()` for per-app overrides**: Combine shared vars with app-specific ones.
- **Atomic bulk updates**: `coolify_envs_bulk` sets all variables in a single API call,
  preventing partial state where some vars are updated and others are not.

## Running

```bash
export TF_VAR_coolify_endpoint="http://localhost:8000"
export TF_VAR_coolify_token="your-token"
export TF_VAR_server_uuid="your-server-uuid"

terraform test -verbose
```
