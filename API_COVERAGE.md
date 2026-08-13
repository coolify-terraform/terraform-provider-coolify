# API Coverage

<!-- Auto-generated from internal/spectest/coverage_test.go. Do not edit manually. -->
<!-- Run: make api-coverage -->

**Route inventory**: source-derived contract `testdata/contracts/coolify-v4.json` (`routes[]`)  
**Field source of truth**: same contract (`models` / endpoint allow lists)  
**Not inventory**: OpenAPI under `testdata/specs/` (partial upstream path list; do not treat as Coolify API completeness)  
**Coverage**: 176 covered / 296 registry entries (59.5%)  
**Planned**: 12 | **Skipped**: 108  
**Registry size**: 296 (contract routes + allowlisted extras)

## Covered

| Endpoint | Terraform Resource / Data Source | Since |
|----------|----------------------------------|-------|
| `DELETE /applications/{uuid}` | `coolify_application + variants` | v0.1.0 |
| `DELETE /applications/{uuid}/envs/{env_uuid}` | `coolify_environment_variable` | v0.1.0 |
| `DELETE /applications/{uuid}/previews/{pull_request_id}` | `client.DeletePreviewDeployment` | v0.2.0 |
| `DELETE /applications/{uuid}/scheduled-tasks/{task_uuid}` | `coolify_scheduled_task` | v0.2.0 |
| `DELETE /applications/{uuid}/storages/{storage_uuid}` | `coolify_storage` | v0.2.0 |
| `DELETE /applications/{uuid}/storages/{storage_uuid}/backups` | `coolify_storage_backup` | v0.1.9 |
| `DELETE /cloud-tokens/{uuid}` | `coolify_cloud_token` | v0.2.0 |
| `DELETE /databases/{uuid}` | `coolify_*_database` | v0.1.0 |
| `DELETE /databases/{uuid}/backups/{scheduled_backup_uuid}` | `coolify_database_backup` | v0.1.0 |
| `DELETE /databases/{uuid}/backups/{scheduled_backup_uuid}/executions/{execution_uuid}` | `client.DeleteBackupExecution` | v0.2.0 |
| `DELETE /databases/{uuid}/envs/{env_uuid}` | `coolify_environment_variable` | v0.2.0 |
| `DELETE /databases/{uuid}/storages/{storage_uuid}` | `coolify_storage` | v0.2.0 |
| `DELETE /databases/{uuid}/storages/{storage_uuid}/backups` | `coolify_storage_backup` | v0.1.9 |
| `DELETE /destinations/{uuid}` | `coolify_destination` | v0.2.0 |
| `DELETE /github-apps/{github_app_id}` | `coolify_github_app` | v0.2.0 |
| `DELETE /projects/{uuid}` | `coolify_project` | v0.1.0 |
| `DELETE /projects/{uuid}/environments/{environment_name_or_uuid}` | `coolify_environment` | v0.2.0 |
| `DELETE /s3-storages/{uuid}` | `coolify_s3_storage` | v0.1.13 |
| `DELETE /security/keys/{uuid}` | `coolify_private_key` | v0.1.0 |
| `DELETE /servers/{uuid}` | `coolify_server` | v0.1.0 |
| `DELETE /services/{uuid}` | `coolify_service` | v0.1.0 |
| `DELETE /services/{uuid}/envs/{env_uuid}` | `coolify_environment_variable` | v0.1.0 |
| `DELETE /services/{uuid}/scheduled-tasks/{task_uuid}` | `coolify_scheduled_task` | v0.2.0 |
| `DELETE /services/{uuid}/storages/{storage_uuid}` | `coolify_storage` | v0.2.0 |
| `DELETE /services/{uuid}/storages/{storage_uuid}/backups` | `coolify_storage_backup` | v0.1.9 |
| `GET /applications` | `data.coolify_applications` | v0.1.0 |
| `GET /applications/{uuid}` | `data.coolify_application` | v0.1.0 |
| `GET /applications/{uuid}/envs` | `data.coolify_environment_variables` | v0.1.0 |
| `GET /applications/{uuid}/logs` | `data.coolify_application_logs` | v0.2.0 |
| `GET /applications/{uuid}/restart` | `coolify_deployment` | v0.1.0 |
| `GET /applications/{uuid}/scheduled-tasks` | `data.coolify_scheduled_tasks` | v0.2.0 |
| `GET /applications/{uuid}/scheduled-tasks/{task_uuid}/executions` | `data.coolify_task_executions` | v0.2.0 |
| `GET /applications/{uuid}/start` | `coolify_resource_action` | v0.3.0 |
| `GET /applications/{uuid}/stop` | `coolify_resource_action` | v0.3.0 |
| `GET /applications/{uuid}/storages` | `data.coolify_storages` | v0.2.0 |
| `GET /cloud-tokens` | `data.coolify_cloud_tokens` | v0.2.0 |
| `GET /cloud-tokens/{uuid}` | `data.coolify_cloud_token` | v0.2.0 |
| `GET /databases` | `data.coolify_databases` | v0.1.0 |
| `GET /databases/{uuid}` | `data.coolify_database` | v0.1.0 |
| `GET /databases/{uuid}/backups` | `coolify_database_backup` | v0.1.0 |
| `GET /databases/{uuid}/backups/{scheduled_backup_uuid}/executions` | `data.coolify_backup_executions` | v0.2.0 |
| `GET /databases/{uuid}/envs` | `data.coolify_environment_variables` | v0.2.0 |
| `GET /databases/{uuid}/restart` | `coolify_resource_action` | v0.3.0 |
| `GET /databases/{uuid}/start` | `coolify_resource_action` | v0.3.0 |
| `GET /databases/{uuid}/stop` | `coolify_resource_action` | v0.3.0 |
| `GET /databases/{uuid}/storages` | `data.coolify_storages` | v0.2.0 |
| `GET /deploy` | `client.Deploy` | v0.2.0 |
| `GET /deployments` | `data.coolify_deployments` | v0.2.0 |
| `GET /deployments/applications/{uuid}` | `data.coolify_deployments` | v0.2.0 |
| `GET /deployments/{uuid}` | `coolify_deployment` | v0.1.0 |
| `GET /destinations` | `data.coolify_destinations` | v0.2.0 |
| `GET /destinations/{uuid}` | `data.coolify_destination` | v0.2.0 |
| `GET /digitalocean/images` | `data.coolify_digitalocean_images` | v0.2.0 |
| `GET /digitalocean/regions` | `data.coolify_digitalocean_regions` | v0.2.0 |
| `GET /digitalocean/sizes` | `data.coolify_digitalocean_sizes` | v0.2.0 |
| `GET /digitalocean/ssh-keys` | `data.coolify_digitalocean_ssh_keys` | v0.2.0 |
| `GET /disable` | `client.DisableAPI` | v0.2.0 |
| `GET /enable` | `client.EnableAPI` | v0.2.0 |
| `GET /github-apps` | `data.coolify_github_apps` | v0.2.0 |
| `GET /github-apps/{github_app_id}/repositories` | `data.coolify_github_app_repositories` | v0.2.0 |
| `GET /github-apps/{github_app_id}/repositories/{owner}/{repo}/branches` | `data.coolify_github_app_branches` | v0.2.0 |
| `GET /health` | `data.coolify_health` | v0.2.0 |
| `GET /hetzner/images` | `data.coolify_hetzner_images` | v0.2.0 |
| `GET /hetzner/locations` | `data.coolify_hetzner_locations` | v0.2.0 |
| `GET /hetzner/server-types` | `data.coolify_hetzner_server_types` | v0.2.0 |
| `GET /hetzner/ssh-keys` | `data.coolify_hetzner_ssh_keys` | v0.2.0 |
| `GET /projects` | `data.coolify_projects` | v0.1.0 |
| `GET /projects/{uuid}` | `data.coolify_project` | v0.1.0 |
| `GET /projects/{uuid}/environments` | `data.coolify_environments` | v0.2.0 |
| `GET /projects/{uuid}/{environment_name_or_uuid}` | `data.coolify_environment` | v0.2.0 |
| `GET /resources` | `data.coolify_resources` | v0.2.0 |
| `GET /s3-storages` | `data.coolify_s3_storages` | v0.1.13 |
| `GET /s3-storages/{uuid}` | `data.coolify_s3_storage` | v0.1.13 |
| `GET /security/keys` | `data.coolify_private_keys` | v0.1.0 |
| `GET /security/keys/{uuid}` | `data.coolify_private_key` | v0.1.0 |
| `GET /servers` | `data.coolify_servers` | v0.1.0 |
| `GET /servers/{server_uuid}/destinations` | `data.coolify_destinations` | v0.2.0 |
| `GET /servers/{uuid}` | `data.coolify_server` | v0.1.0 |
| `GET /servers/{uuid}/domains` | `data.coolify_server_domains` | v0.1.0 |
| `GET /servers/{uuid}/resources` | `data.coolify_server_resources` | v0.1.0 |
| `GET /servers/{uuid}/validate` | `data.coolify_server_validation` | v0.2.0 |
| `GET /services` | `data.coolify_services` | v0.1.0 |
| `GET /services/{uuid}` | `data.coolify_service` | v0.1.0 |
| `GET /services/{uuid}/envs` | `data.coolify_environment_variables` | v0.2.0 |
| `GET /services/{uuid}/restart` | `coolify_resource_action` | v0.3.0 |
| `GET /services/{uuid}/scheduled-tasks` | `data.coolify_scheduled_tasks` | v0.2.0 |
| `GET /services/{uuid}/scheduled-tasks/{task_uuid}/executions` | `data.coolify_task_executions` | v0.2.0 |
| `GET /services/{uuid}/start` | `coolify_resource_action` | v0.3.0 |
| `GET /services/{uuid}/stop` | `coolify_resource_action` | v0.3.0 |
| `GET /services/{uuid}/storages` | `data.coolify_storages` | v0.2.0 |
| `GET /teams` | `data.coolify_teams` | v0.2.0 |
| `GET /teams/current` | `data.coolify_team_members` | v0.2.0 |
| `GET /teams/current/members` | `data.coolify_team_members` | v0.2.0 |
| `GET /teams/{id}` | `data.coolify_team` | v0.1.0 |
| `GET /teams/{id}/members` | `data.coolify_team_members` | v0.2.0 |
| `GET /version` | `data.coolify_version` | v0.1.0 |
| `GET /vultr/os` | `data.coolify_vultr_os` | v0.2.0 |
| `GET /vultr/plans` | `data.coolify_vultr_plans` | v0.2.0 |
| `GET /vultr/regions` | `data.coolify_vultr_regions` | v0.2.0 |
| `GET /vultr/ssh-keys` | `data.coolify_vultr_ssh_keys` | v0.2.0 |
| `PATCH /applications/{uuid}` | `coolify_application + variants` | v0.1.0 |
| `PATCH /applications/{uuid}/envs` | `coolify_environment_variable` | v0.1.0 |
| `PATCH /applications/{uuid}/envs/bulk` | `client.BulkUpdateEnvVars` | v0.2.0 |
| `PATCH /applications/{uuid}/scheduled-tasks/{task_uuid}` | `coolify_scheduled_task` | v0.2.0 |
| `PATCH /applications/{uuid}/storages` | `coolify_storage` | v0.2.0 |
| `PATCH /cloud-tokens/{uuid}` | `coolify_cloud_token` | v0.2.0 |
| `PATCH /databases/{uuid}` | `coolify_*_database` | v0.1.0 |
| `PATCH /databases/{uuid}/backups/{scheduled_backup_uuid}` | `coolify_database_backup` | v0.1.0 |
| `PATCH /databases/{uuid}/envs` | `coolify_environment_variable` | v0.2.0 |
| `PATCH /databases/{uuid}/envs/bulk` | `client.BulkUpdateEnvVars` | v0.2.0 |
| `PATCH /databases/{uuid}/storages` | `coolify_storage` | v0.2.0 |
| `PATCH /github-apps/{github_app_id}` | `coolify_github_app` | v0.2.0 |
| `PATCH /projects/{uuid}` | `coolify_project` | v0.1.0 |
| `PATCH /s3-storages/{uuid}` | `coolify_s3_storage` | v0.1.13 |
| `PATCH /security/keys/{uuid}` | `coolify_private_key` | v0.1.0 |
| `PATCH /servers/{uuid}` | `coolify_server` | v0.1.0 |
| `PATCH /services/{uuid}` | `coolify_service` | v0.1.0 |
| `PATCH /services/{uuid}/envs` | `coolify_environment_variable` | v0.1.0 |
| `PATCH /services/{uuid}/envs/bulk` | `client.BulkUpdateEnvVars` | v0.2.0 |
| `PATCH /services/{uuid}/scheduled-tasks/{task_uuid}` | `coolify_scheduled_task` | v0.2.0 |
| `PATCH /services/{uuid}/storages` | `coolify_storage` | v0.2.0 |
| `POST /applications/dockerfile` | `coolify_application_dockerfile` | v0.2.0 |
| `POST /applications/dockerimage` | `coolify_application_docker_image` | v0.1.0 |
| `POST /applications/private-deploy-key` | `coolify_application_private_git` | v0.1.0 |
| `POST /applications/private-github-app` | `coolify_application_github_app` | v0.2.0 |
| `POST /applications/public` | `coolify_application` | v0.1.0 |
| `POST /applications/{uuid}/envs` | `coolify_environment_variable` | v0.1.0 |
| `POST /applications/{uuid}/restart` | `coolify_resource_action` | v0.3.0 |
| `POST /applications/{uuid}/scheduled-tasks` | `coolify_scheduled_task` | v0.2.0 |
| `POST /applications/{uuid}/start` | `coolify_resource_action` | v0.3.0 |
| `POST /applications/{uuid}/stop` | `coolify_resource_action` | v0.3.0 |
| `POST /applications/{uuid}/storages` | `coolify_storage` | v0.2.0 |
| `POST /cloud-tokens` | `coolify_cloud_token` | v0.2.0 |
| `POST /cloud-tokens/{uuid}/validate` | `client.ValidateCloudToken` | v0.2.0 |
| `POST /databases/clickhouse` | `coolify_database_clickhouse` | v0.1.0 |
| `POST /databases/dragonfly` | `coolify_database_dragonfly` | v0.1.0 |
| `POST /databases/keydb` | `coolify_database_keydb` | v0.1.0 |
| `POST /databases/mariadb` | `coolify_database_mariadb` | v0.1.0 |
| `POST /databases/mongodb` | `coolify_database_mongodb` | v0.1.0 |
| `POST /databases/mysql` | `coolify_database_mysql` | v0.1.0 |
| `POST /databases/postgresql` | `coolify_database_postgresql` | v0.1.0 |
| `POST /databases/redis` | `coolify_database_redis` | v0.1.0 |
| `POST /databases/{uuid}/backups` | `coolify_database_backup` | v0.1.0 |
| `POST /databases/{uuid}/envs` | `coolify_environment_variable` | v0.2.0 |
| `POST /databases/{uuid}/restart` | `coolify_resource_action` | v0.3.0 |
| `POST /databases/{uuid}/start` | `coolify_resource_action` | v0.3.0 |
| `POST /databases/{uuid}/stop` | `coolify_resource_action` | v0.3.0 |
| `POST /databases/{uuid}/storages` | `coolify_storage` | v0.2.0 |
| `POST /deploy` | `client.Deploy` | v0.2.0 |
| `POST /deployments/{uuid}/cancel` | `client.CancelDeployment` | v0.2.0 |
| `POST /disable` | `coolify_api_settings` | v0.2.0 |
| `POST /enable` | `coolify_api_settings` | v0.2.0 |
| `POST /github-apps` | `coolify_github_app` | v0.2.0 |
| `POST /mcp/disable` | `coolify_api_settings (mcp_enabled)` | v0.4.0 |
| `POST /mcp/enable` | `coolify_api_settings (mcp_enabled)` | v0.4.0 |
| `POST /projects` | `coolify_project` | v0.1.0 |
| `POST /projects/{uuid}/environments` | `coolify_environment` | v0.2.0 |
| `POST /s3-storages` | `coolify_s3_storage` | v0.1.13 |
| `POST /s3-storages/{uuid}/validate` | `coolify_s3_storage_validate` | v0.1.14 |
| `POST /security/keys` | `coolify_private_key` | v0.1.0 |
| `POST /servers` | `coolify_server` | v0.1.0 |
| `POST /servers/digitalocean` | `coolify_server_digitalocean` | v0.2.0 |
| `POST /servers/hetzner` | `coolify_server_hetzner` | v0.2.0 |
| `POST /servers/vultr` | `coolify_server_vultr` | v0.2.0 |
| `POST /servers/{server_uuid}/destinations` | `coolify_destination` | v0.2.0 |
| `POST /servers/{uuid}/validate` | `coolify_server_validate` | v0.2.0 |
| `POST /services` | `coolify_service` | v0.1.0 |
| `POST /services/{uuid}/envs` | `coolify_environment_variable` | v0.1.0 |
| `POST /services/{uuid}/restart` | `coolify_resource_action` | v0.3.0 |
| `POST /services/{uuid}/scheduled-tasks` | `coolify_scheduled_task` | v0.2.0 |
| `POST /services/{uuid}/start` | `coolify_resource_action` | v0.3.0 |
| `POST /services/{uuid}/stop` | `coolify_resource_action` | v0.3.0 |
| `POST /services/{uuid}/storages` | `coolify_storage` | v0.2.0 |
| `PUT /applications/{uuid}/storages/{storage_uuid}/backups` | `coolify_storage_backup` | v0.1.9 |
| `PUT /databases/{uuid}/storages/{storage_uuid}/backups` | `coolify_storage_backup` | v0.1.9 |
| `PUT /services/{uuid}/storages/{storage_uuid}/backups` | `coolify_storage_backup` | v0.1.9 |

