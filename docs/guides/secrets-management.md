---
page_title: "Secrets Management"
subcategory: "Guides"
description: |-
  Best practices for managing passwords, tokens, and other sensitive values with the Coolify provider.
---

# Secrets Management

The Coolify provider manages passwords, API tokens, SSH keys, and other
sensitive values across many resources. This guide covers how to handle
them safely.

## How the Provider Handles Secrets

Fields marked `Sensitive: true` in the schema are:

- **Redacted in plan output:** Terraform shows `(sensitive value)` instead of the actual value
- **Redacted in TRACE logs:** the provider replaces passwords, tokens, and keys with `[REDACTED]` in API request/response logs
- **Stored in state:** Terraform state files contain the actual values in plaintext

## Token Permissions

The Coolify API token's permission level affects which fields the provider
can read. This is the most common source of unexpected behavior for new
users.

| Permission | Behavior |
|------------|----------|
| Default (no special permissions) | Sensitive fields (passwords, keys, Dockerfiles) are **hidden** from API responses. The provider sees empty strings. |
| `read:sensitive` | Sensitive fields are returned in API responses. Read-only access. |
| `root` | Full access to all fields and all operations. **Recommended for Terraform.** |

### What breaks without `root` or `read:sensitive`

- `terraform plan` shows diffs on password fields (API returns empty, state has the value)
- `terraform import` imports resources with empty sensitive fields
- Database password attributes read back as empty strings
- Private key content is not returned

~> **Recommendation:** create a dedicated API token with `root` permission
for Terraform. This token should only be used by Terraform, not shared
with other tools or users.

## Providing Secrets to Terraform

### Environment variables (recommended for CI/CD)

```bash
# Provider authentication
export COOLIFY_ENDPOINT="https://coolify.example.com"
export COOLIFY_TOKEN="your-api-token"

# Resource secrets
export TF_VAR_db_password="secure-password-here"
export TF_VAR_private_key_content="$(cat ~/.ssh/deploy_key)"
```

```hcl
variable "db_password" {
  type      = string
  sensitive = true
}

resource "coolify_database_postgresql" "db" {
  # ...
  postgres_password = var.db_password
}
```

### Variable files (local development)

Create a `terraform.tfvars` file and add it to `.gitignore`:

```hcl
# terraform.tfvars (DO NOT commit this file)
db_password     = "local-dev-password"
deploy_key_uuid = "abc-123-def-456"
```

```bash
echo "terraform.tfvars" >> .gitignore
```

Create a `terraform.tfvars.example` template for teammates:

```hcl
# terraform.tfvars.example (safe to commit)
db_password     = "CHANGE_ME"
deploy_key_uuid = "CHANGE_ME"
```

### External secret stores (production teams)

For teams managing multiple environments, integrate with a secret store:

```hcl
# HashiCorp Vault example
data "vault_generic_secret" "coolify" {
  path = "secret/coolify/production"
}

resource "coolify_database_postgresql" "db" {
  # ...
  postgres_password = data.vault_generic_secret.coolify.data["db_password"]
}
```

### What to avoid

- **Never hardcode secrets in `.tf` files.** Even with `sensitive = true`,
  the value is in the file and in version control.
- **Never commit `terraform.tfvars` with real values.**
- **Never pass secrets as command-line `-var` arguments.** They appear
  in shell history and process listings.

## Fields the API Hides

Some fields are write-only or hidden by the Coolify API. After import or
if the API token lacks permissions, these fields will be empty in state.

| Resource | Hidden Fields | Behavior |
|----------|--------------|----------|
| `coolify_database_postgresql` | `postgres_password` | Empty on import; set in config to avoid diffs |
| `coolify_database_mysql` | `mysql_password`, `mysql_root_password` | Empty on import |
| `coolify_database_mariadb` | `mariadb_password`, `mariadb_root_password` | Empty on import |
| `coolify_database_mongodb` | `mongo_initdb_root_password` | Empty on import |
| `coolify_database_clickhouse` | `clickhouse_admin_password` | Empty on import |
| `coolify_private_key` | `private_key` | Requires `root` or `read:sensitive` token |
| `coolify_cloud_token` | `token` | Write-only, never returned by API |
| `coolify_github_app` | `client_secret`, `webhook_secret`, `private_key_uuid` | Write-only |
| `coolify_environment_variable` | `value` | May be hidden without sensitive permissions |

**After importing a resource with hidden fields:**

1. Set the field values in your `.tf` configuration
2. Run `terraform plan` to confirm no unexpected changes
3. If a diff appears on a sensitive field, it usually means the API
   returned empty and Terraform wants to set the configured value

## State File Security

Terraform state contains all resource attributes, including sensitive
values, in plaintext. Protect your state:

| Backend | Encryption |
|---------|-----------|
| Local (default) | No encryption. State file on disk contains secrets. |
| S3 | Enable server-side encryption (SSE-S3 or SSE-KMS) |
| Terraform Cloud / HCP | Encrypted at rest, access-controlled |
| Azure Blob | Enable blob encryption |
| GCS | Encrypted by default |

**Minimum recommendation:** use a remote backend with encryption.
Never commit `terraform.tfstate` to version control.

```hcl
# Example: S3 backend with encryption
terraform {
  backend "s3" {
    bucket         = "my-terraform-state"
    key            = "coolify/production/terraform.tfstate"
    region         = "us-east-1"
    encrypt        = true
    dynamodb_table = "terraform-locks"
  }
}
```

## Working Example

A complete configuration with proper secret handling:

```hcl
terraform {
  required_providers {
    coolify = {
      source = "coolify-terraform/coolify"
    }
  }
}

# Authenticate via environment variables:
#   export COOLIFY_ENDPOINT="https://coolify.example.com"
#   export COOLIFY_TOKEN="your-root-api-token"
provider "coolify" {}

variable "db_password" {
  description = "PostgreSQL password"
  type        = string
  sensitive   = true
}

variable "server_uuid" {
  description = "Target server UUID"
  type        = string
}

resource "coolify_project" "app" {
  name = "secure-app"
}

resource "coolify_database_postgresql" "db" {
  name              = "app-db"
  project_uuid      = coolify_project.app.uuid
  server_uuid       = var.server_uuid
  image             = "postgres:16"
  postgres_user     = "app"
  postgres_password = var.db_password
  postgres_db       = "myapp"
  is_public         = false
}
```

Apply with environment variables:

```bash
export TF_VAR_db_password="$(openssl rand -base64 24)"
export TF_VAR_server_uuid="your-server-uuid"
terraform apply
```
