# ACME Corp External Integrations

This scenario tests external integration resources: GitHub App registration
(with dummy credentials) and Coolify managed services from the service catalog.

## Resources Tested

| Resource | Purpose |
|---|---|
| `coolify_github_app` | GitHub App with dummy credentials (metadata only) |
| `coolify_service` | Managed service from Coolify catalog (uptime-kuma) |

## How It Works

The GitHub App uses dummy values (app_id=12345, fake client_secret). Coolify
stores the metadata without verifying OAuth connectivity, so CRUD operations
work without a real GitHub App configured.

The managed service uses the `uptime-kuma` type from Coolify's built-in
service catalog. Coolify creates the Docker containers for the service.

## Running

```bash
export TF_VAR_coolify_endpoint="http://localhost:8000"
export TF_VAR_coolify_token="your-token"
export TF_VAR_server_uuid="your-server-uuid"

terraform test -verbose
```