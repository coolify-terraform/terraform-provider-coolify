# API Coverage

<!-- Auto-generated from internal/spectest/coverage_test.go. Do not edit manually. -->
<!-- Run: make api-coverage -->

**Route inventory**: pinned OpenAPI spec in `testdata/specs/coolify-v4.json`  
**Field source of truth**: source-derived contract in `testdata/contracts/coolify-v4.json`  
**Coverage**: 136 / 137 endpoints (99.3%)  
**Planned**: 0 | **Skipped**: 1

## Covered

| Endpoint | Terraform Resource / Data Source | Since |
|----------|----------------------------------|-------|
| `DELETE /applications/{uuid}` | `coolify_application + variants` | v0.1.0 |
| `DELETE /applications/{uuid}/envs/{env_uuid}` | `coolify_environment_variable` | v0.1.0 |
| `DELETE /applications/{uuid}/previews/{pull_request_id}` | `client.DeletePreviewDeployment` | v0.2.0 |
| `DELETE /applications/{uuid}/scheduled-tasks/{task_uuid}` | `coolify_scheduled_task` | v0.2.0 |
| `DELETE /applications/{uuid}/storages/{storage_uuid}` | `coolify_storage` | v0.2.0 |
| `DELETE /cloud-tokens/{uuid}` | `coolify_cloud_token` | v0.2.0 |
| `DELETE /databases/{uuid}` | `coolify_*_database` | v0.1.0 |
| `DELETE /databases/{uuid}/backups/{scheduled_backup_uuid}` | `coolify_database_backup` | v0.1.0 |
| `DELETE /databases/{uuid}/backups/{scheduled_backup_uuid}/executions/{execution_uuid}` | `client.DeleteBackupExecution` | v0.2.0 |
| `DELETE /databases/{uuid}/envs/{env_uuid}` | `coolify_environment_variable` | v0.2.0 |
| `DELETE /databases/{uuid}/storages/{storage_uuid}` | `coolify_storage` | v0.2.0 |
| `DELETE /github-apps/{github_app_id}` | `coolify_github_app` | v0.2.0 |
| `DELETE /projects/{uuid}` | `coolify_project` | v0.1.0 |
| `DELETE /projects/{uuid}/environments/{environment_name_or_uuid}` | `coolify_environment` | v0.2.0 |
| `DELETE /security/keys/{uuid}` | `coolify_private_key` | v0.1.0 |
| `DELETE /servers/{uuid}` | `coolify_server` | v0.1.0 |
| `DELETE /services/{uuid}` | `coolify_service` | v0.1.0 |
| `DELETE /services/{uuid}/envs/{env_uuid}` | `coolify_environment_variable` | v0.1.0 |
| `DELETE /services/{uuid}/scheduled-tasks/{task_uuid}` | `coolify_scheduled_task` | v0.2.0 |
| `DELETE /services/{uuid}/storages/{storage_uuid}` | `coolify_storage` | v0.2.0 |
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
| `GET /security/keys` | `data.coolify_private_keys` | v0.1.0 |
| `GET /security/keys/{uuid}` | `data.coolify_private_key` | v0.1.0 |
| `GET /servers` | `data.coolify_servers` | v0.1.0 |
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
| `POST /applications/{uuid}/scheduled-tasks` | `coolify_scheduled_task` | v0.2.0 |
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
| `POST /databases/{uuid}/storages` | `coolify_storage` | v0.2.0 |
| `POST /deployments/{uuid}/cancel` | `client.CancelDeployment` | v0.2.0 |
| `POST /github-apps` | `coolify_github_app` | v0.2.0 |
| `POST /mcp/disable` | `coolify_api_settings (mcp_enabled)` | v0.4.0 |
| `POST /mcp/enable` | `coolify_api_settings (mcp_enabled)` | v0.4.0 |
| `POST /projects` | `coolify_project` | v0.1.0 |
| `POST /projects/{uuid}/environments` | `coolify_environment` | v0.2.0 |
| `POST /security/keys` | `coolify_private_key` | v0.1.0 |
| `POST /servers` | `coolify_server` | v0.1.0 |
| `POST /servers/hetzner` | `coolify_server_hetzner` | v0.2.0 |
| `POST /services` | `coolify_service` | v0.1.0 |
| `POST /services/{uuid}/envs` | `coolify_environment_variable` | v0.1.0 |
| `POST /services/{uuid}/scheduled-tasks` | `coolify_scheduled_task` | v0.2.0 |
| `POST /services/{uuid}/storages` | `coolify_storage` | v0.2.0 |

## Planned

Ordered by priority (1 = most needed by users).

| Priority | Endpoint | Notes |
|----------|----------|-------|

## Intentionally Skipped

These endpoints are intentionally not modeled directly in Terraform.

| Endpoint | Reason |
|----------|--------|
| `POST /applications/dockercompose` | Deprecated alias: use POST /services instead because this flow creates a Service, not an Application |

## New in Spec (Unclassified)

_None. All spec endpoints are classified._

This section appears when the pinned spec is updated with new endpoints
that haven't been added to the coverage registry yet. The
`TestSpecCoverage_Completeness` test also fails in this case.
