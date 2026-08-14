# API Coverage

<!-- Auto-generated from internal/spectest/coverage_test.go. Do not edit manually. -->
<!-- Run: make api-coverage -->

This page answers: **which Coolify HTTP routes does the provider wrap, and what do I use when it does not?**

It is a **route inventory** against Coolify source (`testdata/contracts/coolify-v4.json`). It is not a field catalog, and it does not say which Coolify **version** you need.

- **Which Coolify version for each resource and field:** [Coolify Version Support](docs/guides/coolify-version-support.md)
- **Resource and attribute docs:** [docs/](docs/) (also on the Terraform Registry)
- **Field-level gaps** (numeric FKs, UI-only columns on an existing GET) live in `internal/spectest/contract_skips.go`, not in this route list.

**Coverage**: 244 covered / 296 registry entries (82.4%)  
**Planned**: 0 | **Skipped**: 52  
**Registry size**: 296 (contract routes + allowlisted extras)

## What Terraform does not wrap

Terraform models desired durable state. Coolify also exposes one-shot buttons, log streams, URL aliases, and control-plane actions. Those routes are skipped on purpose.

### Clone, migrate, and move

Coolify can copy or relocate an existing app, database, service, or server in one API call. Terraform then either owns two objects or loses the original.

**Use this instead:** `coolify_application` (and variants), `coolify_database_*`, `coolify_service`, or `coolify_server` to create the destination. For a one-time move, use the Coolify UI.

| Route |
|-------|
| `POST /applications/{uuid}/clone` |
| `POST /applications/{uuid}/migrate` |
| `POST /applications/{uuid}/move` |
| `POST /databases/{uuid}/clone` |
| `POST /databases/{uuid}/migrate` |
| `POST /databases/{uuid}/move` |
| `POST /servers/{uuid}/migrate` |
| `POST /services/{uuid}/clone` |
| `POST /services/{uuid}/migrate` |
| `POST /services/{uuid}/move` |

### Run now (backup or scheduled task)

These POSTs fire a single backup or task execution. They are not a schedule you can keep in state.

**Use this instead:** `coolify_storage_backup` or `coolify_database_backup` for the schedule. `coolify_scheduled_task` for the task definition. Trigger a run from the Coolify UI when you need one now.

| Route |
|-------|
| `POST /applications/{uuid}/scheduled-tasks/{task_uuid}/execute` |
| `POST /applications/{uuid}/storages/{storage_uuid}/backups/run` |
| `POST /databases/{uuid}/storages/{storage_uuid}/backups/run` |
| `POST /services/{uuid}/scheduled-tasks/{task_uuid}/execute` |
| `POST /services/{uuid}/storages/{storage_uuid}/backups/run` |

### Rollback and prior images

Listing rollback images and posting a rollback is an operational recovery step, not desired state.

**Use this instead:** The Coolify UI. There is no `coolify_rollback` resource.

| Route |
|-------|
| `GET /applications/{uuid}/rollback-images` |
| `POST /applications/{uuid}/rollback` |

### Log streams

Log endpoints stream runtime output. That is not durable Terraform state.

**Use this instead:** `data.coolify_application_logs` for application logs. Database and service log streams stay in the Coolify UI.

| Route |
|-------|
| `GET /databases/{uuid}/logs` |
| `GET /services/{uuid}/applications/{app_uuid}/logs` |
| `GET /services/{uuid}/databases/{database_uuid}/logs` |
| `GET /services/{uuid}/logs` |
| `POST /services/{uuid}/applications/{app_uuid}/logs` |

### Nested compose service apps and databases

A catalog `coolify_service` owns the whole compose stack. Coolify also exposes each inner app and database as its own API object.

**Use this instead:** `coolify_service` for the stack. Do not manage inner `/services/{uuid}/applications/...` or `/services/{uuid}/databases/...` as separate resources.

| Route |
|-------|
| `GET /services/{uuid}/applications` |
| `GET /services/{uuid}/applications/{app_uuid}` |
| `GET /services/{uuid}/applications/{app_uuid}/restart` |
| `GET /services/{uuid}/applications/{app_uuid}/start` |
| `GET /services/{uuid}/applications/{app_uuid}/stop` |
| `GET /services/{uuid}/databases` |
| `GET /services/{uuid}/databases/{database_uuid}` |
| `PATCH /services/{uuid}/applications/{app_uuid}` |
| `PATCH /services/{uuid}/databases/{database_uuid}` |
| `POST /services/{uuid}/applications/{app_uuid}/restart` |
| `POST /services/{uuid}/applications/{app_uuid}/start` |
| `POST /services/{uuid}/applications/{app_uuid}/stop` |
| `POST /services/{uuid}/databases/{database_uuid}/restart` |
| `POST /services/{uuid}/databases/{database_uuid}/start` |
| `POST /services/{uuid}/databases/{database_uuid}/stop` |

### Server control-plane actions

Export, import, claim, transfer, Sentinel push, proxy restart, and one-shot docker cleanup are operator actions, not server settings.

**Use this instead:** `coolify_server_proxy`, `coolify_server_sentinel`, `coolify_server_docker_cleanup`, and `coolify_server_cloudflare_tunnel` for settings. Use the Coolify UI for export, claim, transfer, and run-now cleanup.

| Route |
|-------|
| `GET /servers/{uuid}/docker-cleanup/executions` |
| `GET /servers/{uuid}/export` |
| `POST /sentinel/push` |
| `POST /servers/import` |
| `POST /servers/{uuid}/claim` |
| `POST /servers/{uuid}/docker-cleanup/run` |
| `POST /servers/{uuid}/export/mailbox` |
| `POST /servers/{uuid}/proxy/restart` |
| `POST /servers/{uuid}/transfer/complete` |

### Cloudflare tunnel enable and disable POSTs

Coolify also has POST enable/disable routes. The provider writes tunnel settings with PATCH.

**Use this instead:** `coolify_server_cloudflare_tunnel`.

| Route |
|-------|
| `POST /servers/{uuid}/cloudflare-tunnel/disable` |
| `POST /servers/{uuid}/cloudflare-tunnel/enable` |

### Team URL aliases

`GET /team` and `GET /team/members` are aliases of `/teams/current` and `/teams/current/members`.

**Use this instead:** `data.coolify_team` and `data.coolify_team_members`.

| Route |
|-------|
| `GET /team` |
| `GET /team/members` |

### Deprecated docker-compose create

`POST /applications/dockercompose` creates a Service, not an Application. Coolify kept the path as a deprecated alias.

**Use this instead:** `coolify_service` (`POST /services`).

| Route |
|-------|
| `POST /applications/dockercompose` |

### Product feedback

`POST /feedback` is a Coolify product endpoint, not infrastructure.

**Use this instead:** Nothing in Terraform. Send feedback through Coolify.

| Route |
|-------|
| `POST /feedback` |

## Routes by Terraform resource

A row here means the provider calls that Coolify route. `client.*` helpers are used from resources or are not a standalone resource.

### `client.BulkUpdateEnvVars`

| Route | Since |
|-------|-------|
| `PATCH /applications/{uuid}/envs/bulk` | v0.2.0 |
| `PATCH /databases/{uuid}/envs/bulk` | v0.2.0 |
| `PATCH /services/{uuid}/envs/bulk` | v0.2.0 |

### `client.CancelDeployment`

