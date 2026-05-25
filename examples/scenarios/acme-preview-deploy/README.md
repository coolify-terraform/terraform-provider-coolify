# ACME Corp Preview Deployments

This scenario sets up PR-based preview environments, demonstrating:

1. **Application** (`coolify_application_dockerfile`) deployed from an
   inline Dockerfile (any application type supports previews).
2. **Preview environments** (`coolify_application_preview`) created for
   specific pull requests, cleaned up automatically on `terraform destroy`.

## Resources Tested

| Resource | Purpose |
|----------|---------|
| `coolify_project` | Project container |
| `coolify_application_dockerfile` | Application to attach previews to |
| `coolify_application_preview` (x2) | PR preview environments for PRs #1 and #2 |

## Running

```bash
export TF_VAR_coolify_endpoint="http://localhost:8000"
export TF_VAR_coolify_token="your-token"
export TF_VAR_server_uuid="your-server-uuid"

terraform test -verbose
```