## Planned

Ordered by priority (1 = most needed by users).

| Priority | Endpoint | Notes |
|----------|----------|-------|
| 2 | `GET /notifications/discord` | Notification channel resources (#394) |
| 2 | `GET /notifications/email` | Notification channel resources (#394) |
| 2 | `GET /notifications/pushover` | Notification channel resources (#394) |
| 2 | `GET /notifications/slack` | Notification channel resources (#394) |
| 2 | `GET /notifications/telegram` | Notification channel resources (#394) |
| 2 | `GET /notifications/webhook` | Notification channel resources (#394) |
| 2 | `PATCH /notifications/discord` | Notification channel resources (#394) |
| 2 | `PATCH /notifications/email` | Notification channel resources (#394) |
| 2 | `PATCH /notifications/pushover` | Notification channel resources (#394) |
| 2 | `PATCH /notifications/slack` | Notification channel resources (#394) |
| 2 | `PATCH /notifications/telegram` | Notification channel resources (#394) |
| 2 | `PATCH /notifications/webhook` | Notification channel resources (#394) |

## Intentionally Skipped

These endpoints are intentionally not modeled directly in Terraform.

| Endpoint | Reason |
|----------|--------|
| `DELETE /applications/{uuid}/destinations/{destination_uuid}` | Application multi-destination attach/detach; primary TF surface is coolify_destination |
| `DELETE /applications/{uuid}/tags/{tag_uuid}` | Resource tags API; not modeled as first-class TF resources |
| `DELETE /cloud-init-scripts/{uuid}` | Cloud-init script library; not a TF resource |
| `DELETE /databases/{uuid}/tags/{tag_uuid}` | Resource tags API; not modeled as first-class TF resources |
| `DELETE /gitlab-apps/{gitlab_app_id}` | GitLab App integration; provider covers GitHub Apps only today |
| `DELETE /projects/{uuid}/environments/{environment_name_or_uuid}/envs/{env_id}` | Shared/project/server/team env vars; provider models resource-scoped envs (app/db/service) |
| `DELETE /projects/{uuid}/envs/{env_id}` | Shared/project/server/team env vars; provider models resource-scoped envs (app/db/service) |
| `DELETE /servers/{uuid}/envs/{env_id}` | Shared/project/server/team env vars; provider models resource-scoped envs (app/db/service) |
| `DELETE /services/{uuid}/tags/{tag_uuid}` | Resource tags API; not modeled as first-class TF resources |
| `DELETE /tags/{uuid}` | Resource tags API; not modeled as first-class TF resources |
| `DELETE /team/envs/{env_id}` | Shared/project/server/team env vars; provider models resource-scoped envs (app/db/service) |
| `GET /applications/{uuid}/destinations` | Application multi-destination attach/detach; primary TF surface is coolify_destination |
| `GET /applications/{uuid}/rollback-images` | Application rollback/images; operational, not TF lifecycle |
| `GET /applications/{uuid}/tags` | Resource tags API; not modeled as first-class TF resources |
| `GET /cloud-init-scripts` | Cloud-init script library; not a TF resource |
| `GET /cloud-init-scripts/{uuid}` | Cloud-init script library; not a TF resource |
| `GET /databases/{uuid}/logs` | Resource logs streaming; not durable TF state |
| `GET /databases/{uuid}/tags` | Resource tags API; not modeled as first-class TF resources |
| `GET /gitlab-apps` | GitLab App integration; provider covers GitHub Apps only today |
| `GET /hetzner/firewalls` | Hetzner firewalls/networks list; not required for coolify_server_hetzner |
| `GET /hetzner/networks` | Hetzner firewalls/networks list; not required for coolify_server_hetzner |
| `GET /projects/{uuid}/environments/{environment_name_or_uuid}/envs` | Shared/project/server/team env vars; provider models resource-scoped envs (app/db/service) |
| `GET /projects/{uuid}/envs` | Shared/project/server/team env vars; provider models resource-scoped envs (app/db/service) |
| `GET /servers/{uuid}/cloudflare-tunnel` | Server operational/control-plane API; not modeled as TF resources |
| `GET /servers/{uuid}/docker-cleanup` | Server operational/control-plane API; not modeled as TF resources |
| `GET /servers/{uuid}/docker-cleanup/executions` | Server operational/control-plane API; not modeled as TF resources |
| `GET /servers/{uuid}/envs` | Shared/project/server/team env vars; provider models resource-scoped envs (app/db/service) |
| `GET /servers/{uuid}/export` | Server operational/control-plane API; not modeled as TF resources |
| `GET /servers/{uuid}/log-drains` | Server operational/control-plane API; not modeled as TF resources |
| `GET /servers/{uuid}/proxy` | Server operational/control-plane API; not modeled as TF resources |
| `GET /servers/{uuid}/sentinel` | Server operational/control-plane API; not modeled as TF resources |
| `GET /services/{uuid}/applications` | Nested service components; coolify_service manages the compose service unit |
| `GET /services/{uuid}/applications/{app_uuid}` | Nested service components; coolify_service manages the compose service unit |
| `GET /services/{uuid}/applications/{app_uuid}/logs` | Resource logs streaming; not durable TF state |
| `GET /services/{uuid}/applications/{app_uuid}/restart` | Nested service components; coolify_service manages the compose service unit |
| `GET /services/{uuid}/applications/{app_uuid}/start` | Nested service components; coolify_service manages the compose service unit |
| `GET /services/{uuid}/applications/{app_uuid}/stop` | Nested service components; coolify_service manages the compose service unit |
| `GET /services/{uuid}/databases` | Nested service components; coolify_service manages the compose service unit |
| `GET /services/{uuid}/databases/{database_uuid}` | Nested service components; coolify_service manages the compose service unit |
| `GET /services/{uuid}/databases/{database_uuid}/logs` | Resource logs streaming; not durable TF state |
| `GET /services/{uuid}/logs` | Resource logs streaming; not durable TF state |
| `GET /services/{uuid}/tags` | Resource tags API; not modeled as first-class TF resources |
| `GET /tags` | Resource tags API; not modeled as first-class TF resources |
| `GET /team` | Alias of /teams/current*; use data.coolify_team / team_members |
| `GET /team/envs` | Shared/project/server/team env vars; provider models resource-scoped envs (app/db/service) |
| `GET /team/members` | Alias of /teams/current*; use data.coolify_team / team_members |
| `PATCH /cloud-init-scripts/{uuid}` | Cloud-init script library; not a TF resource |
| `PATCH /destinations/{uuid}` | Destination fields RequireReplace; provider does not call PATCH |
| `PATCH /gitlab-apps/{gitlab_app_id}` | GitLab App integration; provider covers GitHub Apps only today |
| `PATCH /projects/{uuid}/environments/{environment_name_or_uuid}` | Environment description is TF state-only today; client has no Update call |
| `PATCH /projects/{uuid}/environments/{environment_name_or_uuid}/envs/{env_id}` | Shared/project/server/team env vars; provider models resource-scoped envs (app/db/service) |
| `PATCH /projects/{uuid}/envs/{env_id}` | Shared/project/server/team env vars; provider models resource-scoped envs (app/db/service) |
| `PATCH /servers/{uuid}/cloudflare-tunnel` | Server operational/control-plane API; not modeled as TF resources |
| `PATCH /servers/{uuid}/docker-cleanup` | Server operational/control-plane API; not modeled as TF resources |
| `PATCH /servers/{uuid}/envs/{env_id}` | Shared/project/server/team env vars; provider models resource-scoped envs (app/db/service) |
| `PATCH /servers/{uuid}/log-drains` | Server operational/control-plane API; not modeled as TF resources |
| `PATCH /servers/{uuid}/proxy` | Server operational/control-plane API; not modeled as TF resources |
| `PATCH /servers/{uuid}/sentinel` | Server operational/control-plane API; not modeled as TF resources |
| `PATCH /services/{uuid}/applications/{app_uuid}` | Nested service components; coolify_service manages the compose service unit |
| `PATCH /services/{uuid}/databases/{database_uuid}` | Nested service components; coolify_service manages the compose service unit |
| `PATCH /tags/{uuid}` | Resource tags API; not modeled as first-class TF resources |
| `PATCH /team/envs/{env_id}` | Shared/project/server/team env vars; provider models resource-scoped envs (app/db/service) |
| `POST /applications/dockercompose` | Deprecated alias: use POST /services instead because this flow creates a Service, not an Application |
| `POST /applications/{uuid}/clone` | One-shot clone/migrate/move; not Terraform lifecycle |
| `POST /applications/{uuid}/destinations` | Application multi-destination attach/detach; primary TF surface is coolify_destination |
| `POST /applications/{uuid}/migrate` | One-shot clone/migrate/move; not Terraform lifecycle |
| `POST /applications/{uuid}/move` | One-shot clone/migrate/move; not Terraform lifecycle |
| `POST /applications/{uuid}/rollback` | Application rollback/images; operational, not TF lifecycle |
| `POST /applications/{uuid}/scheduled-tasks/{task_uuid}/execute` | One-shot task execute; coolify_scheduled_task manages definition only |
| `POST /applications/{uuid}/storages/{storage_uuid}/backups/run` | One-shot volume backup run; coolify_storage_backup manages schedule only |
| `POST /applications/{uuid}/tags` | Resource tags API; not modeled as first-class TF resources |
| `POST /cloud-init-scripts` | Cloud-init script library; not a TF resource |
| `POST /databases/{uuid}/clone` | One-shot clone/migrate/move; not Terraform lifecycle |
| `POST /databases/{uuid}/migrate` | One-shot clone/migrate/move; not Terraform lifecycle |
| `POST /databases/{uuid}/move` | One-shot clone/migrate/move; not Terraform lifecycle |
| `POST /databases/{uuid}/storages/{storage_uuid}/backups/run` | One-shot volume backup run; coolify_storage_backup manages schedule only |
| `POST /databases/{uuid}/tags` | Resource tags API; not modeled as first-class TF resources |
| `POST /feedback` | Coolify product feedback endpoint; not TF |
| `POST /gitlab-apps` | GitLab App integration; provider covers GitHub Apps only today |
| `POST /projects/{uuid}/environments/{environment_name_or_uuid}/envs` | Shared/project/server/team env vars; provider models resource-scoped envs (app/db/service) |
| `POST /projects/{uuid}/envs` | Shared/project/server/team env vars; provider models resource-scoped envs (app/db/service) |
| `POST /sentinel/push` | Server operational/control-plane API; not modeled as TF resources |
| `POST /servers/import` | Server operational/control-plane API; not modeled as TF resources |
| `POST /servers/{uuid}/claim` | Server operational/control-plane API; not modeled as TF resources |
| `POST /servers/{uuid}/cloudflare-tunnel/disable` | Server operational/control-plane API; not modeled as TF resources |
| `POST /servers/{uuid}/cloudflare-tunnel/enable` | Server operational/control-plane API; not modeled as TF resources |
| `POST /servers/{uuid}/docker-cleanup/run` | Server operational/control-plane API; not modeled as TF resources |
| `POST /servers/{uuid}/envs` | Shared/project/server/team env vars; provider models resource-scoped envs (app/db/service) |
| `POST /servers/{uuid}/export/mailbox` | Server operational/control-plane API; not modeled as TF resources |
| `POST /servers/{uuid}/migrate` | One-shot clone/migrate/move; not Terraform lifecycle |
| `POST /servers/{uuid}/proxy/restart` | Server operational/control-plane API; not modeled as TF resources |
| `POST /servers/{uuid}/transfer/complete` | Server operational/control-plane API; not modeled as TF resources |
| `POST /services/{uuid}/applications/{app_uuid}/logs` | Resource logs streaming; not durable TF state |
| `POST /services/{uuid}/applications/{app_uuid}/restart` | Nested service app/DB lifecycle; manage via parent coolify_service |
| `POST /services/{uuid}/applications/{app_uuid}/start` | Nested service app/DB lifecycle; manage via parent coolify_service |
| `POST /services/{uuid}/applications/{app_uuid}/stop` | Nested service app/DB lifecycle; manage via parent coolify_service |
| `POST /services/{uuid}/clone` | One-shot clone/migrate/move; not Terraform lifecycle |
| `POST /services/{uuid}/databases/{database_uuid}/restart` | Nested service app/DB lifecycle; manage via parent coolify_service |
| `POST /services/{uuid}/databases/{database_uuid}/start` | Nested service app/DB lifecycle; manage via parent coolify_service |
| `POST /services/{uuid}/databases/{database_uuid}/stop` | Nested service app/DB lifecycle; manage via parent coolify_service |
| `POST /services/{uuid}/migrate` | One-shot clone/migrate/move; not Terraform lifecycle |
| `POST /services/{uuid}/move` | One-shot clone/migrate/move; not Terraform lifecycle |
| `POST /services/{uuid}/scheduled-tasks/{task_uuid}/execute` | One-shot task execute; coolify_scheduled_task manages definition only |
| `POST /services/{uuid}/storages/{storage_uuid}/backups/run` | One-shot volume backup run; coolify_storage_backup manages schedule only |
| `POST /services/{uuid}/tags` | Resource tags API; not modeled as first-class TF resources |
| `POST /tags` | Resource tags API; not modeled as first-class TF resources |
| `POST /team/envs` | Shared/project/server/team env vars; provider models resource-scoped envs (app/db/service) |
| `PUT /servers/{uuid}/proxy/configuration` | Server operational/control-plane API; not modeled as TF resources |

## Unclassified contract routes

_None. All pin contract routes are classified in `coveredEndpoints()`._

When `make contract-extract` adds routes, classify them in
`internal/spectest/coverage_test.go` or `TestSpecCoverage_Completeness` fails.
