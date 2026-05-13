# Terraform Provider for Coolify

[![CI](https://github.com/SebTardifLabs/terraform-provider-coolify/actions/workflows/ci.yml/badge.svg)](https://github.com/SebTardifLabs/terraform-provider-coolify/actions/workflows/ci.yml)
![Go Version](https://img.shields.io/github/go-mod/go-version/SebTardifLabs/terraform-provider-coolify)
![License](https://img.shields.io/github/license/SebTardifLabs/terraform-provider-coolify)

A Terraform provider for managing resources in [Coolify](https://coolify.io/), the open-source, self-hosted Platform-as-a-Service. 27 resources, 46 data sources, 570+ tests, 100% API coverage.

## Getting Started

- [Installation](docs/guides/installation.md) -- install and configure the provider
- [Core Concepts](docs/guides/concepts.md) -- understand the resource model
- [Quick Start](docs/guides/quickstart.md) -- deploy your first app in under 5 minutes

## Resources

| Resource | Description |
|----------|-------------|
| `coolify_project` | Manage projects (logical grouping for resources) |
| `coolify_server` | Register and configure servers |
| `coolify_private_key` | Manage SSH keys for server access |
| `coolify_application` | Deploy apps from public Git repositories |
| `coolify_dockerfile_application` | Deploy apps from Dockerfiles |
| `coolify_docker_image_application` | Deploy apps from Docker images (Docker Hub, GHCR, etc.) |
| `coolify_docker_compose_application` | Deploy apps from Docker Compose files |
| `coolify_private_git_application` | Deploy apps from private Git repos (SSH deploy key) |
| `coolify_github_app_application` | Deploy apps via GitHub App integration |
| `coolify_environment` | Manage project environments |
| `coolify_environment_variable` | Manage env vars for applications, services, and databases |
| `coolify_deployment` | Trigger application deployments |
| `coolify_service` | Deploy one-click services from the Coolify catalog |
| `coolify_postgresql_database` | Provision PostgreSQL databases |
| `coolify_mysql_database` | Provision MySQL databases |
| `coolify_mariadb_database` | Provision MariaDB databases |
| `coolify_redis_database` | Provision Redis instances |
| `coolify_mongodb_database` | Provision MongoDB databases |
| `coolify_clickhouse_database` | Provision ClickHouse databases |
| `coolify_keydb_database` | Provision KeyDB databases (Redis-compatible) |
| `coolify_dragonfly_database` | Provision DragonFly databases (Redis-compatible) |
| `coolify_database_backup` | Schedule automated database backups |
| `coolify_s3_storage` | Manage S3 storage destinations for backups |
| `coolify_scheduled_task` | Manage scheduled tasks on applications/services |
| `coolify_storage` | Manage persistent storage volumes |
| `coolify_cloud_token` | Manage cloud provider tokens (Hetzner) |
| `coolify_github_app` | Manage GitHub App integrations |

## Data Sources

| Data Source | Description |
|-------------|-------------|
| `coolify_project` / `coolify_projects` | Read project(s) |
| `coolify_server` / `coolify_servers` | Read server(s) |
| `coolify_server_resources` / `coolify_server_domains` | List resources and domains on a server |
| `coolify_server_validation` | Validate server connectivity |
| `coolify_private_key` / `coolify_private_keys` | Read SSH key(s) |
| `coolify_application` / `coolify_applications` | Read application(s) |
| `coolify_application_logs` | Read application logs |
| `coolify_database` / `coolify_databases` | Read database(s) |
| `coolify_environment` / `coolify_environments` | Read environment(s) |
| `coolify_environment_variable` / `coolify_environment_variables` | Read / list env vars for an application, service, or database |
| `coolify_deployment` / `coolify_deployments` | Read / list deployments for an application |
| `coolify_service` / `coolify_services` | Read service(s) |
| `coolify_s3_storage` / `coolify_s3_storages` | Read S3 storage destination(s) |
| `coolify_scheduled_task` / `coolify_scheduled_tasks` / `coolify_task_executions` | Read scheduled task(s) and executions |
| `coolify_storage` / `coolify_storages` | Read / list persistent storage volumes |
| `coolify_cloud_token` / `coolify_cloud_tokens` | Read cloud token(s) |
| `coolify_github_app` / `coolify_github_apps` / `coolify_github_app_repositories` / `coolify_github_app_branches` | Read GitHub App(s), repos, branches |
| `coolify_backup_executions` | List backup execution history |
| `coolify_resources` | List all resources on a server |
| `coolify_team` / `coolify_teams` / `coolify_team_members` | Read team(s) and members |
| `coolify_health` | Read Coolify instance health status |
| `coolify_version` | Read the Coolify instance version |
| `coolify_hetzner_images` / `coolify_hetzner_locations` / `coolify_hetzner_server_types` / `coolify_hetzner_ssh_keys` | Read Hetzner cloud resources |

## Features

- **Configurable timeouts** on all application resources (`timeouts = { create = "30m" }`)
- **Input validators**: UUID format, FQDN format, cron syntax, port range (1-65535), build pack enum, env var name format
- **Provider health check**: validates API connection during configuration
- **Import support**: all resources support `terraform import` ([guide](docs/guides/import.md))
- **Retryable HTTP client**: automatic retry on 429/5xx with exponential backoff
- **Versioned User-Agent** header for API diagnostics

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.26 (for building from source)
- A running [Coolify](https://coolify.io/) instance (v4.x)

## Usage

```hcl
terraform {
  required_providers {
    coolify = {
      source = "SebTardifLabs/coolify"
    }
  }
}

provider "coolify" {
  endpoint = "https://your-coolify-instance"
  token    = "your-api-token"
}

resource "coolify_project" "example" {
  name        = "my-project"
  description = "Managed by Terraform"
}

resource "coolify_postgresql_database" "db" {
  name         = "my-database"
  project_uuid = coolify_project.example.uuid
  server_uuid  = "your-server-uuid"
  image        = "postgres:16"
  postgres_user = "app"
  postgres_db  = "myapp"
}

resource "coolify_application" "web" {
  name           = "my-web-app"
  project_uuid   = coolify_project.example.uuid
  server_uuid    = "your-server-uuid"
  git_repository = "https://github.com/example/app"
  git_branch     = "main"
  build_pack     = "nixpacks"
  ports_exposes  = "3000"
  fqdn           = "https://app.example.com"
}
```

See the [examples/](examples/) directory for more. The
[examples/scenarios/](examples/scenarios/) directory has 7 real-world
scenarios tested against a live Coolify instance:

| Scenario | What it tests |
|---|---|
| `acme-website` | Project, PostgreSQL, application, env vars |
| `acme-api` | Dockerfile + Docker image apps, Redis, scheduled tasks, backups |
| `acme-multi-env` | Terraform modules for dev/staging environments |
| `acme-databases` | All 8 database engines (PostgreSQL, MySQL, MariaDB, Redis, MongoDB, ClickHouse, KeyDB, Dragonfly) |
| `acme-platform` | Private keys, environments, storage, data sources |
| `acme-docker` | Docker image tag normalization, scheduled tasks, storage |
| `acme-integrations` | Managed services from the Coolify catalog |

## Authentication

The provider requires a Coolify API token. Generate one in the Coolify UI under **Security > API Tokens**.

| Attribute | Environment Variable | Description |
|-----------|---------------------|-------------|
| `endpoint` | `COOLIFY_ENDPOINT` | Coolify API base URL |
| `token` | `COOLIFY_TOKEN` | API bearer token |

## Development

```bash
make build       # Compile the provider
make test        # Run unit tests (570+ tests, race detector enabled)
make testacc     # Run acceptance tests (needs running Coolify instance)
make lint        # Run golangci-lint
make fmt         # Format code (gofmt + go mod tidy)
make docs        # Regenerate documentation via tfplugindocs
make validate    # Check HCL formatting in examples/
make install     # Install provider to local Go bin
```

For local provider testing with `dev_overrides`, acceptance test setup, and
project structure details, see [CONTRIBUTING.md](CONTRIBUTING.md) and
[TESTING.md](TESTING.md).

### CI Pipeline

13 jobs in the CI workflow: Detect Changes, Test, Lint, Validate Examples,
Docs, Govulncheck, Trivy, Gitleaks, GoReleaser Check, Scenario Tests,
Acceptance Tests, Spec Freshness (weekly), and a CI gate job.

## License

MPL-2.0