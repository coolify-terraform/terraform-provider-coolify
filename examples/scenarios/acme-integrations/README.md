# ACME Corp External Integrations

This scenario tests external integration resources through a managed service
from the Coolify catalog. GitHub App registration is documented separately
because application creation requires a live GitHub App installation.

## Resources Tested

| Resource | Purpose |
|---|---|
| `coolify_service` | Managed service from Coolify catalog (uptime-kuma) |

## How It Works

The managed service uses the `uptime-kuma` type from Coolify's built-in
service catalog. Coolify creates the Docker containers for the service.

GitHub App-backed applications are intentionally not part of this scenario.
Coolify verifies repository access during `POST /applications/private-github-app`,
so a realistic end-to-end example needs a live GitHub App installation with
repository access.

## Running

```bash
export TF_VAR_coolify_endpoint="http://localhost:8000"
export TF_VAR_coolify_token="your-token"
export TF_VAR_server_uuid="your-server-uuid"

terraform test -verbose
```