# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Breaking Changes

- `coolify_github_app`: The `private_key` attribute has been renamed to `private_key_uuid` to match the Coolify API spec. This field now accepts a UUID referencing an existing `coolify_private_key` resource instead of raw key content.
- `coolify_database_backup`: The `retain_days` attribute has been renamed to `retain_amount_locally`. The old name was misleading (it stored a count of backup copies, not days). Users must update their `.tf` files to use the new name.
- `coolify_s3_storage` resource, `coolify_s3_storage` data source, and `coolify_s3_storages` data source have been removed. Current Coolify v4 has no public top-level S3 storage API. Manage S3 storages in the Coolify web UI and reference their UUIDs from `coolify_database_backup.s3_storage_uuid`.

### Fixed

- API response bodies are now redacted in TRACE logs, preventing sensitive fields (passwords, keys) from appearing in debug output
- Custom TLS configuration (`ca_cert`, `insecure`) no longer silently disables HTTP retry logic
- `redactJSON` now handles JSON arrays and nested objects (previously only top-level objects were redacted)

### Changed

- Consolidated `is_include_timestamps`, `enable_ssl`, and `ssl_mode` handling into shared database helpers, reducing duplication across all 8 database resources
- Minimum Terraform version requirement updated to >= 1.6 (consistent across all documentation)
- Added TRACE-level logging to version and health check endpoints for easier connection debugging

### Added

- `coolify_deployment`: `wait_for_completion` attribute polls deployment status until `finished` or `error`; `timeouts` block for configurable Create timeout
- `coolify_database_backup`: 12 new fields for S3 toggle, selective backup, retention policies, and job timeout
- All application resources: 16 new fields for resource limits, health checks, and auto-deploy control
- All database and service resources: `timeouts` block with configurable Create timeout (default 10 minutes)
- 4 new singular data sources: `coolify_deployment`, `coolify_environment_variable`, `coolify_scheduled_task`, `coolify_storage`
- `tflog.Debug` structured logging in all resource CRUD methods
- **Provider configuration** with `endpoint` and `token` attributes (env var fallback: `COOLIFY_ENDPOINT`, `COOLIFY_TOKEN`)
- Health check during `Configure` validates API connection by calling `/api/v1/version`
- **Resources:**
  - `coolify_project` - Manage projects
  - `coolify_server` - Register and configure servers
  - `coolify_private_key` - Manage SSH keys
  - `coolify_application` - Deploy applications from public Git repositories
  - `coolify_dockerfile_application` - Deploy applications from Dockerfiles
  - `coolify_docker_image_application` - Deploy applications from Docker images (Docker Hub, GHCR, etc.)
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
  - `coolify_scheduled_task` - Manage scheduled tasks on applications/services
  - `coolify_storage` - Manage persistent storage volumes
  - `coolify_cloud_token` - Manage cloud provider tokens (Hetzner)
  - `coolify_github_app` - Manage GitHub App integrations
- **Data Sources:**
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
  - `coolify_environment_variable` / `coolify_environment_variables` - Read / list environment variables for an application, service, or database
  - `coolify_deployment` / `coolify_deployments` - Read / list deployments for an application
  - `coolify_scheduled_task` / `coolify_scheduled_tasks` / `coolify_task_executions` - Read scheduled task(s) and executions
  - `coolify_storage` / `coolify_storages` - Read / list persistent storage volumes
  - `coolify_cloud_token` / `coolify_cloud_tokens` - Read cloud token(s)
  - `coolify_github_app` / `coolify_github_apps` / `coolify_github_app_repositories` / `coolify_github_app_branches` - Read GitHub App(s) and repos
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
- 420+ unit tests with race detection across 34 packages
- CI pipeline: 8 jobs (detect changes, test, lint, validate, scenario tests, acceptance tests, spec freshness, CI gate)
- GoReleaser config for GPG-signed releases
- Computed `status` field on all application resources
- Full-stack deployment example

### Changed

- `coolify_github_app`: `app_id`, `installation_id`, `client_id`, `client_secret`, `private_key_uuid`, and `organization_name` can now be updated in-place (previously forced destroy/recreate). This matches the Coolify API's PATCH support for these fields.
- `coolify_github_app_application`: `github_app_uuid` can now be updated in-place (previously forced destroy/recreate).

### Fixed

- `coolify_service` resource: changing `name`, `description`, or `environment_name` now triggers destroy/recreate (previously produced an "Update not supported" error during apply)
- `coolify_clickhouse_database`: `clickhouse_admin_user` and `clickhouse_admin_password` are now sent during resource creation (previously silently ignored, only applied on update)
- All 8 database resources: removing `description` from config no longer leaves stale values in state (now correctly sets null when API returns empty)
- All 8 database resources: `environment_name` now has `RequiresReplace` (changing it forces a new resource, matching the API's actual behavior)
- `coolify_storage` resource: `UpdateStorageInput` now includes `UUID` field so PATCH correctly identifies the target storage
- `coolify_deployment` resource: `GetDeployment` errors during Create now produce a warning diagnostic instead of silently defaulting to "queued" status
- `coolify_private_key` resource: empty description from API now correctly becomes `null` in state (consistent with all other resources)