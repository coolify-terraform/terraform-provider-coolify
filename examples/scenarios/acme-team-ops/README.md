# ACME Corp Team Operations

Demonstrates data-source-heavy patterns for operational visibility into
a Coolify instance:

- **Team discovery**: Read team info and list team members
- **Server inventory**: List all servers, read details for a specific server
- **Project audit**: List all projects and their environments
- **Version check**: Read the current Coolify version

## Use case

Operators who want to audit their Coolify infrastructure state using
Terraform data sources. Unlike other scenarios that focus on creating
resources, this scenario focuses on reading and observing existing state.

## Resources used

| Type | Resource |
|------|----------|
| Resource | `coolify_project` |
| Data source | `coolify_version`, `coolify_team`, `coolify_team_members` |
| Data source | `coolify_servers`, `coolify_server` |
| Data source | `coolify_projects`, `coolify_environments` |

## Running

```bash
export TF_VAR_coolify_endpoint="http://localhost:8000"
export TF_VAR_coolify_token="your-api-token"
export TF_VAR_server_uuid="your-server-uuid"

terraform init
terraform apply
```