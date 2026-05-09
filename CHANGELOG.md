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
  - `coolify_database_backup` - Schedule automated database backups with S3 storage and retention policies
  - `coolify_docker_compose_application` - Deploy applications from Docker Compose files
  - `coolify_clickhouse_database` - Provision ClickHouse databases
  - `coolify_s3_storage` - Manage S3 storage destinations for database backups
  - `coolify_keydb_database` - Provision KeyDB databases (high-performance Redis-compatible)
  - `coolify_dragonfly_database` - Provision DragonFly databases (modern Redis-compatible in-memory store)
- **Data Sources:**
  - `coolify_project` / `coolify_projects` - Read project(s)
  - `coolify_server` / `coolify_servers` - Read server(s)
  - `coolify_server_resources` - List all resources deployed on a server
  - `coolify_server_domains` - List all domains configured on a server
  - `coolify_private_key` / `coolify_private_keys` - Read SSH key(s)
  - `coolify_application` / `coolify_applications` - Read application(s)
  - `coolify_database` / `coolify_databases` - Read database(s)
  - `coolify_service` / `coolify_services` - Read service(s)
  - `coolify_team` - Read team info and members
  - `coolify_s3_storage` / `coolify_s3_storages` - Read S3 storage destination(s)
  - `coolify_version` - Read the Coolify instance version
- All resources support `terraform import`
- Retryable HTTP client with automatic retry on 429/5xx (3 retries, 30s timeout)
- Input validators: `build_pack` OneOf, port range (1-65535)
- Graceful handling of out-of-band resource deletion (404 in Read removes from state)
- 101 unit tests with 74% overall coverage
- CI pipeline: test, lint, format, docs generation
- GoReleaser config for GPG-signed releases
- Full-stack deployment example