| Route | Since |
|-------|-------|
| `POST /deployments/{uuid}/cancel` | v0.2.0 |

### `client.DeleteBackupExecution`

| Route | Since |
|-------|-------|
| `DELETE /databases/{uuid}/backups/{scheduled_backup_uuid}/executions/{execution_uuid}` | v0.2.0 |

### `client.DeletePreviewDeployment`

| Route | Since |
|-------|-------|
| `DELETE /applications/{uuid}/previews/{pull_request_id}` | v0.2.0 |

### `client.Deploy`

| Route | Since |
|-------|-------|
| `GET /deploy` | v0.2.0 |
| `POST /deploy` | v0.2.0 |

### `client.DisableAPI`

| Route | Since |
|-------|-------|
| `GET /disable` | v0.2.0 |

### `client.EnableAPI`

| Route | Since |
|-------|-------|
| `GET /enable` | v0.2.0 |

### `client.ValidateCloudToken`

| Route | Since |
|-------|-------|
| `POST /cloud-tokens/{uuid}/validate` | v0.2.0 |

### `coolify_*_database`

| Route | Since |
|-------|-------|
| `DELETE /databases/{uuid}` | v0.1.0 |
| `PATCH /databases/{uuid}` | v0.1.0 |

### `coolify_api_settings`

| Route | Since |
|-------|-------|
| `POST /disable` | v0.2.0 |
| `POST /enable` | v0.2.0 |

### `coolify_api_settings (mcp_enabled)`

| Route | Since |
|-------|-------|
| `POST /mcp/disable` | v0.4.0 |
| `POST /mcp/enable` | v0.4.0 |

### `coolify_application`

| Route | Since |
|-------|-------|
| `POST /applications/public` | v0.1.0 |

### `coolify_application + variants`

| Route | Since |
|-------|-------|
| `DELETE /applications/{uuid}` | v0.1.0 |
| `PATCH /applications/{uuid}` | v0.1.0 |

### `coolify_application_destination`

| Route | Since |
|-------|-------|
| `DELETE /applications/{uuid}/destinations/{destination_uuid}` | v0.1.15 |
| `GET /applications/{uuid}/destinations` | v0.1.15 |
| `POST /applications/{uuid}/destinations` | v0.1.15 |

### `coolify_application_docker_image`

| Route | Since |
|-------|-------|
| `POST /applications/dockerimage` | v0.1.0 |

### `coolify_application_dockerfile`

| Route | Since |
|-------|-------|
| `POST /applications/dockerfile` | v0.2.0 |

### `coolify_application_github_app`

| Route | Since |
|-------|-------|
| `POST /applications/private-github-app` | v0.2.0 |

### `coolify_application_private_git`

| Route | Since |
|-------|-------|
| `POST /applications/private-deploy-key` | v0.1.0 |

### `coolify_cloud_init_script`

| Route | Since |
|-------|-------|
| `DELETE /cloud-init-scripts/{uuid}` | v0.1.15 |
| `GET /cloud-init-scripts` | v0.1.15 |
| `GET /cloud-init-scripts/{uuid}` | v0.1.15 |
| `PATCH /cloud-init-scripts/{uuid}` | v0.1.15 |
| `POST /cloud-init-scripts` | v0.1.15 |

### `coolify_cloud_token`

| Route | Since |
|-------|-------|
| `DELETE /cloud-tokens/{uuid}` | v0.2.0 |
| `PATCH /cloud-tokens/{uuid}` | v0.2.0 |
| `POST /cloud-tokens` | v0.2.0 |

### `coolify_database_backup`

| Route | Since |
|-------|-------|
| `DELETE /databases/{uuid}/backups/{scheduled_backup_uuid}` | v0.1.0 |
| `GET /databases/{uuid}/backups` | v0.1.0 |
| `PATCH /databases/{uuid}/backups/{scheduled_backup_uuid}` | v0.1.0 |
| `POST /databases/{uuid}/backups` | v0.1.0 |

### `coolify_database_clickhouse`

| Route | Since |
|-------|-------|
| `POST /databases/clickhouse` | v0.1.0 |

### `coolify_database_dragonfly`

| Route | Since |
|-------|-------|
| `POST /databases/dragonfly` | v0.1.0 |

### `coolify_database_keydb`

| Route | Since |
|-------|-------|
| `POST /databases/keydb` | v0.1.0 |

### `coolify_database_mariadb`

| Route | Since |
|-------|-------|
| `POST /databases/mariadb` | v0.1.0 |

### `coolify_database_mongodb`

| Route | Since |
|-------|-------|
| `POST /databases/mongodb` | v0.1.0 |

### `coolify_database_mysql`

| Route | Since |
|-------|-------|
| `POST /databases/mysql` | v0.1.0 |

### `coolify_database_postgresql`

| Route | Since |
|-------|-------|
| `POST /databases/postgresql` | v0.1.0 |

### `coolify_database_redis`

| Route | Since |
|-------|-------|
| `POST /databases/redis` | v0.1.0 |

### `coolify_deployment`

| Route | Since |
|-------|-------|
| `GET /applications/{uuid}/restart` | v0.1.0 |
| `GET /deployments/{uuid}` | v0.1.0 |

### `coolify_destination`

| Route | Since |
|-------|-------|
| `DELETE /destinations/{uuid}` | v0.2.0 |
| `PATCH /destinations/{uuid}` | v0.1.15 |
| `POST /servers/{server_uuid}/destinations` | v0.2.0 |

### `coolify_environment`

| Route | Since |
|-------|-------|
| `DELETE /projects/{uuid}/environments/{environment_name_or_uuid}` | v0.2.0 |
| `PATCH /projects/{uuid}/environments/{environment_name_or_uuid}` | v0.1.15 |
| `POST /projects/{uuid}/environments` | v0.2.0 |

### `coolify_environment_variable`

| Route | Since |
|-------|-------|
| `DELETE /applications/{uuid}/envs/{env_uuid}` | v0.1.0 |
| `DELETE /databases/{uuid}/envs/{env_uuid}` | v0.2.0 |
| `DELETE /services/{uuid}/envs/{env_uuid}` | v0.1.0 |
| `PATCH /applications/{uuid}/envs` | v0.1.0 |
| `PATCH /databases/{uuid}/envs` | v0.2.0 |
| `PATCH /services/{uuid}/envs` | v0.1.0 |
| `POST /applications/{uuid}/envs` | v0.1.0 |
| `POST /databases/{uuid}/envs` | v0.2.0 |
| `POST /services/{uuid}/envs` | v0.1.0 |

### `coolify_github_app`

| Route | Since |
|-------|-------|
| `DELETE /github-apps/{github_app_id}` | v0.2.0 |
| `PATCH /github-apps/{github_app_id}` | v0.2.0 |
| `POST /github-apps` | v0.2.0 |

### `coolify_gitlab_app`

| Route | Since |
|-------|-------|
| `DELETE /gitlab-apps/{gitlab_app_id}` | v0.1.15 |
| `GET /gitlab-apps` | v0.1.15 |
| `PATCH /gitlab-apps/{gitlab_app_id}` | v0.1.15 |
| `POST /gitlab-apps` | v0.1.15 |

### `coolify_notification_discord`

| Route | Since |
|-------|-------|
| `GET /notifications/discord` | v0.1.14 |
| `PATCH /notifications/discord` | v0.1.14 |

### `coolify_notification_email`

