# Terraform Provider for Coolify

[![CI](https://github.com/SebTardif/terraform-provider-coolify/actions/workflows/ci.yml/badge.svg)](https://github.com/SebTardif/terraform-provider-coolify/actions/workflows/ci.yml)

A Terraform provider for managing resources in [Coolify](https://coolify.io/), the open-source, self-hosted Platform-as-a-Service.

## Features

- **Projects**: Create and manage Coolify projects
- **Servers**: Register and configure servers
- **Private Keys**: Manage SSH keys for server access
- **Applications**: Deploy applications from public Git repositories
- **Environment Variables**: Manage env vars for applications and services
- **Databases**: Provision PostgreSQL, MySQL, MariaDB, Redis, and MongoDB databases
- **Services**: Deploy one-click services from the Coolify catalog
- **Teams**: Read team information and membership

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