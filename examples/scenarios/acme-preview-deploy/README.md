# ACME Corp Preview Deployments

This scenario tracks PR-based preview environments that Coolify already
created (webhook or UI). Terraform does not create those previews.

1. **Application** (`coolify_application_dockerfile`) deployed from an
   inline Dockerfile (any application type supports previews).
2. **Preview tracking** (`coolify_application_preview`) for pull requests
   #1 and #2. `terraform destroy` DELETEs each tracked preview if it exists.

Domain PATCH examples live in
[`examples/resources/coolify_application_preview/resource.tf`](../../resources/coolify_application_preview/resource.tf).

## Resources Tested

| Resource | Purpose |
|----------|---------|
| `coolify_project` | Project container |
| `coolify_application_dockerfile` | Application to attach previews to |
| `coolify_application_preview` (x2) | Track PR 1/2 (Coolify creates the preview) |

## Running

```bash
export TF_VAR_coolify_endpoint="http://localhost:8000"
export TF_VAR_coolify_token="your-token"
export TF_VAR_server_uuid="your-server-uuid"

terraform test -verbose
```