| Route | Since |
|-------|-------|
| `GET /notifications/email` | v0.1.14 |
| `PATCH /notifications/email` | v0.1.14 |

### `coolify_notification_pushover`

| Route | Since |
|-------|-------|
| `GET /notifications/pushover` | v0.1.14 |
| `PATCH /notifications/pushover` | v0.1.14 |

### `coolify_notification_slack`

| Route | Since |
|-------|-------|
| `GET /notifications/slack` | v0.1.14 |
| `PATCH /notifications/slack` | v0.1.14 |

### `coolify_notification_telegram`

| Route | Since |
|-------|-------|
| `GET /notifications/telegram` | v0.1.14 |
| `PATCH /notifications/telegram` | v0.1.14 |

### `coolify_notification_webhook`

| Route | Since |
|-------|-------|
| `GET /notifications/webhook` | v0.1.14 |
| `PATCH /notifications/webhook` | v0.1.14 |

### `coolify_private_key`

| Route | Since |
|-------|-------|
| `DELETE /security/keys/{uuid}` | v0.1.0 |
| `PATCH /security/keys/{uuid}` | v0.1.0 |
| `POST /security/keys` | v0.1.0 |

### `coolify_project`

| Route | Since |
|-------|-------|
| `DELETE /projects/{uuid}` | v0.1.0 |
| `PATCH /projects/{uuid}` | v0.1.0 |
| `POST /projects` | v0.1.0 |

### `coolify_resource_action`

| Route | Since |
|-------|-------|
| `GET /applications/{uuid}/start` | v0.3.0 |
| `GET /applications/{uuid}/stop` | v0.3.0 |
| `GET /databases/{uuid}/restart` | v0.3.0 |
| `GET /databases/{uuid}/start` | v0.3.0 |
| `GET /databases/{uuid}/stop` | v0.3.0 |
| `GET /services/{uuid}/restart` | v0.3.0 |
| `GET /services/{uuid}/start` | v0.3.0 |
| `GET /services/{uuid}/stop` | v0.3.0 |
| `POST /applications/{uuid}/restart` | v0.3.0 |
| `POST /applications/{uuid}/start` | v0.3.0 |
| `POST /applications/{uuid}/stop` | v0.3.0 |
| `POST /databases/{uuid}/restart` | v0.3.0 |
| `POST /databases/{uuid}/start` | v0.3.0 |
| `POST /databases/{uuid}/stop` | v0.3.0 |
| `POST /services/{uuid}/restart` | v0.3.0 |
| `POST /services/{uuid}/start` | v0.3.0 |
| `POST /services/{uuid}/stop` | v0.3.0 |

### `coolify_s3_storage`

| Route | Since |
|-------|-------|
| `DELETE /s3-storages/{uuid}` | v0.1.13 |
| `PATCH /s3-storages/{uuid}` | v0.1.13 |
| `POST /s3-storages` | v0.1.13 |

### `coolify_s3_storage_validate`

| Route | Since |
|-------|-------|
| `POST /s3-storages/{uuid}/validate` | v0.1.14 |

### `coolify_scheduled_task`

| Route | Since |
|-------|-------|
| `DELETE /applications/{uuid}/scheduled-tasks/{task_uuid}` | v0.2.0 |
| `DELETE /services/{uuid}/scheduled-tasks/{task_uuid}` | v0.2.0 |
| `PATCH /applications/{uuid}/scheduled-tasks/{task_uuid}` | v0.2.0 |
| `PATCH /services/{uuid}/scheduled-tasks/{task_uuid}` | v0.2.0 |
| `POST /applications/{uuid}/scheduled-tasks` | v0.2.0 |
| `POST /services/{uuid}/scheduled-tasks` | v0.2.0 |

### `coolify_server`

| Route | Since |
|-------|-------|
| `DELETE /servers/{uuid}` | v0.1.0 |
| `PATCH /servers/{uuid}` | v0.1.0 |
| `POST /servers` | v0.1.0 |

### `coolify_server_cloudflare_tunnel`

| Route | Since |
|-------|-------|
| `GET /servers/{uuid}/cloudflare-tunnel` | v0.1.15 |
| `PATCH /servers/{uuid}/cloudflare-tunnel` | v0.1.15 |

### `coolify_server_digitalocean`

| Route | Since |
|-------|-------|
| `POST /servers/digitalocean` | v0.2.0 |

### `coolify_server_docker_cleanup`

| Route | Since |
|-------|-------|
| `GET /servers/{uuid}/docker-cleanup` | v0.1.15 |
| `PATCH /servers/{uuid}/docker-cleanup` | v0.1.15 |

### `coolify_server_hetzner`

| Route | Since |
|-------|-------|
| `POST /servers/hetzner` | v0.2.0 |

### `coolify_server_log_drain`

| Route | Since |
|-------|-------|
| `GET /servers/{uuid}/log-drains` | v0.1.15 |
| `PATCH /servers/{uuid}/log-drains` | v0.1.15 |

### `coolify_server_proxy`

| Route | Since |
|-------|-------|
| `GET /servers/{uuid}/proxy` | v0.1.15 |
| `PATCH /servers/{uuid}/proxy` | v0.1.15 |
| `PUT /servers/{uuid}/proxy/configuration` | v0.1.15 |

### `coolify_server_sentinel`

| Route | Since |
|-------|-------|
| `GET /servers/{uuid}/sentinel` | v0.1.15 |
| `PATCH /servers/{uuid}/sentinel` | v0.1.15 |

### `coolify_server_validate`

| Route | Since |
|-------|-------|
| `POST /servers/{uuid}/validate` | v0.2.0 |

### `coolify_server_vultr`

| Route | Since |
|-------|-------|
| `POST /servers/vultr` | v0.2.0 |

### `coolify_service`

| Route | Since |
|-------|-------|
| `DELETE /services/{uuid}` | v0.1.0 |
| `PATCH /services/{uuid}` | v0.1.0 |
| `POST /services` | v0.1.0 |

### `coolify_shared_environment_variable`

| Route | Since |
|-------|-------|
| `DELETE /projects/{uuid}/environments/{environment_name_or_uuid}/envs/{env_id}` | v0.1.15 |
| `DELETE /projects/{uuid}/envs/{env_id}` | v0.1.15 |
| `DELETE /servers/{uuid}/envs/{env_id}` | v0.1.15 |
| `DELETE /team/envs/{env_id}` | v0.1.15 |
| `GET /projects/{uuid}/environments/{environment_name_or_uuid}/envs` | v0.1.15 |
| `GET /projects/{uuid}/envs` | v0.1.15 |
| `GET /servers/{uuid}/envs` | v0.1.15 |
| `GET /team/envs` | v0.1.15 |
| `PATCH /projects/{uuid}/environments/{environment_name_or_uuid}/envs/{env_id}` | v0.1.15 |
| `PATCH /projects/{uuid}/envs/{env_id}` | v0.1.15 |
| `PATCH /servers/{uuid}/envs/{env_id}` | v0.1.15 |
| `PATCH /team/envs/{env_id}` | v0.1.15 |
| `POST /projects/{uuid}/environments/{environment_name_or_uuid}/envs` | v0.1.15 |
| `POST /projects/{uuid}/envs` | v0.1.15 |
| `POST /servers/{uuid}/envs` | v0.1.15 |
| `POST /team/envs` | v0.1.15 |

### `coolify_storage`

