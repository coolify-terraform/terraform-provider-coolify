# Terraform Provider for Coolify

[![CI](https://github.com/SebTardif/terraform-provider-coolify/actions/workflows/ci.yml/badge.svg)](https://github.com/SebTardif/terraform-provider-coolify/actions/workflows/ci.yml)

A Terraform provider for managing resources in [Coolify](https://coolify.io/), the open-source, self-hosted Platform-as-a-Service.

## Resources

| Resource | Description |
|----------|-------------|
| `coolify_project` | Manage projects (logical grouping for resources) |
| `coolify_server` | Register and configure servers |
| `coolify_private_key` | Manage SSH keys for server access |
| `coolify_application` | Deploy apps from public Git repositories |
| `coolify_private_git_application` | Deploy apps from private Git repos (with SSH deploy key) |
| `coolify_docker_image_application` | Deploy apps from Docker images (Docker Hub, GHCR, etc.) |
| `coolify_environment_variable` | Manage env vars for applications and services |
| `coolify_postgresql_database` | Provision PostgreSQL databases |
| `coolify_mysql_database` | Provision MySQL databases |
| `coolify_mariadb_database` | Provision MariaDB databases |
| `coolify_redis_database` | Provision Redis instances |
| `coolify_mongodb_database` | Provision MongoDB databases |
| `coolify_docker_compose_application` | Deploy apps from Docker Compose files |
| `coolify_service` | Deploy one-click services from the Coolify catalog |
| `coolify_deployment` | Trigger application deployments |
| `coolify_database_backup` | Schedule automated database backups |
| `coolify_clickhouse_database` | Provision ClickHouse databases |
| `coolify_keydb_database` | Provision KeyDB databases (Redis-compatible) |
| `coolify_dragonfly_database` | Provision DragonFly databases (Redis-compatible) |

## Data Sources

| Data Source | Description |
|-------------|-------------|
| `coolify_project` / `coolify_projects` | Read project(s) |
| `coolify_server` / `coolify_servers` | Read server(s) |
| `coolify_server_resources` | List resources deployed on a server |
| `coolify_server_domains` | List domains configured on a server |
| `coolify_private_key` / `coolify_private_keys` | Read SSH key(s) |
| `coolify_application` / `coolify_applications` | Read application(s) |
| `coolify_database` / `coolify_databases` | Read database(s) |
| `coolify_service` / `coolify_services` | Read service(s) |
| `coolify_team` | Read team info and members |

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.24 (for building from source)
- A running [Coolify](https://coolify.io/) instance (v4.x)

## Usage

```hcl
terraform {
  required_providers {
    coolify = {
      source = "SebTardif/coolify"
    }
  }
}

provider "coolify" {
  endpoint = "http://your-coolify-instance:8000"
  token    = "your-api-token"
}

resource "coolify_project" "example" {
  name        = "my-project"
  description = "Managed by Terraform"
}

resource "coolify_postgresql_database" "db" {
  name             = "my-database"
  project_uuid     = coolify_project.example.uuid
  server_uuid      = "your-server-uuid"
  image            = "postgres:16"
  postgres_user    = "app"
  postgres_db      = "myapp"
}

resource "coolify_application" "web" {
  name             = "my-web-app"
  project_uuid     = coolify_project.example.uuid
  server_uuid      = "your-server-uuid"
  git_repository   = "https://github.com/example/app"
  git_branch       = "main"
  build_pack       = "nixpacks"
  ports_exposes    = "3000"
  fqdn             = "https://app.example.com"
}
```

## Authentication

The provider requires a Coolify API token. Generate one in the Coolify UI under **Keys & Tokens > API tokens**.

| Attribute | Environment Variable | Description |
|-----------|---------------------|-------------|
| `endpoint` | `COOLIFY_ENDPOINT` | Coolify API base URL |
| `token` | `COOLIFY_TOKEN` | API bearer token |

## Development

```bash
# Build
make build

# Run unit tests
make test

# Run acceptance tests (requires a running Coolify instance)
export COOLIFY_ENDPOINT="http://localhost:8000"
export COOLIFY_TOKEN="your-token"
make testacc

# Lint
make lint

# Generate docs
make docs
```

## License

MPL-2.0