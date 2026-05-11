# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **Provider configuration** with `endpoint` and `token` attributes (env var fallback: `COOLIFY_ENDPOINT`, `COOLIFY_TOKEN`)
- Health check during `Configure` validates API connection by calling `/api/v1/version`
- **27 Resources:**
  - `coolify_project` - Manage projects
  - `coolify_server` - Register and configure servers
  - `coolify_private_key` - Manage SSH keys
  - `coolify_application` - Deploy applications from public Git repositories
  - `coolify_dockerfile_application` - Deploy applications from Dockerfiles
  - `coolify_docker_image_application` - Deploy applications from Docker images (Docker Hub, GHCR, etc.)
  - `coolify_docker_compose_application` - Deploy applications from Docker Compose files
  - `coolify_private_git_application` - Deploy applications from private Git repositories (SSH deploy key)
  - `coolify_github_app_application` - Deploy applications via GitHub App integration
  - `coolify_environment` - Manage project environments
  - `coolify_environment_variable` - Manage env vars for applications, services, and databases
  - `coolify_deployment` - Trigger application deployments (with `triggers` map for force-redeploy)
  - `coolify_service` - Deploy one-click services from the Coolify catalog
  - `coolify_postgresql_database` - Provision PostgreSQL databases
  - `coolify_mysql_database` - Provision MySQL databases
  - `coolify_mariadb_database` - Provision MariaDB databases
  - `coolify_redis_database` - Provision Redis databases
  - `coolify_mongodb_database` - Provision MongoDB databases
  - `coolify_clickhouse_database` - Provision ClickHouse databases
  - `coolify_keydb_database` - Provision KeyDB databases (Redis-compatible)
  - `coolify_dragonfly_database` - Provision DragonFly databases (Redis-compatible in-memory store)
  - `coolify_database_backup` - Schedule automated database backups with S3 storage and retention
  - `coolify_s3_storage` - Manage S3 storage destinations for database backups
  - `coolify_scheduled_task` - Manage scheduled tasks on applications/services
  - `coolify_storage` - Manage persistent storage volumes
  - `coolify_cloud_token` - Manage cloud provider tokens (Hetzner)
  - `coolify_github_app` - Manage GitHub App integrations
- **41 Data Sources:**
  - `coolify_project` / `coolify_projects` - Read project(s)
  - `coolify_server` / `coolify_servers` - Read server(s)
  - `coolify_server_resources` - List all resources deployed on a server
  - `coolify_server_domains` - List all domains configured on a server
  - `coolify_server_validation` - Validate a server's connectivity
  - `coolify_private_key` / `coolify_private_keys` - Read SSH key(s)
  - `coolify_application` / `coolify_applications` - Read application(s)
  - `coolify_application_logs` - Read application logs
  - `coolify_database` / `coolify_databases` - Read database(s)
  - `coolify_service` / `coolify_services` - Read service(s)
  - `coolify_environment` / `coolify_environments` - Read environment(s)
  - `coolify_environment_variables` - List environment variables for an application, service, or database
  - `coolify_deployments` - List deployments for an application
  - `coolify_s3_storage` / `coolify_s3_storages` - Read S3 storage destination(s)
  - `coolify_scheduled_tasks` / `coolify_task_executions` - Read scheduled tasks and executions
  - `coolify_storages` - List persistent storage volumes
  - `coolify_cloud_token` / `coolify_cloud_tokens` - Read cloud token(s)
  - `coolify_github_apps` / `coolify_github_app_repositories` / `coolify_github_app_branches` - Read GitHub App(s) and repos
  - `coolify_backup_executions` - List backup execution history
  - `coolify_resources` - List all resources on a server
  - `coolify_team` / `coolify_teams` / `coolify_team_members` - Read team(s) and members
  - `coolify_health` - Read Coolify instance health status
  - `coolify_version` - Read the Coolify instance version
  - `coolify_hetzner_images` / `coolify_hetzner_locations` / `coolify_hetzner_server_types` / `coolify_hetzner_ssh_keys` - Read Hetzner cloud resources
- All resources support `terraform import`
- 100% Coolify v4 API coverage (135/135 endpoints)
- OpenAPI spec-driven test validation with libopenapi-validator
- API coverage tracking with auto-generated `API_COVERAGE.md`
- UUID format validators on all UUID input fields
- Retryable HTTP client with automatic retry on 429/5xx (3 retries, 30s timeout)
- Input validators: `build_pack` OneOf, FQDN format, cron syntax, port range (1-65535), UUID format, environment variable name format
- Configurable `timeouts` block on all application resources
- Graceful handling of out-of-band resource deletion (404 in Read removes from state)
- 450+ unit tests with race detection across 34 packages
- CI pipeline: 9 jobs (detect changes, test, lint, validate examples, docs, govulncheck, trivy, gitleaks, spec freshness)
- GoReleaser config for GPG-signed releases
- Computed `status` field on all application resources
- Full-stack deployment example

### Fixed

- `coolify_service` resource: changing `name`, `description`, or `environment_name` now triggers destroy/recreate (previously produced an "Update not supported" error during apply)
- `coolify_clickhouse_database`: `clickhouse_admin_user` and `clickhouse_admin_password` are now sent during resource creation (previously silently ignored, only applied on update)
- All 8 database resources: removing `description` from config no longer leaves stale values in state (now correctly sets null when API returns empty)
- All 8 database resources: `environment_name` now has `RequiresReplace` (changing it forces a new resource, matching the API's actual behavior)
- `coolify_storage` resource: `UpdateStorageInput` now includes `UUID` field so PATCH correctly identifies the target storage
- `coolify_deployment` resource: `GetDeployment` errors during Create now produce a warning diagnostic instead of silently defaulting to "queued" status
- `coolify_private_key` resource: empty description from API now correctly becomes `null` in state (consistent with all other resources)