| Route | Since |
|-------|-------|
| `DELETE /applications/{uuid}/storages/{storage_uuid}` | v0.2.0 |
| `DELETE /databases/{uuid}/storages/{storage_uuid}` | v0.2.0 |
| `DELETE /services/{uuid}/storages/{storage_uuid}` | v0.2.0 |
| `PATCH /applications/{uuid}/storages` | v0.2.0 |
| `PATCH /databases/{uuid}/storages` | v0.2.0 |
| `PATCH /services/{uuid}/storages` | v0.2.0 |
| `POST /applications/{uuid}/storages` | v0.2.0 |
| `POST /databases/{uuid}/storages` | v0.2.0 |
| `POST /services/{uuid}/storages` | v0.2.0 |

### `coolify_storage_backup`

| Route | Since |
|-------|-------|
| `DELETE /applications/{uuid}/storages/{storage_uuid}/backups` | v0.1.9 |
| `DELETE /databases/{uuid}/storages/{storage_uuid}/backups` | v0.1.9 |
| `DELETE /services/{uuid}/storages/{storage_uuid}/backups` | v0.1.9 |
| `PUT /applications/{uuid}/storages/{storage_uuid}/backups` | v0.1.9 |
| `PUT /databases/{uuid}/storages/{storage_uuid}/backups` | v0.1.9 |
| `PUT /services/{uuid}/storages/{storage_uuid}/backups` | v0.1.9 |

### `coolify_tag + coolify_resource_tag`

| Route | Since |
|-------|-------|
| `DELETE /applications/{uuid}/tags/{tag_uuid}` | v0.1.15 |
| `DELETE /databases/{uuid}/tags/{tag_uuid}` | v0.1.15 |
| `DELETE /services/{uuid}/tags/{tag_uuid}` | v0.1.15 |
| `DELETE /tags/{uuid}` | v0.1.15 |
| `GET /applications/{uuid}/tags` | v0.1.15 |
| `GET /databases/{uuid}/tags` | v0.1.15 |
| `GET /services/{uuid}/tags` | v0.1.15 |
| `GET /tags` | v0.1.15 |
| `PATCH /tags/{uuid}` | v0.1.15 |
| `POST /applications/{uuid}/tags` | v0.1.15 |
| `POST /databases/{uuid}/tags` | v0.1.15 |
| `POST /services/{uuid}/tags` | v0.1.15 |
| `POST /tags` | v0.1.15 |

### `data.coolify_application`

| Route | Since |
|-------|-------|
| `GET /applications/{uuid}` | v0.1.0 |

### `data.coolify_application_logs`

| Route | Since |
|-------|-------|
| `GET /applications/{uuid}/logs` | v0.2.0 |

### `data.coolify_applications`

| Route | Since |
|-------|-------|
| `GET /applications` | v0.1.0 |

### `data.coolify_backup_executions`

| Route | Since |
|-------|-------|
| `GET /databases/{uuid}/backups/{scheduled_backup_uuid}/executions` | v0.2.0 |

### `data.coolify_cloud_token`

| Route | Since |
|-------|-------|
| `GET /cloud-tokens/{uuid}` | v0.2.0 |

### `data.coolify_cloud_tokens`

| Route | Since |
|-------|-------|
| `GET /cloud-tokens` | v0.2.0 |

### `data.coolify_database`

| Route | Since |
|-------|-------|
| `GET /databases/{uuid}` | v0.1.0 |

### `data.coolify_databases`

| Route | Since |
|-------|-------|
| `GET /databases` | v0.1.0 |

### `data.coolify_deployments`

| Route | Since |
|-------|-------|
| `GET /deployments` | v0.2.0 |
| `GET /deployments/applications/{uuid}` | v0.2.0 |

### `data.coolify_destination`

| Route | Since |
|-------|-------|
| `GET /destinations/{uuid}` | v0.2.0 |

### `data.coolify_destinations`

| Route | Since |
|-------|-------|
| `GET /destinations` | v0.2.0 |
| `GET /servers/{server_uuid}/destinations` | v0.2.0 |

### `data.coolify_digitalocean_images`

| Route | Since |
|-------|-------|
| `GET /digitalocean/images` | v0.2.0 |

### `data.coolify_digitalocean_regions`

| Route | Since |
|-------|-------|
| `GET /digitalocean/regions` | v0.2.0 |

### `data.coolify_digitalocean_sizes`

| Route | Since |
|-------|-------|
| `GET /digitalocean/sizes` | v0.2.0 |

### `data.coolify_digitalocean_ssh_keys`

| Route | Since |
|-------|-------|
| `GET /digitalocean/ssh-keys` | v0.2.0 |

### `data.coolify_environment`

| Route | Since |
|-------|-------|
| `GET /projects/{uuid}/{environment_name_or_uuid}` | v0.2.0 |

### `data.coolify_environment_variables`

| Route | Since |
|-------|-------|
| `GET /applications/{uuid}/envs` | v0.1.0 |
| `GET /databases/{uuid}/envs` | v0.2.0 |
| `GET /services/{uuid}/envs` | v0.2.0 |

### `data.coolify_environments`

| Route | Since |
|-------|-------|
| `GET /projects/{uuid}/environments` | v0.2.0 |

### `data.coolify_github_app_branches`

| Route | Since |
|-------|-------|
| `GET /github-apps/{github_app_id}/repositories/{owner}/{repo}/branches` | v0.2.0 |

### `data.coolify_github_app_repositories`

| Route | Since |
|-------|-------|
| `GET /github-apps/{github_app_id}/repositories` | v0.2.0 |

### `data.coolify_github_apps`

| Route | Since |
|-------|-------|
| `GET /github-apps` | v0.2.0 |

### `data.coolify_health`

| Route | Since |
|-------|-------|
| `GET /health` | v0.2.0 |

### `data.coolify_hetzner_firewalls`

| Route | Since |
|-------|-------|
| `GET /hetzner/firewalls` | v0.1.16 |

### `data.coolify_hetzner_images`

| Route | Since |
|-------|-------|
| `GET /hetzner/images` | v0.2.0 |

### `data.coolify_hetzner_locations`

| Route | Since |
|-------|-------|
| `GET /hetzner/locations` | v0.2.0 |

### `data.coolify_hetzner_networks`

| Route | Since |
|-------|-------|
| `GET /hetzner/networks` | v0.1.16 |

### `data.coolify_hetzner_server_types`

| Route | Since |
|-------|-------|
| `GET /hetzner/server-types` | v0.2.0 |

### `data.coolify_hetzner_ssh_keys`

| Route | Since |
|-------|-------|
| `GET /hetzner/ssh-keys` | v0.2.0 |

### `data.coolify_private_key`

| Route | Since |
|-------|-------|
| `GET /security/keys/{uuid}` | v0.1.0 |

### `data.coolify_private_keys`

| Route | Since |
|-------|-------|
| `GET /security/keys` | v0.1.0 |

### `data.coolify_project`

| Route | Since |
|-------|-------|
| `GET /projects/{uuid}` | v0.1.0 |

### `data.coolify_projects`

| Route | Since |
|-------|-------|
| `GET /projects` | v0.1.0 |

### `data.coolify_resources`

| Route | Since |
|-------|-------|
| `GET /resources` | v0.2.0 |

### `data.coolify_s3_storage`

| Route | Since |
|-------|-------|
| `GET /s3-storages/{uuid}` | v0.1.13 |

