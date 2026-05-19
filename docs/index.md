---
page_title: "Coolify Provider"
subcategory: ""
description: |-
  Manage your Coolify infrastructure as code: applications, databases, servers, backups, and environment variables, all version-controlled and reproducible.
---

# Coolify Provider

Define your entire [Coolify](https://coolify.io/) infrastructure as code. Deploy applications, provision databases, configure backups, and manage environment variables, all version-controlled and reproducible with `terraform apply`.

Coolify gives you a self-hosted PaaS with a great UI. This provider adds what the UI cannot: reproducible environments, pull-request-based infrastructure reviews, disaster recovery from state, and CI/CD integration. Whether you run a single VPS or manage Coolify for a team, Terraform turns your infrastructure into a reviewable, testable, git-tracked configuration.

**26 resources, 44 data sources, 99%+ Coolify API coverage.**

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= 1.6
- A running [Coolify](https://coolify.io/) instance (v4.x)

## Authentication

The provider requires a Coolify API token.

~> **The Coolify API is disabled by default.** You must enable it in the
Coolify dashboard under **Settings** before generating a token. Without
this, all provider operations fail with "Unauthenticated."

Generate a token in the Coolify UI under **Security > API Tokens**.

You can configure credentials in two ways:

### Environment variables (recommended for CI/CD)

```bash
export COOLIFY_ENDPOINT="https://coolify.example.com"
export COOLIFY_TOKEN="your-api-token"
```

### Provider block

```terraform
provider "coolify" {
  endpoint = var.coolify_endpoint # Or set COOLIFY_ENDPOINT env var
  token    = var.coolify_token    # Or set COOLIFY_TOKEN env var
}
```

## Getting Started

- **[Quick Start](guides/quickstart)** - Deploy your first application in under 5 minutes
- **[Installation](guides/installation)** - Detailed setup and configuration options
- **[Concepts](guides/concepts)** - How Coolify resources map to Terraform
- **[Import Guide](guides/import)** - Bring existing Coolify resources under Terraform management
- **[Scenario Examples](guides/scenario-testing)** - Real-world ACME Corp deployment scenarios
- **[Troubleshooting](guides/troubleshooting)** - Debugging tips and diagnostic logs
- **[API Contract Accuracy](guides/api-contract-accuracy)** - OpenAPI spec vs real Coolify source code

## Resource Overview

| Category | Resources |
|----------|-----------|
| Infrastructure | `coolify_project`, `coolify_environment`, `coolify_server`, `coolify_hetzner_server`, `coolify_private_key`, `coolify_cloud_token`, `coolify_github_app` |
| Applications | `coolify_application`, `coolify_application_dockerfile`, `coolify_application_docker_image`, `coolify_application_private_git`, `coolify_application_github_app` |
| Databases | `coolify_database_postgresql`, `coolify_database_mysql`, `coolify_database_mariadb`, `coolify_database_redis`, `coolify_database_mongodb`, `coolify_database_clickhouse`, `coolify_database_keydb`, `coolify_database_dragonfly` |
| Services | `coolify_service` |
| Operations | `coolify_environment_variable`, `coolify_storage`, `coolify_scheduled_task`, `coolify_deployment`, `coolify_database_backup` |

All resources support `terraform import`. See the [Import Guide](guides/import)
for composite ID formats, known limitations, and field normalization behaviors.

## Coolify API Behaviors

Coolify's API normalizes certain values. The provider handles these
automatically, but they are worth knowing:

| Behavior | Affected Resources | What happens |
|---|---|---|
| **Git URL prefix stripping** | All application types | `https://github.com/org/repo` is stored as `org/repo` |
| **Docker image tag stripping** | `coolify_application_docker_image` | `redis:7-alpine` is stored as `redis` (all tags, not just `:latest`) |
| **Port override** | `coolify_application_dockerfile` | Coolify may return a different port than configured |
| **Immutable fields omitted** | All databases, apps, services | `project_uuid`, `server_uuid`, `environment_name` not returned by GET |
| **Storage name prefixing** | `coolify_storage` | Coolify prepends an internal resource UUID to the storage name |
| **Base64 encoding required** | `coolify_application_dockerfile` | `dockerfile_location` must be base64-encoded despite the field name; use `base64encode()` |
| **Application refresh can lag create** | All applications | Terraform fails with `Application created but refresh failed` if Coolify returns a UUID but the app is not yet readable through the API |
| **Cloud token validation** | `coolify_cloud_token` | Coolify validates the token against the cloud provider's API on creation (cannot use placeholder values) |
| **GitHub App repository validation** | `coolify_application_github_app` | Coolify verifies repository access during create, so the referenced `coolify_github_app` must have installation access to the target repository |
| **Async deletion** | `coolify_project` | Child resources are deleted asynchronously; project delete retries automatically |

The provider preserves your configured values for all normalized fields.
You should not see diffs on `terraform plan` after the initial apply.

<!-- schema generated by tfplugindocs -->
## Schema

### Optional

- `ca_cert` (String) PEM-encoded CA certificate to trust for TLS connections to the Coolify API. Use this when your Coolify instance uses a self-signed certificate or an internal CA. Env: `COOLIFY_CA_CERT`.
- `endpoint` (String) Coolify API endpoint. Env: COOLIFY_ENDPOINT.
- `insecure` (Boolean) Skip TLS certificate verification. **Not recommended for production.** Use `ca_cert` instead when possible. Env: `COOLIFY_INSECURE`.
- `retry_max` (Number) Maximum number of API request retries (default: 3).
- `retry_max_wait` (Number) Maximum wait between retries in seconds (default: 30).
- `retry_min_wait` (Number) Minimum wait between retries in seconds (default: 1).
- `token` (String, Sensitive) Coolify API token. Env: COOLIFY_TOKEN.