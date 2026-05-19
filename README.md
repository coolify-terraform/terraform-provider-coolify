# Terraform Provider for Coolify

[![CI](https://github.com/SebTardifLabs/terraform-provider-coolify/actions/workflows/ci.yml/badge.svg)](https://github.com/SebTardifLabs/terraform-provider-coolify/actions/workflows/ci.yml)
![Go Version](https://img.shields.io/github/go-mod/go-version/SebTardifLabs/terraform-provider-coolify)
![License](https://img.shields.io/github/license/SebTardifLabs/terraform-provider-coolify)

Define your entire [Coolify](https://coolify.io/) infrastructure as code: applications, databases, servers, backups, and environment variables, all version-controlled and reproducible with a single `terraform apply`.

## Why Terraform for Coolify?

Coolify gives you a self-hosted PaaS with a great UI. This provider adds the things the UI cannot:

- **Reproducibility** -- rebuild your entire stack from scratch after a server failure, in minutes instead of hours of clicking
- **Version control** -- review infrastructure changes in pull requests before they hit production
- **Multi-environment consistency** -- stamp out identical dev, staging, and production setups from the same configuration
- **Team collaboration** -- stop sharing screenshots of Coolify settings; share `.tf` files instead
- **Disaster recovery** -- your Terraform state is a complete inventory of what should be running and how it should be configured
- **Automation** -- integrate Coolify deployments into your existing CI/CD pipeline

## Getting Started

- [Quick Start](docs/guides/quickstart.md) -- deploy your first app in under 5 minutes
- [Installation](docs/guides/installation.md) -- install and configure the provider
- [Core Concepts](docs/guides/concepts.md) -- understand the resource model

## Resources

| Resource | Description |
|----------|-------------|
| `coolify_project` | Manage projects (logical grouping for resources) |
| `coolify_server` | Register and configure servers |
| `coolify_server_hetzner` | Provision Hetzner Cloud servers via Coolify |
| `coolify_private_key` | Manage SSH keys for server access |
| `coolify_application` | Deploy apps from public Git repositories |
| `coolify_application_dockerfile` | Deploy apps from Dockerfiles |
| `coolify_application_docker_image` | Deploy apps from Docker images (Docker Hub, GHCR, etc.) |
| `coolify_application_private_git` | Deploy apps from private Git repos (SSH deploy key) |
| `coolify_application_github_app` | Deploy apps via GitHub App integration |
| `coolify_environment` | Manage project environments |
| `coolify_environment_variable` | Manage env vars for applications, services, and databases |
| `coolify_deployment` | Trigger application deployments |
| `coolify_service` | Deploy one-click services from the Coolify catalog |
| `coolify_database_postgresql` | Provision PostgreSQL databases |
| `coolify_database_mysql` | Provision MySQL databases |
| `coolify_database_mariadb` | Provision MariaDB databases |
| `coolify_database_redis` | Provision Redis instances |
| `coolify_database_mongodb` | Provision MongoDB databases |
| `coolify_database_clickhouse` | Provision ClickHouse databases |
| `coolify_database_keydb` | Provision KeyDB databases (Redis-compatible) |
| `coolify_database_dragonfly` | Provision DragonFly databases (Redis-compatible) |
| `coolify_database_backup` | Schedule automated database backups |
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

## What You Can Do

Deploy a full stack (app, database, backups, env vars) in a single `terraform apply`, or adopt the provider incrementally by importing your existing Coolify resources.

**Real-world scenarios included** -- 9 tested ACME Corp examples cover common patterns:

| Scenario | What it deploys |
|---|---|
| [acme-website](examples/scenarios/acme-website) | Project + PostgreSQL + web app + env vars |
| [acme-api](examples/scenarios/acme-api) | Dockerfile + Docker image apps + Redis + scheduled tasks + backups |
| [acme-backups](examples/scenarios/acme-backups) | Backup scheduling, S3 off-site storage, execution monitoring |
| [acme-multi-env](examples/scenarios/acme-multi-env) | Terraform modules for dev/staging environments |
| [acme-databases](examples/scenarios/acme-databases) | All 8 database engines side by side |
| [acme-platform](examples/scenarios/acme-platform) | Private keys, environments, storage, data sources |
| [acme-docker](examples/scenarios/acme-docker) | Docker image tag handling, scheduled tasks, storage |
| [acme-integrations](examples/scenarios/acme-integrations) | Managed services from the Coolify catalog |
| [acme-private-repo](examples/scenarios/acme-private-repo) | SSH deploy key + private Git repo + deployment with wait |

Every scenario has `terraform test` integration tests that run against a real Coolify instance.

## Features

- **Import existing resources** -- bring your current Coolify setup under Terraform management without rebuilding ([guide](docs/guides/import.md))
- **Configurable timeouts** -- handle slow builds gracefully (`timeouts = { create = "30m" }`)
- **Input validation** -- catch mistakes at plan time (invalid UUIDs, bad cron expressions, out-of-range ports)
- **Connection health check** -- the provider validates your API connection before making any changes
- **Reliable API calls** -- automatic retry with exponential backoff on transient failures (429, 5xx, network errors)

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= 1.6
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

resource "coolify_database_postgresql" "db" {
  name          = "my-database"
  project_uuid  = coolify_project.example.uuid
  server_uuid   = "your-server-uuid"
  image         = "postgres:16"
  postgres_user = "app"
  postgres_db   = "myapp"
  # postgres_password omitted here. The provider stores the generated
  # sensitive value in Terraform state after create.
}

resource "coolify_application" "web" {
  name           = "my-web-app"
  project_uuid   = coolify_project.example.uuid
  server_uuid    = "your-server-uuid"
  git_repository = "https://github.com/example/app"
  git_branch     = "main"
  build_pack     = "nixpacks"
  ports_exposes  = "3000"
  domains = "https://app.example.com"
}
```

See the [examples/](examples/) directory for per-resource examples (including
sensitive variable handling for
[`coolify_database_postgresql`](examples/resources/coolify_database_postgresql/resource.tf)
and [`coolify_github_app`](examples/resources/coolify_github_app/resource.tf)),
and the [ACME Corp scenarios](#what-you-can-do) above for full-stack examples.

## Authentication

The provider requires a Coolify API token. Coolify's API is disabled by default, so first enable it in the Coolify UI under **Settings**. Then generate a token under **Security > API Tokens**. Otherwise provider operations fail with `Unauthenticated`.

| Attribute | Environment Variable | Description |
|-----------|---------------------|-------------|
| `endpoint` | `COOLIFY_ENDPOINT` | Coolify API base URL |
| `token` | `COOLIFY_TOKEN` | API bearer token |

## Development

Install the local prerequisites and CI-pinned tools before running the commands below:

- Python 3.9+
- `golangci-lint` v2.12.2 exactly
- `goreleaser` v2.x
- `tfplugindocs`

See [CONTRIBUTING.md](CONTRIBUTING.md) for the exact install commands and
full local setup. Run `make help` to list the supported local targets from
[GNUmakefile](GNUmakefile).

```bash
make build       # Compile the provider
make test        # Run unit tests (690+ tests, race detector enabled)
make testacc     # Run acceptance tests (needs running Coolify instance)
make lint        # Run golangci-lint
make fmt         # Format code (gofmt + go mod tidy)
make docs        # Regenerate documentation via tfplugindocs
make validate    # Check HCL formatting in examples/
make python-test # Run Python unit tests for scripts/
make install     # Install provider to local Go bin
make ci          # Run the aggregate local checks (includes python-test; acceptance tests run separately)
```

`make ci` does not run acceptance tests. If your change touches real Coolify
API behavior, also run `make testacc` or targeted `TF_ACC=1 go test ...`
commands.

Cloud token and Hetzner-related acceptance tests need a real
`COOLIFY_HETZNER_TOKEN` in addition to the normal `COOLIFY_ENDPOINT` and
`COOLIFY_TOKEN` setup, because Coolify validates the token against Hetzner on
create.

GitHub App application acceptance additionally needs the optional
`COOLIFY_GITHUB_APP_*` fixture variables documented in [TESTING.md](TESTING.md),
because Coolify verifies repository access during application creation.

For local provider testing with `dev_overrides`, acceptance test setup, and
project structure details, see [CONTRIBUTING.md](CONTRIBUTING.md) and
[TESTING.md](TESTING.md).

### CI Pipeline

8 jobs in the CI workflow: Detect Changes, Test, Lint, Validate (includes
examples, docs, Trivy, Gitleaks), Scenario Tests, Acceptance Tests, Spec
Freshness (weekly), and a CI gate job.

## Troubleshooting

Enable provider debug logging to diagnose issues:

```bash
# Debug level: CRUD operations and state changes
TF_LOG_PROVIDER=DEBUG terraform plan

# Trace level: full HTTP request/response logging
TF_LOG_PROVIDER=TRACE terraform plan
```

Sensitive fields (passwords, tokens, private keys) are automatically
redacted in log output. See the
[Troubleshooting Guide](https://registry.terraform.io/providers/SebTardifLabs/coolify/latest/docs/guides/troubleshooting)
for details.

## License

MPL-2.0