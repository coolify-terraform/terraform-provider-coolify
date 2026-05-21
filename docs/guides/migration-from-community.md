---
page_title: "Migration from Community Provider - coolify Provider"
subcategory: ""
description: |-
  How to migrate from the SierraJC community Coolify provider to this provider.
---

# Migration from Community Provider

This guide helps you migrate from the
[SierraJC community provider](https://registry.terraform.io/providers/SierraJC/coolify)
(`SierraJC/coolify`) to this provider (`SebTardifLabs/coolify`).

## Why Migrate

This provider is a complete rewrite using the Terraform Plugin Framework
(not the older SDK v2). It offers:

- **33 resources** vs. ~10 in the community provider
- **44 data sources** with filtering support
- **770+ tests** (unit + acceptance + scenario)
- Database backup management, scheduled tasks, storage volumes,
  cloud tokens, GitHub App sources, and resource actions (start/stop/restart)
- Contract-verified field coverage against the real Coolify source code
- Active maintenance with weekly upstream drift detection

## Step 1: Update the Provider Block

```hcl
# Before (SierraJC)
terraform {
  required_providers {
    coolify = {
      source  = "SierraJC/coolify"
      version = "~> 0.10"
    }
  }
}

provider "coolify" {
  host  = "https://coolify.example.com"
  token = var.coolify_token
}

# After (SebTardifLabs)
terraform {
  required_providers {
    coolify = {
      source  = "SebTardifLabs/coolify"
    }
  }
}

provider "coolify" {
  endpoint = "https://coolify.example.com"  # renamed from "host"
  token    = var.coolify_token
}
```

Key change: the `host` attribute is renamed to `endpoint`.

## Step 2: Rename Resources

| SierraJC resource | This provider | Action |
|-------------------|--------------|--------|
| `coolify_project` | `coolify_project` | Compatible, no rename needed |
| `coolify_private_key` | `coolify_private_key` | Compatible, no rename needed |
| `coolify_server` | `coolify_server` | Compatible (more fields available) |
| `coolify_service` | `coolify_service` | Compatible core schema |
| `coolify_postgresql_database` | `coolify_database_postgresql` | Name order swapped |
| `coolify_application` | See note below | Split by deployment type |
| `coolify_application_envs` | `coolify_environment_variable` | One resource per variable |
| `coolify_service_envs` | `coolify_environment_variable` | One resource per variable |

### Application resources

The community provider uses a single `coolify_application` resource. This
provider splits applications by deployment type for schema precision:

| Deployment type | This provider resource |
|----------------|----------------------|
| Dockerfile (git repo with Dockerfile) | `coolify_application_dockerfile` |
| Docker image (pre-built image) | `coolify_application_docker_image` |
| Docker Compose | `coolify_application_docker_compose` |
| Public git repository | `coolify_application_public_git` |
| Private git repository (SSH key) | `coolify_application_private_git` |
| GitHub App source | `coolify_application_github_app` |

Choose the resource type that matches your deployment method.

### Environment variables

The community provider uses bulk resources (`coolify_application_envs`,
`coolify_service_envs`). This provider offers both approaches:

- `coolify_environment_variable`: One resource per variable (recommended
  for fine-grained control)
- `coolify_envs_bulk`: Bulk set/replace all variables at once

## Step 3: Migrate State

After updating your `.tf` files, use `terraform state mv` to rename
resources in state without destroying and recreating them:

```bash
# Database name order swap
terraform state mv coolify_postgresql_database.mydb coolify_database_postgresql.mydb

# Application type split (example: Dockerfile-based app)
terraform state mv coolify_application.api coolify_application_dockerfile.api

# Bulk env vars to individual variables (requires manual split)
# No direct state migration; re-import individual variables instead:
terraform import coolify_environment_variable.db_url \
  "application:<app-uuid>:<env-var-uuid>"
```

## Step 4: Re-initialize

```bash
# Remove old provider lock
rm -rf .terraform .terraform.lock.hcl

# Initialize with the new provider
terraform init

# Verify the plan shows no unexpected changes
terraform plan
```

## Step 5: Verify

Run `terraform plan` and confirm:
- No resources are marked for destruction
- No resources are marked for creation (unless you added new ones)
- Any diffs are limited to new optional fields with defaults

## New Features After Migration

After migrating, you gain access to resources and data sources not
available in the community provider:

| Category | Resources |
|----------|----------|
| Databases | `coolify_database_mysql`, `coolify_database_mariadb`, `coolify_database_mongodb`, `coolify_database_redis`, `coolify_database_clickhouse`, `coolify_database_dragonfly`, `coolify_database_keydb` |
| Backups | `coolify_database_backup`, `data.coolify_backup_executions` |
| Operations | `coolify_resource_action` (start/stop/restart), `coolify_scheduled_task`, `coolify_deployment` |
| Infrastructure | `coolify_server_hetzner`, `coolify_cloud_token`, `coolify_storage` |
| Git sources | `coolify_github_app`, `coolify_application_github_app`, `coolify_application_preview` |
| Configuration | `coolify_api_settings`, `coolify_envs_bulk`, `coolify_environment` |

See the [full resource documentation](/docs) for details on each.