### `data.coolify_s3_storages`

| Route | Since |
|-------|-------|
| `GET /s3-storages` | v0.1.13 |

### `data.coolify_scheduled_tasks`

| Route | Since |
|-------|-------|
| `GET /applications/{uuid}/scheduled-tasks` | v0.2.0 |
| `GET /services/{uuid}/scheduled-tasks` | v0.2.0 |

### `data.coolify_server`

| Route | Since |
|-------|-------|
| `GET /servers/{uuid}` | v0.1.0 |

### `data.coolify_server_domains`

| Route | Since |
|-------|-------|
| `GET /servers/{uuid}/domains` | v0.1.0 |

### `data.coolify_server_resources`

| Route | Since |
|-------|-------|
| `GET /servers/{uuid}/resources` | v0.1.0 |

### `data.coolify_server_validation`

| Route | Since |
|-------|-------|
| `GET /servers/{uuid}/validate` | v0.2.0 |

### `data.coolify_servers`

| Route | Since |
|-------|-------|
| `GET /servers` | v0.1.0 |

### `data.coolify_service`

| Route | Since |
|-------|-------|
| `GET /services/{uuid}` | v0.1.0 |

### `data.coolify_services`

| Route | Since |
|-------|-------|
| `GET /services` | v0.1.0 |

### `data.coolify_storages`

| Route | Since |
|-------|-------|
| `GET /applications/{uuid}/storages` | v0.2.0 |
| `GET /databases/{uuid}/storages` | v0.2.0 |
| `GET /services/{uuid}/storages` | v0.2.0 |

### `data.coolify_task_executions`

| Route | Since |
|-------|-------|
| `GET /applications/{uuid}/scheduled-tasks/{task_uuid}/executions` | v0.2.0 |
| `GET /services/{uuid}/scheduled-tasks/{task_uuid}/executions` | v0.2.0 |

### `data.coolify_team`

| Route | Since |
|-------|-------|
| `GET /teams/{id}` | v0.1.0 |

### `data.coolify_team_members`

| Route | Since |
|-------|-------|
| `GET /teams/current` | v0.2.0 |
| `GET /teams/current/members` | v0.2.0 |
| `GET /teams/{id}/members` | v0.2.0 |

### `data.coolify_teams`

| Route | Since |
|-------|-------|
| `GET /teams` | v0.2.0 |

### `data.coolify_version`

| Route | Since |
|-------|-------|
| `GET /version` | v0.1.0 |

### `data.coolify_vultr_os`

| Route | Since |
|-------|-------|
| `GET /vultr/os` | v0.2.0 |

### `data.coolify_vultr_plans`

| Route | Since |
|-------|-------|
| `GET /vultr/plans` | v0.2.0 |

### `data.coolify_vultr_regions`

| Route | Since |
|-------|-------|
| `GET /vultr/regions` | v0.2.0 |

### `data.coolify_vultr_ssh_keys`

| Route | Since |
|-------|-------|
| `GET /vultr/ssh-keys` | v0.2.0 |

## Appendix: all classified routes

Completeness tests use this list. Sorted by `METHOD /path`.

