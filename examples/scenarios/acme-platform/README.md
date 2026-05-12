# ACME Corp Platform Setup

This scenario tests infrastructure-level resources that other scenarios depend
on: SSH keys, custom environments, persistent storage, cloud tokens, and
read-only data sources.

## Resources Tested

| Resource | Purpose |
|---|---|
| `coolify_private_key` | SSH key for server authentication |
| `coolify_project` | Project container |
| `coolify_environment` | Custom environment (staging) |
| `coolify_application` | App with attached storage |
| `coolify_storage` | Persistent volume mount |
| `coolify_cloud_token` | API access token |

## Data Sources Tested

| Data Source | Purpose |
|---|---|
| `coolify_version` | API version check |
| `coolify_health` | Health endpoint |
| `coolify_server` | Server details and status |
| `coolify_teams` | Team listing |

## Prerequisites

- A running Coolify instance with API enabled
- A registered and reachable server

## Running

```bash
export TF_VAR_coolify_endpoint="http://localhost:8000"
export TF_VAR_coolify_token="your-token"
export TF_VAR_server_uuid="your-server-uuid"

terraform test -verbose
```