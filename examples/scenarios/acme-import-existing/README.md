# ACME Corp Import Existing Resources

The most common entry point for teams adopting Terraform on an existing
Coolify instance. Instead of recreating resources from scratch, this
scenario shows how to bring pre-existing Coolify resources under
Terraform management.

## The Story

ACME Corp has been running services on Coolify for months, managing
everything through the web UI. They now want to adopt infrastructure
as code with Terraform. Rather than tearing down and rebuilding, they
import their existing project, applications, and environment variables
into Terraform state.

## What Gets Created

| # | Resource | Type | Purpose |
|---|----------|------|---------|
| 1 | `coolify_project.existing` | Project | Groups imported resources |
| 2 | `coolify_application_docker_image.web` | Application | Nginx web server |
| 3 | `coolify_environment_variable.app_env` | Env var | APP_ENV=production |

## Import Workflow

### 1. Discover existing resource UUIDs

Use data sources or the Coolify API:

```bash
# List projects
curl -s -H "Authorization: Bearer $COOLIFY_TOKEN" \
  "$COOLIFY_ENDPOINT/api/v1/projects" | jq '.[] | {uuid, name}'

# List applications in a project environment
curl -s -H "Authorization: Bearer $COOLIFY_TOKEN" \
  "$COOLIFY_ENDPOINT/api/v1/projects/<project-uuid>/production" \
  | jq '.applications[] | {uuid, name}'
```

### 2. Write the Terraform configuration

Define resource blocks in `.tf` files that match your existing setup
(see `main.tf`).

### 3. Import with CLI commands

```bash
# Simple UUID import for projects:
terraform import coolify_project.existing <project-uuid>

# Compound import for applications (recommended):
terraform import coolify_application_docker_image.web \
  <project-uuid>:<server-uuid>:production:<app-uuid>

# Composite key import for environment variables:
terraform import coolify_environment_variable.app_env \
  application:<app-uuid>:<env-var-uuid>
```

### 3b. Or use import blocks (Terraform 1.5+)

See `import.tf` for the declarative alternative. Add import blocks,
run `terraform plan` to preview, then `terraform apply` to execute.

### 4. Reconcile and apply

```bash
# Preview what Terraform detects
terraform plan

# Fill in fields not returned by the API (see Known Limitations below)
# Then confirm no changes
terraform plan
```

## Known Limitations

The Coolify API does not return all fields in GET responses. After
importing, you may need to set these fields manually in your `.tf`
configuration:

| Resource | Fields to set manually |
|----------|----------------------|
| Applications | `project_uuid`, `server_uuid`, `environment_name` (use compound import to avoid this) |
| Databases | `project_uuid`, `server_uuid`, `environment_name`, password fields |
| `coolify_github_app` | `client_secret`, `webhook_secret`, `private_key_uuid` |
| `coolify_private_key` | `private_key` (needs `root` or `read:sensitive` API permission) |
| `coolify_environment_variable` | `value` (sensitive, not reliably returned) |

For the full list, see the [Import Guide](../../docs/guides/import.md).

## Prerequisites

1. A running [Coolify](https://coolify.io/) instance (v4+)
2. An API token (Security > API Tokens)
3. A registered server UUID (Settings > Servers)
4. [Terraform](https://developer.hashicorp.com/terraform/install) >= 1.0

## Usage

```bash
cp terraform.tfvars.example terraform.tfvars
# Edit terraform.tfvars with your values
terraform init
terraform plan
terraform apply
```

## Clean Up

```bash
terraform destroy
```
