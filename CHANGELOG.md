# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **Provider configuration** with `endpoint` and `token` attributes (env var fallback: `COOLIFY_ENDPOINT`, `COOLIFY_TOKEN`)
- **Resources:**
  - `coolify_project` - Manage projects
  - `coolify_server` - Register and configure servers
  - `coolify_private_key` - Manage SSH keys
  - `coolify_application` - Deploy applications from public Git repositories
  - `coolify_environment_variable` - Manage env vars for applications and services
  - `coolify_postgresql_database` - Provision PostgreSQL databases
  - `coolify_mysql_database` - Provision MySQL databases
  - `coolify_mariadb_database` - Provision MariaDB databases
  - `coolify_redis_database` - Provision Redis databases
  - `coolify_mongodb_database` - Provision MongoDB databases
  - `coolify_service` - Deploy one-click services from the Coolify catalog
  - `coolify_deployment` - Trigger application deployments (with `triggers` map for force-redeploy)
  - `coolify_docker_image_application` - Deploy applications from Docker images (Docker Hub, GHCR, etc.)
  - `coolify_private_git_application` - Deploy applications from private Git repositories (with SSH deploy key)
- **Data Sources:**
  - `coolify_project` / `coolify_projects` - Read project(s)
  - `coolify_server` / `coolify_servers` - Read server(s)
  - `coolify_private_key` / `coolify_private_keys` - Read SSH key(s)
  - `coolify_application` / `coolify_applications` - Read application(s)
  - `coolify_databases` - List all databases
  - `coolify_services` - List all services
  - `coolify_team` - Read team info and members
- All resources support `terraform import`
- Retryable HTTP client with automatic retry on 429/5xx (3 retries, 30s timeout)
- Input validators: `build_pack` OneOf, port range (1-65535)
- Graceful handling of out-of-band resource deletion (404 in Read removes from state)
- 101 unit tests with 74% overall coverage
- CI pipeline: test, lint, format, docs generation
- GoReleaser config for GPG-signed releases
- Full-stack deployment example