| Route | Status | Resource or skip class |
|-------|--------|------------------------|
| `DELETE /applications/{uuid}` | covered | `coolify_application + variants` |
| `DELETE /applications/{uuid}/destinations/{destination_uuid}` | covered | `coolify_application_destination` |
| `DELETE /applications/{uuid}/envs/{env_uuid}` | covered | `coolify_environment_variable` |
| `DELETE /applications/{uuid}/previews/{pull_request_id}` | covered | `client.DeletePreviewDeployment` |
| `DELETE /applications/{uuid}/scheduled-tasks/{task_uuid}` | covered | `coolify_scheduled_task` |
| `DELETE /applications/{uuid}/storages/{storage_uuid}` | covered | `coolify_storage` |
| `DELETE /applications/{uuid}/storages/{storage_uuid}/backups` | covered | `coolify_storage_backup` |
| `DELETE /applications/{uuid}/tags/{tag_uuid}` | covered | `coolify_tag + coolify_resource_tag` |
| `DELETE /cloud-init-scripts/{uuid}` | covered | `coolify_cloud_init_script` |
| `DELETE /cloud-tokens/{uuid}` | covered | `coolify_cloud_token` |
| `DELETE /databases/{uuid}` | covered | `coolify_*_database` |
| `DELETE /databases/{uuid}/backups/{scheduled_backup_uuid}` | covered | `coolify_database_backup` |
| `DELETE /databases/{uuid}/backups/{scheduled_backup_uuid}/executions/{execution_uuid}` | covered | `client.DeleteBackupExecution` |
| `DELETE /databases/{uuid}/envs/{env_uuid}` | covered | `coolify_environment_variable` |
| `DELETE /databases/{uuid}/storages/{storage_uuid}` | covered | `coolify_storage` |
| `DELETE /databases/{uuid}/storages/{storage_uuid}/backups` | covered | `coolify_storage_backup` |
| `DELETE /databases/{uuid}/tags/{tag_uuid}` | covered | `coolify_tag + coolify_resource_tag` |
| `DELETE /destinations/{uuid}` | covered | `coolify_destination` |
| `DELETE /github-apps/{github_app_id}` | covered | `coolify_github_app` |
| `DELETE /gitlab-apps/{gitlab_app_id}` | covered | `coolify_gitlab_app` |
| `DELETE /projects/{uuid}` | covered | `coolify_project` |
| `DELETE /projects/{uuid}/environments/{environment_name_or_uuid}` | covered | `coolify_environment` |
| `DELETE /projects/{uuid}/environments/{environment_name_or_uuid}/envs/{env_id}` | covered | `coolify_shared_environment_variable` |
| `DELETE /projects/{uuid}/envs/{env_id}` | covered | `coolify_shared_environment_variable` |
| `DELETE /s3-storages/{uuid}` | covered | `coolify_s3_storage` |
| `DELETE /security/keys/{uuid}` | covered | `coolify_private_key` |
| `DELETE /servers/{uuid}` | covered | `coolify_server` |
| `DELETE /servers/{uuid}/envs/{env_id}` | covered | `coolify_shared_environment_variable` |
| `DELETE /services/{uuid}` | covered | `coolify_service` |
| `DELETE /services/{uuid}/envs/{env_uuid}` | covered | `coolify_environment_variable` |
| `DELETE /services/{uuid}/scheduled-tasks/{task_uuid}` | covered | `coolify_scheduled_task` |
| `DELETE /services/{uuid}/storages/{storage_uuid}` | covered | `coolify_storage` |
| `DELETE /services/{uuid}/storages/{storage_uuid}/backups` | covered | `coolify_storage_backup` |
| `DELETE /services/{uuid}/tags/{tag_uuid}` | covered | `coolify_tag + coolify_resource_tag` |
| `DELETE /tags/{uuid}` | covered | `coolify_tag + coolify_resource_tag` |
| `DELETE /team/envs/{env_id}` | covered | `coolify_shared_environment_variable` |
| `GET /applications` | covered | `data.coolify_applications` |
| `GET /applications/{uuid}` | covered | `data.coolify_application` |
| `GET /applications/{uuid}/destinations` | covered | `coolify_application_destination` |
| `GET /applications/{uuid}/envs` | covered | `data.coolify_environment_variables` |
| `GET /applications/{uuid}/logs` | covered | `data.coolify_application_logs` |
| `GET /applications/{uuid}/restart` | covered | `coolify_deployment` |
| `GET /applications/{uuid}/rollback-images` | skipped | `rollback` |
| `GET /applications/{uuid}/scheduled-tasks` | covered | `data.coolify_scheduled_tasks` |
| `GET /applications/{uuid}/scheduled-tasks/{task_uuid}/executions` | covered | `data.coolify_task_executions` |
| `GET /applications/{uuid}/start` | covered | `coolify_resource_action` |
| `GET /applications/{uuid}/stop` | covered | `coolify_resource_action` |
| `GET /applications/{uuid}/storages` | covered | `data.coolify_storages` |
| `GET /applications/{uuid}/tags` | covered | `coolify_tag + coolify_resource_tag` |
| `GET /cloud-init-scripts` | covered | `coolify_cloud_init_script` |
| `GET /cloud-init-scripts/{uuid}` | covered | `coolify_cloud_init_script` |
| `GET /cloud-tokens` | covered | `data.coolify_cloud_tokens` |
| `GET /cloud-tokens/{uuid}` | covered | `data.coolify_cloud_token` |
| `GET /databases` | covered | `data.coolify_databases` |
| `GET /databases/{uuid}` | covered | `data.coolify_database` |
| `GET /databases/{uuid}/backups` | covered | `coolify_database_backup` |
| `GET /databases/{uuid}/backups/{scheduled_backup_uuid}/executions` | covered | `data.coolify_backup_executions` |
| `GET /databases/{uuid}/envs` | covered | `data.coolify_environment_variables` |
| `GET /databases/{uuid}/logs` | skipped | `logs` |
| `GET /databases/{uuid}/restart` | covered | `coolify_resource_action` |
| `GET /databases/{uuid}/start` | covered | `coolify_resource_action` |
| `GET /databases/{uuid}/stop` | covered | `coolify_resource_action` |
| `GET /databases/{uuid}/storages` | covered | `data.coolify_storages` |
| `GET /databases/{uuid}/tags` | covered | `coolify_tag + coolify_resource_tag` |
| `GET /deploy` | covered | `client.Deploy` |
| `GET /deployments` | covered | `data.coolify_deployments` |
| `GET /deployments/applications/{uuid}` | covered | `data.coolify_deployments` |
| `GET /deployments/{uuid}` | covered | `coolify_deployment` |
| `GET /destinations` | covered | `data.coolify_destinations` |
| `GET /destinations/{uuid}` | covered | `data.coolify_destination` |
| `GET /digitalocean/images` | covered | `data.coolify_digitalocean_images` |
| `GET /digitalocean/regions` | covered | `data.coolify_digitalocean_regions` |
| `GET /digitalocean/sizes` | covered | `data.coolify_digitalocean_sizes` |
| `GET /digitalocean/ssh-keys` | covered | `data.coolify_digitalocean_ssh_keys` |
| `GET /disable` | covered | `client.DisableAPI` |
| `GET /enable` | covered | `client.EnableAPI` |
| `GET /github-apps` | covered | `data.coolify_github_apps` |
| `GET /github-apps/{github_app_id}/repositories` | covered | `data.coolify_github_app_repositories` |
| `GET /github-apps/{github_app_id}/repositories/{owner}/{repo}/branches` | covered | `data.coolify_github_app_branches` |
| `GET /gitlab-apps` | covered | `coolify_gitlab_app` |
| `GET /health` | covered | `data.coolify_health` |
| `GET /hetzner/firewalls` | covered | `data.coolify_hetzner_firewalls` |
| `GET /hetzner/images` | covered | `data.coolify_hetzner_images` |
| `GET /hetzner/locations` | covered | `data.coolify_hetzner_locations` |
| `GET /hetzner/networks` | covered | `data.coolify_hetzner_networks` |
| `GET /hetzner/server-types` | covered | `data.coolify_hetzner_server_types` |
| `GET /hetzner/ssh-keys` | covered | `data.coolify_hetzner_ssh_keys` |
| `GET /notifications/discord` | covered | `coolify_notification_discord` |
| `GET /notifications/email` | covered | `coolify_notification_email` |
| `GET /notifications/pushover` | covered | `coolify_notification_pushover` |
| `GET /notifications/slack` | covered | `coolify_notification_slack` |
| `GET /notifications/telegram` | covered | `coolify_notification_telegram` |
| `GET /notifications/webhook` | covered | `coolify_notification_webhook` |
| `GET /projects` | covered | `data.coolify_projects` |
| `GET /projects/{uuid}` | covered | `data.coolify_project` |
| `GET /projects/{uuid}/environments` | covered | `data.coolify_environments` |
| `GET /projects/{uuid}/environments/{environment_name_or_uuid}/envs` | covered | `coolify_shared_environment_variable` |
| `GET /projects/{uuid}/envs` | covered | `coolify_shared_environment_variable` |
| `GET /projects/{uuid}/{environment_name_or_uuid}` | covered | `data.coolify_environment` |
| `GET /resources` | covered | `data.coolify_resources` |
| `GET /s3-storages` | covered | `data.coolify_s3_storages` |
| `GET /s3-storages/{uuid}` | covered | `data.coolify_s3_storage` |
| `GET /security/keys` | covered | `data.coolify_private_keys` |
| `GET /security/keys/{uuid}` | covered | `data.coolify_private_key` |
| `GET /servers` | covered | `data.coolify_servers` |
| `GET /servers/{server_uuid}/destinations` | covered | `data.coolify_destinations` |
| `GET /servers/{uuid}` | covered | `data.coolify_server` |
| `GET /servers/{uuid}/cloudflare-tunnel` | covered | `coolify_server_cloudflare_tunnel` |
| `GET /servers/{uuid}/docker-cleanup` | covered | `coolify_server_docker_cleanup` |
| `GET /servers/{uuid}/docker-cleanup/executions` | skipped | `control-plane` |
| `GET /servers/{uuid}/domains` | covered | `data.coolify_server_domains` |
| `GET /servers/{uuid}/envs` | covered | `coolify_shared_environment_variable` |
| `GET /servers/{uuid}/export` | skipped | `control-plane` |
| `GET /servers/{uuid}/log-drains` | covered | `coolify_server_log_drain` |
| `GET /servers/{uuid}/proxy` | covered | `coolify_server_proxy` |
| `GET /servers/{uuid}/resources` | covered | `data.coolify_server_resources` |
| `GET /servers/{uuid}/sentinel` | covered | `coolify_server_sentinel` |
| `GET /servers/{uuid}/validate` | covered | `data.coolify_server_validation` |
| `GET /services` | covered | `data.coolify_services` |
| `GET /services/{uuid}` | covered | `data.coolify_service` |
| `GET /services/{uuid}/applications` | skipped | `nested-service` |
| `GET /services/{uuid}/applications/{app_uuid}` | skipped | `nested-service` |
| `GET /services/{uuid}/applications/{app_uuid}/logs` | skipped | `logs` |
| `GET /services/{uuid}/applications/{app_uuid}/restart` | skipped | `nested-service` |
| `GET /services/{uuid}/applications/{app_uuid}/start` | skipped | `nested-service` |
| `GET /services/{uuid}/applications/{app_uuid}/stop` | skipped | `nested-service` |
| `GET /services/{uuid}/databases` | skipped | `nested-service` |
| `GET /services/{uuid}/databases/{database_uuid}` | skipped | `nested-service` |
| `GET /services/{uuid}/databases/{database_uuid}/logs` | skipped | `logs` |
| `GET /services/{uuid}/envs` | covered | `data.coolify_environment_variables` |
| `GET /services/{uuid}/logs` | skipped | `logs` |
| `GET /services/{uuid}/restart` | covered | `coolify_resource_action` |
| `GET /services/{uuid}/scheduled-tasks` | covered | `data.coolify_scheduled_tasks` |
| `GET /services/{uuid}/scheduled-tasks/{task_uuid}/executions` | covered | `data.coolify_task_executions` |
| `GET /services/{uuid}/start` | covered | `coolify_resource_action` |
| `GET /services/{uuid}/stop` | covered | `coolify_resource_action` |
| `GET /services/{uuid}/storages` | covered | `data.coolify_storages` |
| `GET /services/{uuid}/tags` | covered | `coolify_tag + coolify_resource_tag` |
| `GET /tags` | covered | `coolify_tag + coolify_resource_tag` |
| `GET /team` | skipped | `alias` |
| `GET /team/envs` | covered | `coolify_shared_environment_variable` |
| `GET /team/members` | skipped | `alias` |
| `GET /teams` | covered | `data.coolify_teams` |
| `GET /teams/current` | covered | `data.coolify_team_members` |
| `GET /teams/current/members` | covered | `data.coolify_team_members` |
| `GET /teams/{id}` | covered | `data.coolify_team` |
| `GET /teams/{id}/members` | covered | `data.coolify_team_members` |
| `GET /version` | covered | `data.coolify_version` |
| `GET /vultr/os` | covered | `data.coolify_vultr_os` |
| `GET /vultr/plans` | covered | `data.coolify_vultr_plans` |
| `GET /vultr/regions` | covered | `data.coolify_vultr_regions` |
| `GET /vultr/ssh-keys` | covered | `data.coolify_vultr_ssh_keys` |
| `PATCH /applications/{uuid}` | covered | `coolify_application + variants` |
| `PATCH /applications/{uuid}/envs` | covered | `coolify_environment_variable` |
| `PATCH /applications/{uuid}/envs/bulk` | covered | `client.BulkUpdateEnvVars` |
| `PATCH /applications/{uuid}/scheduled-tasks/{task_uuid}` | covered | `coolify_scheduled_task` |
| `PATCH /applications/{uuid}/storages` | covered | `coolify_storage` |
| `PATCH /cloud-init-scripts/{uuid}` | covered | `coolify_cloud_init_script` |
| `PATCH /cloud-tokens/{uuid}` | covered | `coolify_cloud_token` |
| `PATCH /databases/{uuid}` | covered | `coolify_*_database` |
| `PATCH /databases/{uuid}/backups/{scheduled_backup_uuid}` | covered | `coolify_database_backup` |
| `PATCH /databases/{uuid}/envs` | covered | `coolify_environment_variable` |
| `PATCH /databases/{uuid}/envs/bulk` | covered | `client.BulkUpdateEnvVars` |
| `PATCH /databases/{uuid}/storages` | covered | `coolify_storage` |
| `PATCH /destinations/{uuid}` | covered | `coolify_destination` |
| `PATCH /github-apps/{github_app_id}` | covered | `coolify_github_app` |
| `PATCH /gitlab-apps/{gitlab_app_id}` | covered | `coolify_gitlab_app` |
| `PATCH /notifications/discord` | covered | `coolify_notification_discord` |
| `PATCH /notifications/email` | covered | `coolify_notification_email` |
| `PATCH /notifications/pushover` | covered | `coolify_notification_pushover` |
| `PATCH /notifications/slack` | covered | `coolify_notification_slack` |
| `PATCH /notifications/telegram` | covered | `coolify_notification_telegram` |
| `PATCH /notifications/webhook` | covered | `coolify_notification_webhook` |
| `PATCH /projects/{uuid}` | covered | `coolify_project` |
| `PATCH /projects/{uuid}/environments/{environment_name_or_uuid}` | covered | `coolify_environment` |
| `PATCH /projects/{uuid}/environments/{environment_name_or_uuid}/envs/{env_id}` | covered | `coolify_shared_environment_variable` |
| `PATCH /projects/{uuid}/envs/{env_id}` | covered | `coolify_shared_environment_variable` |
| `PATCH /s3-storages/{uuid}` | covered | `coolify_s3_storage` |
| `PATCH /security/keys/{uuid}` | covered | `coolify_private_key` |
| `PATCH /servers/{uuid}` | covered | `coolify_server` |
| `PATCH /servers/{uuid}/cloudflare-tunnel` | covered | `coolify_server_cloudflare_tunnel` |
| `PATCH /servers/{uuid}/docker-cleanup` | covered | `coolify_server_docker_cleanup` |
| `PATCH /servers/{uuid}/envs/{env_id}` | covered | `coolify_shared_environment_variable` |
| `PATCH /servers/{uuid}/log-drains` | covered | `coolify_server_log_drain` |
| `PATCH /servers/{uuid}/proxy` | covered | `coolify_server_proxy` |
| `PATCH /servers/{uuid}/sentinel` | covered | `coolify_server_sentinel` |
| `PATCH /services/{uuid}` | covered | `coolify_service` |
| `PATCH /services/{uuid}/applications/{app_uuid}` | skipped | `nested-service` |
| `PATCH /services/{uuid}/databases/{database_uuid}` | skipped | `nested-service` |
| `PATCH /services/{uuid}/envs` | covered | `coolify_environment_variable` |
| `PATCH /services/{uuid}/envs/bulk` | covered | `client.BulkUpdateEnvVars` |
| `PATCH /services/{uuid}/scheduled-tasks/{task_uuid}` | covered | `coolify_scheduled_task` |
| `PATCH /services/{uuid}/storages` | covered | `coolify_storage` |
| `PATCH /tags/{uuid}` | covered | `coolify_tag + coolify_resource_tag` |
| `PATCH /team/envs/{env_id}` | covered | `coolify_shared_environment_variable` |
| `POST /applications/dockercompose` | skipped | `deprecated` |
| `POST /applications/dockerfile` | covered | `coolify_application_dockerfile` |
| `POST /applications/dockerimage` | covered | `coolify_application_docker_image` |
| `POST /applications/private-deploy-key` | covered | `coolify_application_private_git` |
| `POST /applications/private-github-app` | covered | `coolify_application_github_app` |
| `POST /applications/public` | covered | `coolify_application` |
| `POST /applications/{uuid}/clone` | skipped | `clone-move` |
| `POST /applications/{uuid}/destinations` | covered | `coolify_application_destination` |
| `POST /applications/{uuid}/envs` | covered | `coolify_environment_variable` |
| `POST /applications/{uuid}/migrate` | skipped | `clone-move` |
| `POST /applications/{uuid}/move` | skipped | `clone-move` |
| `POST /applications/{uuid}/restart` | covered | `coolify_resource_action` |
| `POST /applications/{uuid}/rollback` | skipped | `rollback` |
| `POST /applications/{uuid}/scheduled-tasks` | covered | `coolify_scheduled_task` |
| `POST /applications/{uuid}/scheduled-tasks/{task_uuid}/execute` | skipped | `run-now` |
| `POST /applications/{uuid}/start` | covered | `coolify_resource_action` |
| `POST /applications/{uuid}/stop` | covered | `coolify_resource_action` |
| `POST /applications/{uuid}/storages` | covered | `coolify_storage` |
| `POST /applications/{uuid}/storages/{storage_uuid}/backups/run` | skipped | `run-now` |
| `POST /applications/{uuid}/tags` | covered | `coolify_tag + coolify_resource_tag` |
| `POST /cloud-init-scripts` | covered | `coolify_cloud_init_script` |
| `POST /cloud-tokens` | covered | `coolify_cloud_token` |
| `POST /cloud-tokens/{uuid}/validate` | covered | `client.ValidateCloudToken` |
| `POST /databases/clickhouse` | covered | `coolify_database_clickhouse` |
| `POST /databases/dragonfly` | covered | `coolify_database_dragonfly` |
| `POST /databases/keydb` | covered | `coolify_database_keydb` |
| `POST /databases/mariadb` | covered | `coolify_database_mariadb` |
| `POST /databases/mongodb` | covered | `coolify_database_mongodb` |
| `POST /databases/mysql` | covered | `coolify_database_mysql` |
| `POST /databases/postgresql` | covered | `coolify_database_postgresql` |
| `POST /databases/redis` | covered | `coolify_database_redis` |
| `POST /databases/{uuid}/backups` | covered | `coolify_database_backup` |
| `POST /databases/{uuid}/clone` | skipped | `clone-move` |
| `POST /databases/{uuid}/envs` | covered | `coolify_environment_variable` |
| `POST /databases/{uuid}/migrate` | skipped | `clone-move` |
| `POST /databases/{uuid}/move` | skipped | `clone-move` |
| `POST /databases/{uuid}/restart` | covered | `coolify_resource_action` |
| `POST /databases/{uuid}/start` | covered | `coolify_resource_action` |
| `POST /databases/{uuid}/stop` | covered | `coolify_resource_action` |
| `POST /databases/{uuid}/storages` | covered | `coolify_storage` |
| `POST /databases/{uuid}/storages/{storage_uuid}/backups/run` | skipped | `run-now` |
| `POST /databases/{uuid}/tags` | covered | `coolify_tag + coolify_resource_tag` |
| `POST /deploy` | covered | `client.Deploy` |
| `POST /deployments/{uuid}/cancel` | covered | `client.CancelDeployment` |
| `POST /disable` | covered | `coolify_api_settings` |
| `POST /enable` | covered | `coolify_api_settings` |
| `POST /feedback` | skipped | `not-infra` |
| `POST /github-apps` | covered | `coolify_github_app` |
| `POST /gitlab-apps` | covered | `coolify_gitlab_app` |
| `POST /mcp/disable` | covered | `coolify_api_settings (mcp_enabled)` |
| `POST /mcp/enable` | covered | `coolify_api_settings (mcp_enabled)` |
| `POST /projects` | covered | `coolify_project` |
| `POST /projects/{uuid}/environments` | covered | `coolify_environment` |
| `POST /projects/{uuid}/environments/{environment_name_or_uuid}/envs` | covered | `coolify_shared_environment_variable` |
| `POST /projects/{uuid}/envs` | covered | `coolify_shared_environment_variable` |
| `POST /s3-storages` | covered | `coolify_s3_storage` |
| `POST /s3-storages/{uuid}/validate` | covered | `coolify_s3_storage_validate` |
| `POST /security/keys` | covered | `coolify_private_key` |
| `POST /sentinel/push` | skipped | `control-plane` |
| `POST /servers` | covered | `coolify_server` |
| `POST /servers/digitalocean` | covered | `coolify_server_digitalocean` |
| `POST /servers/hetzner` | covered | `coolify_server_hetzner` |
| `POST /servers/import` | skipped | `control-plane` |
| `POST /servers/vultr` | covered | `coolify_server_vultr` |
| `POST /servers/{server_uuid}/destinations` | covered | `coolify_destination` |
| `POST /servers/{uuid}/claim` | skipped | `control-plane` |
| `POST /servers/{uuid}/cloudflare-tunnel/disable` | skipped | `enable-disable` |
| `POST /servers/{uuid}/cloudflare-tunnel/enable` | skipped | `enable-disable` |
| `POST /servers/{uuid}/docker-cleanup/run` | skipped | `control-plane` |
| `POST /servers/{uuid}/envs` | covered | `coolify_shared_environment_variable` |
| `POST /servers/{uuid}/export/mailbox` | skipped | `control-plane` |
| `POST /servers/{uuid}/migrate` | skipped | `clone-move` |
| `POST /servers/{uuid}/proxy/restart` | skipped | `control-plane` |
| `POST /servers/{uuid}/transfer/complete` | skipped | `control-plane` |
| `POST /servers/{uuid}/validate` | covered | `coolify_server_validate` |
| `POST /services` | covered | `coolify_service` |
| `POST /services/{uuid}/applications/{app_uuid}/logs` | skipped | `logs` |
| `POST /services/{uuid}/applications/{app_uuid}/restart` | skipped | `nested-service` |
| `POST /services/{uuid}/applications/{app_uuid}/start` | skipped | `nested-service` |
| `POST /services/{uuid}/applications/{app_uuid}/stop` | skipped | `nested-service` |
| `POST /services/{uuid}/clone` | skipped | `clone-move` |
| `POST /services/{uuid}/databases/{database_uuid}/restart` | skipped | `nested-service` |
| `POST /services/{uuid}/databases/{database_uuid}/start` | skipped | `nested-service` |
| `POST /services/{uuid}/databases/{database_uuid}/stop` | skipped | `nested-service` |
| `POST /services/{uuid}/envs` | covered | `coolify_environment_variable` |
| `POST /services/{uuid}/migrate` | skipped | `clone-move` |
| `POST /services/{uuid}/move` | skipped | `clone-move` |
| `POST /services/{uuid}/restart` | covered | `coolify_resource_action` |
| `POST /services/{uuid}/scheduled-tasks` | covered | `coolify_scheduled_task` |
| `POST /services/{uuid}/scheduled-tasks/{task_uuid}/execute` | skipped | `run-now` |
| `POST /services/{uuid}/start` | covered | `coolify_resource_action` |
| `POST /services/{uuid}/stop` | covered | `coolify_resource_action` |
| `POST /services/{uuid}/storages` | covered | `coolify_storage` |
| `POST /services/{uuid}/storages/{storage_uuid}/backups/run` | skipped | `run-now` |
| `POST /services/{uuid}/tags` | covered | `coolify_tag + coolify_resource_tag` |
| `POST /tags` | covered | `coolify_tag + coolify_resource_tag` |
| `POST /team/envs` | covered | `coolify_shared_environment_variable` |
| `PUT /applications/{uuid}/storages/{storage_uuid}/backups` | covered | `coolify_storage_backup` |
| `PUT /databases/{uuid}/storages/{storage_uuid}/backups` | covered | `coolify_storage_backup` |
| `PUT /servers/{uuid}/proxy/configuration` | covered | `coolify_server_proxy` |
| `PUT /services/{uuid}/storages/{storage_uuid}/backups` | covered | `coolify_storage_backup` |

## Unclassified contract routes

_None. All pin contract routes are classified in `coveredEndpoints()`._

When `make contract-extract` adds routes, classify them in
`internal/spectest/coverage_test.go` or `TestSpecCoverage_Completeness` fails.
