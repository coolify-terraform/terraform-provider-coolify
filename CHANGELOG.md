# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.6](https://github.com/coolify-terraform/terraform-provider-coolify/compare/v0.1.5...v0.1.6) (2026-06-24)


### Features

* auto-update social preview image and repo description with live stats ([#531](https://github.com/coolify-terraform/terraform-provider-coolify/issues/531)) ([4fc0484](https://github.com/coolify-terraform/terraform-provider-coolify/commit/4fc0484e1172294ccfb30fb7114d5158f58a893a)), closes [#530](https://github.com/coolify-terraform/terraform-provider-coolify/issues/530)


### Bug Fixes

* address AI code quality findings ([#537](https://github.com/coolify-terraform/terraform-provider-coolify/issues/537)) ([7d8d5c1](https://github.com/coolify-terraform/terraform-provider-coolify/commit/7d8d5c1a83c8466153168e26a36fd98c3a2c9ba5))
* database health_check 422 on Coolify &lt; v4.1.2, bump min version to 4.1.0 ([#550](https://github.com/coolify-terraform/terraform-provider-coolify/issues/550)) ([01c1e57](https://github.com/coolify-terraform/terraform-provider-coolify/commit/01c1e5726dad39fba1b5d7e96628bd5317c24046))
* document RELEASE_NOTES.md must be on main, not release branch ([#527](https://github.com/coolify-terraform/terraform-provider-coolify/issues/527)) ([3e39513](https://github.com/coolify-terraform/terraform-provider-coolify/commit/3e395131bdacaa650009f0fa356fe524d1dca759)), closes [#526](https://github.com/coolify-terraform/terraform-provider-coolify/issues/526)
* release notes cleanup respects branch protection ([#529](https://github.com/coolify-terraform/terraform-provider-coolify/issues/529)) ([55b897d](https://github.com/coolify-terraform/terraform-provider-coolify/commit/55b897d97bbb564597065e67efa05334dbc33495)), closes [#526](https://github.com/coolify-terraform/terraform-provider-coolify/issues/526)
* trigger social preview update on release, add upload script ([#533](https://github.com/coolify-terraform/terraform-provider-coolify/issues/533)) ([8c2ea20](https://github.com/coolify-terraform/terraform-provider-coolify/commit/8c2ea2015df35504ebd8d601725041059c104d84)), closes [#530](https://github.com/coolify-terraform/terraform-provider-coolify/issues/530)

## [0.1.5](https://github.com/coolify-terraform/terraform-provider-coolify/compare/v0.1.4...v0.1.5) (2026-06-13)


### Features

* support RELEASE_NOTES.md override for curated release descriptions ([#525](https://github.com/coolify-terraform/terraform-provider-coolify/issues/525)) ([2597aa8](https://github.com/coolify-terraform/terraform-provider-coolify/commit/2597aa8eacaa46896a19b915670702aaf12acb8e)), closes [#524](https://github.com/coolify-terraform/terraform-provider-coolify/issues/524)
* update contract to Coolify v4.1.2 ([#518](https://github.com/coolify-terraform/terraform-provider-coolify/issues/518)) ([092ab4e](https://github.com/coolify-terraform/terraform-provider-coolify/commit/092ab4e9f5a62d04f3f7d026cb15e694b331c304)), closes [#517](https://github.com/coolify-terraform/terraform-provider-coolify/issues/517)


### Bug Fixes

* multi-perspective improvement cycle 1 ([#515](https://github.com/coolify-terraform/terraform-provider-coolify/issues/515)) ([3aa4252](https://github.com/coolify-terraform/terraform-provider-coolify/commit/3aa42527c95e0b20ead1006eb4645f33dfba8af3))
* multi-perspective improvement cycle 2 ([#520](https://github.com/coolify-terraform/terraform-provider-coolify/issues/520)) ([4bfec53](https://github.com/coolify-terraform/terraform-provider-coolify/commit/4bfec53ad983101e74bf429c65103306bfc4ca22))

## [0.1.4](https://github.com/coolify-terraform/terraform-provider-coolify/compare/v0.1.3...v0.1.4) (2026-06-03)


### Bug Fixes

* handle PATCH decode error in mock, deduplicate test mux, simplify merge target ([#508](https://github.com/coolify-terraform/terraform-provider-coolify/issues/508)) ([93fef84](https://github.com/coolify-terraform/terraform-provider-coolify/commit/93fef84f04397d3bab628e5a72f3eae79a6cdc93))
* increase polling timeout test context + bump Go 1.26.4 ([#503](https://github.com/coolify-terraform/terraform-provider-coolify/issues/503)) ([6223542](https://github.com/coolify-terraform/terraform-provider-coolify/commit/6223542987da2d73453cfb582a2c45eced18cf91))

## [0.1.3](https://github.com/coolify-terraform/terraform-provider-coolify/compare/v0.1.2...v0.1.3) (2026-06-02)


### Bug Fixes

* add coolify-v4-latest.json to .gitignore ([09bc849](https://github.com/coolify-terraform/terraform-provider-coolify/commit/09bc849d56c1c573453be4a1f223ae2b39813416))
* improve test honesty, CI safety, and error handling ([#485](https://github.com/coolify-terraform/terraform-provider-coolify/issues/485)) ([ab40882](https://github.com/coolify-terraform/terraform-provider-coolify/commit/ab408820e39d6612e7df1073b4b8c51cd1f08745))

## [0.1.2](https://github.com/coolify-terraform/terraform-provider-coolify/compare/v0.1.1...v0.1.2) (2026-06-01)


### Bug Fixes

* **ci:** exclude release-please compare URLs from lychee link check ([715124c](https://github.com/coolify-terraform/terraform-provider-coolify/commit/715124cdfac4fd1048ef93cd7232906e26449dca))

## [0.1.1](https://github.com/coolify-terraform/terraform-provider-coolify/compare/v0.1.0...v0.1.1) (2026-06-01)


### Features

* add auto-approve workflow for solo maintainer PRs ([de9e28f](https://github.com/coolify-terraform/terraform-provider-coolify/commit/de9e28ff1f3c3f734bba6b2415c9cd2dfe63a6cd)), closes [#457](https://github.com/coolify-terraform/terraform-provider-coolify/issues/457)
* adopt release-please for automated releases ([#447](https://github.com/coolify-terraform/terraform-provider-coolify/issues/447)) ([56e1add](https://github.com/coolify-terraform/terraform-provider-coolify/commit/56e1add93def5d03e9653ba222ac2f5ec310b827))


### Bug Fixes

* add deleted_at to internal fields exclusion test ([#442](https://github.com/coolify-terraform/terraform-provider-coolify/issues/442)) ([8b89d2a](https://github.com/coolify-terraform/terraform-provider-coolify/commit/8b89d2ab4bf60a77f99e41ba239bc6a6b3d3f22e))
* add make merge target and FOSSA false-positive filter ([#435](https://github.com/coolify-terraform/terraform-provider-coolify/issues/435)) ([5323f97](https://github.com/coolify-terraform/terraform-provider-coolify/commit/5323f97e6f6bba50d0ed75649856ff19ee1d925e))
* **ci:** use original filename for FOSSA CLI sha256 verification ([#440](https://github.com/coolify-terraform/terraform-provider-coolify/issues/440)) ([f2fe25f](https://github.com/coolify-terraform/terraform-provider-coolify/commit/f2fe25f096c145477a88490bcb9a8bb780304065))
* remove unused Python imports and variables ([#441](https://github.com/coolify-terraform/terraform-provider-coolify/issues/441)) ([24c928d](https://github.com/coolify-terraform/terraform-provider-coolify/commit/24c928df2db127fcbedc9b8a6b2c546852f0db6e))
* update CI job count to 9, add DCO, validate in counts-check ([#461](https://github.com/coolify-terraform/terraform-provider-coolify/issues/461)) ([d5d78c0](https://github.com/coolify-terraform/terraform-provider-coolify/commit/d5d78c015e7571d07f65aceaec240c5206bd57b2)), closes [#460](https://github.com/coolify-terraform/terraform-provider-coolify/issues/460)
* update contract with new POST /sentinel/push route ([#472](https://github.com/coolify-terraform/terraform-provider-coolify/issues/472)) ([c446a7f](https://github.com/coolify-terraform/terraform-provider-coolify/commit/c446a7fba19f593fb082b44208ff74e2ce1f4445)), closes [#471](https://github.com/coolify-terraform/terraform-provider-coolify/issues/471)
* update dependencies and pin FOSSA CLI for Scorecard ([#438](https://github.com/coolify-terraform/terraform-provider-coolify/issues/438)) ([626f0bf](https://github.com/coolify-terraform/terraform-provider-coolify/commit/626f0bf2bb343af2d725e73a703af2d84374321a))
* update stale CHANGELOG URL to current org ([e5e698a](https://github.com/coolify-terraform/terraform-provider-coolify/commit/e5e698a36e8f8f16f90234d565ec0f169adc9170))
* upgrade golang.org/x/crypto in tools module to v0.52.0 ([#466](https://github.com/coolify-terraform/terraform-provider-coolify/issues/466)) ([87c90cf](https://github.com/coolify-terraform/terraform-provider-coolify/commit/87c90cf8cf413aace9a202484dab4806388430ab))
* use stable PR author check in auto-approve workflow ([7f7599b](https://github.com/coolify-terraform/terraform-provider-coolify/commit/7f7599b5949238051ba1979fe4b2fc365166bef0))
* use workflow badge for FOSSA instead of API badge ([#437](https://github.com/coolify-terraform/terraform-provider-coolify/issues/437)) ([ca8c4c1](https://github.com/coolify-terraform/terraform-provider-coolify/commit/ca8c4c12da2863925db2fe6e3713f1394910ef76))

## [0.1.0](https://github.com/coolify-terraform/terraform-provider-coolify/releases/tag/v0.1.0) (2026-05-30)

### Breaking Changes

- `coolify_github_app`: The `private_key` attribute has been renamed to `private_key_uuid` to match the Coolify API spec. This field now accepts a UUID referencing an existing `coolify_private_key` resource instead of raw key content.
- `coolify_database_backup`: The `retain_days` attribute has been renamed to `retain_amount_locally`. The old name was misleading (it stored a count of backup copies, not days). Users must update their `.tf` files to use the new name.
- `coolify_s3_storage` resource, `coolify_s3_storage` data source, and `coolify_s3_storages` data source have been removed. Current Coolify v4 has no public top-level S3 storage API. Manage S3 storages in the Coolify web UI and reference their UUIDs from `coolify_database_backup.s3_storage_uuid`. Before upgrading, remove these from state: `terraform state rm coolify_s3_storage.<name>`.

### Added

- UUID format validation on 13 attributes across server, Hetzner, backup, scheduled task, and GitHub App resources/data sources (catches malformed input at plan time instead of API time)
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
  - `coolify_application_dockerfile` - Deploy applications from Dockerfiles
  - `coolify_application_docker_image` - Deploy applications from Docker images (Docker Hub, GHCR, etc.)
  - `coolify_application_private_git` - Deploy applications from private Git repositories (SSH deploy key)
  - `coolify_application_github_app` - Deploy applications via GitHub App integration
  - `coolify_environment` - Manage project environments
  - `coolify_environment_variable` - Manage env vars for applications, services, and databases
  - `coolify_deployment` - Trigger application deployments (with `triggers` map for force-redeploy)
  - `coolify_service` - Deploy one-click services from the Coolify catalog
  - `coolify_database_postgresql` - Provision PostgreSQL databases
  - `coolify_database_mysql` - Provision MySQL databases
  - `coolify_database_mariadb` - Provision MariaDB databases
  - `coolify_database_redis` - Provision Redis databases
  - `coolify_database_mongodb` - Provision MongoDB databases
  - `coolify_database_clickhouse` - Provision ClickHouse databases
  - `coolify_database_keydb` - Provision KeyDB databases (Redis-compatible)
  - `coolify_database_dragonfly` - Provision DragonFly databases (Redis-compatible in-memory store)
  - `coolify_database_backup` - Schedule automated database backups with S3 storage and retention
  - `coolify_scheduled_task` - Manage scheduled tasks on applications/services
  - `coolify_storage` - Manage persistent storage volumes
  - `coolify_cloud_token` - Manage cloud provider tokens (Hetzner)
  - `coolify_github_app` - Manage GitHub App integrations
  - `coolify_server_hetzner` - Provision Hetzner Cloud servers via Coolify
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
- All stateful resources support `terraform import` (action/validation resources are lifecycle-only)
- 99%+ Coolify v4 API coverage (134/135 endpoints)
- OpenAPI spec-driven test validation with libopenapi-validator
- API coverage tracking with auto-generated `API_COVERAGE.md`
- UUID format validators on all UUID input fields
- Retryable HTTP client with automatic retry on 429/5xx (3 retries, 30s timeout)
- Input validators: `build_pack` OneOf, FQDN format, cron syntax, port range (1-65535), UUID format, environment variable name format
- Configurable `timeouts` block on all application resources
- Graceful handling of out-of-band resource deletion (404 in Read removes from state)
- 750+ unit tests with race detection across 40 packages
- CI pipeline: 8 jobs (detect changes, test, lint, validate, scenario tests, acceptance tests, spec freshness, CI gate)
- GoReleaser config for GPG-signed releases
- Computed `status` field on all application resources
- Full-stack deployment example

### Changed

- `redeploy_on_update` now triggers a restart for all configuration fields including `name`, `description`, webhook secrets, auto-deploy settings, and container label options. Previously only runtime-affecting fields (ports, limits, health checks, build settings) were covered. Only immutable, computed-only, and the `redeploy_on_update` flag itself are excluded.
- `dockerfile` and `docker_compose_raw` attributes are now marked `Sensitive` (they can contain embedded secrets such as build arguments or service credentials)
- Redundant `UseStateForUnknown` plan modifier removed from `deployment_queue_limit` on server resources (the `Default` value already handles this; no user-visible behavior change)
- Consolidated `is_include_timestamps`, `enable_ssl`, and `ssl_mode` handling into shared database helpers, reducing duplication across all 8 database resources
- Minimum Terraform version requirement updated to >= 1.6 (consistent across all documentation)
- Added TRACE-level logging to version and health check endpoints for easier connection debugging
- `coolify_github_app`: `app_id`, `installation_id`, `client_id`, `client_secret`, `private_key_uuid`, and `organization_name` can now be updated in-place (previously forced destroy/recreate). This matches the Coolify API's PATCH support for these fields.
- `coolify_application_github_app`: `github_app_uuid` can now be updated in-place (previously forced destroy/recreate).

### Fixed

- API response bodies are now redacted in TRACE logs, preventing sensitive fields (passwords, keys) from appearing in debug output
- Custom TLS configuration (`ca_cert`, `insecure`) no longer silently disables HTTP retry logic
- `redactJSON` now handles JSON arrays and nested objects (previously only top-level objects were redacted)

- `coolify_service` resource: changing `name`, `description`, or `environment_name` now triggers destroy/recreate (previously produced an "Update not supported" error during apply)
- `coolify_database_clickhouse`: `clickhouse_admin_user` and `clickhouse_admin_password` are now sent during resource creation (previously silently ignored, only applied on update)
- All 8 database resources: removing `description` from config no longer leaves stale values in state (now correctly sets null when API returns empty)
- All 8 database resources: `environment_name` now has `RequiresReplace` (changing it forces a new resource, matching the API's actual behavior)
- `coolify_storage` resource: `UpdateStorageInput` now includes `UUID` field so PATCH correctly identifies the target storage
- `coolify_deployment` resource: `GetDeployment` errors during Create now produce a warning diagnostic instead of silently defaulting to "queued" status
- `coolify_private_key` resource: empty description from API now correctly becomes `null` in state (consistent with all other resources)
- `PollUntilDeleted` (used by application and service Delete) now respects the parent context's deadline instead of always using a hardcoded 2-minute timeout. Resources with a `timeouts` block now have their configured timeout honored during delete polling.
