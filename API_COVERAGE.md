# API Coverage

<!-- Auto-generated from internal/spectest/coverage_test.go. Do not edit manually. -->
<!-- Run: make api-coverage -->

**Spec**: Coolify v4 (pinned in `testdata/specs/coolify-v4.json`)  
**Coverage**: 57 / 135 endpoints (42.2%)  
**Planned**: 65 | **Skipped**: 13

## Covered

| Endpoint | Terraform Resource / Data Source | Since |
|----------|----------------------------------|-------|
| `DELETE /applications/{uuid}` | `coolify_application + variants` | v0.1.0 |
| `DELETE /applications/{uuid}/envs/{env_uuid}` | `coolify_environment_variable` | v0.1.0 |
| `DELETE /databases/{uuid}` | `coolify_*_database` | v0.1.0 |
| `DELETE /databases/{uuid}/backups/{scheduled_backup_uuid}` | `coolify_database_backup` | v0.1.0 |
| `DELETE /projects/{uuid}` | `coolify_project` | v0.1.0 |
| `DELETE /security/keys/{uuid}` | `coolify_private_key` | v0.1.0 |
| `DELETE /servers/{uuid}` | `coolify_server` | v0.1.0 |
| `DELETE /services/{uuid}` | `coolify_service` | v0.1.0 |
| `DELETE /services/{uuid}/envs/{env_uuid}` | `coolify_environment_variable` | v0.1.0 |
| `GET /applications` | `data.coolify_applications` | v0.1.0 |
| `GET /applications/{uuid}` | `data.coolify_application` | v0.1.0 |
| `GET /applications/{uuid}/envs` | `data.coolify_environment_variables` | v0.1.0 |
| `GET /applications/{uuid}/restart` | `coolify_deployment` | v0.1.0 |
| `GET /databases` | `data.coolify_databases` | v0.1.0 |
| `GET /databases/{uuid}` | `data.coolify_database` | v0.1.0 |
| `GET /databases/{uuid}/backups` | `coolify_database_backup` | v0.1.0 |
| `GET /deployments/{uuid}` | `coolify_deployment` | v0.1.0 |
| `GET /projects` | `data.coolify_projects` | v0.1.0 |
| `GET /projects/{uuid}` | `data.coolify_project` | v0.1.0 |
| `GET /security/keys` | `data.coolify_private_keys` | v0.1.0 |
| `GET /security/keys/{uuid}` | `data.coolify_private_key` | v0.1.0 |
| `GET /servers` | `data.coolify_servers` | v0.1.0 |
| `GET /servers/{uuid}` | `data.coolify_server` | v0.1.0 |
| `GET /servers/{uuid}/domains` | `data.coolify_server_domains` | v0.1.0 |
| `GET /servers/{uuid}/resources` | `data.coolify_server_resources` | v0.1.0 |
| `GET /services` | `data.coolify_services` | v0.1.0 |
| `GET /services/{uuid}` | `data.coolify_service` | v0.1.0 |
| `GET /teams/{id}` | `data.coolify_team` | v0.1.0 |
| `GET /version` | `data.coolify_version` | v0.1.0 |
| `PATCH /applications/{uuid}` | `coolify_application + variants` | v0.1.0 |
| `PATCH /applications/{uuid}/envs` | `coolify_environment_variable` | v0.1.0 |
| `PATCH /databases/{uuid}` | `coolify_*_database` | v0.1.0 |
| `PATCH /databases/{uuid}/backups/{scheduled_backup_uuid}` | `coolify_database_backup` | v0.1.0 |
| `PATCH /projects/{uuid}` | `coolify_project` | v0.1.0 |
| `PATCH /security/keys` | `coolify_private_key` | v0.1.0 |
| `PATCH /servers/{uuid}` | `coolify_server` | v0.1.0 |
| `PATCH /services/{uuid}` | `coolify_service` | v0.1.0 |
| `PATCH /services/{uuid}/envs` | `coolify_environment_variable` | v0.1.0 |
| `POST /applications/dockercompose` | `coolify_docker_compose_application` | v0.1.0 |
| `POST /applications/dockerimage` | `coolify_docker_image_application` | v0.1.0 |
| `POST /applications/private-deploy-key` | `coolify_private_git_application` | v0.1.0 |
| `POST /applications/public` | `coolify_application` | v0.1.0 |
| `POST /applications/{uuid}/envs` | `coolify_environment_variable` | v0.1.0 |
| `POST /databases/clickhouse` | `coolify_clickhouse_database` | v0.1.0 |
| `POST /databases/dragonfly` | `coolify_dragonfly_database` | v0.1.0 |
| `POST /databases/keydb` | `coolify_keydb_database` | v0.1.0 |
| `POST /databases/mariadb` | `coolify_mariadb_database` | v0.1.0 |
| `POST /databases/mongodb` | `coolify_mongodb_database` | v0.1.0 |
| `POST /databases/mysql` | `coolify_mysql_database` | v0.1.0 |
| `POST /databases/postgresql` | `coolify_postgresql_database` | v0.1.0 |
| `POST /databases/redis` | `coolify_redis_database` | v0.1.0 |
| `POST /databases/{uuid}/backups` | `coolify_database_backup` | v0.1.0 |
| `POST /projects` | `coolify_project` | v0.1.0 |
| `POST /security/keys` | `coolify_private_key` | v0.1.0 |
| `POST /servers` | `coolify_server` | v0.1.0 |
| `POST /services` | `coolify_service` | v0.1.0 |
| `POST /services/{uuid}/envs` | `coolify_environment_variable` | v0.1.0 |

## Planned

Ordered by priority (1 = most needed by users).

| Priority | Endpoint | Notes |
|----------|----------|-------|
| 1 | `DELETE /databases/{uuid}/envs/{env_uuid}` | Database env vars CRUD |
| 1 | `DELETE /projects/{uuid}/environments/{environment_name_or_uuid}` | Environment CRUD |
| 1 | `GET /databases/{uuid}/envs` | Database env vars; parity with application env vars |
| 1 | `GET /deployments` | Deployment list data source |
| 1 | `GET /deployments/applications/{uuid}` | Application deployment history |
| 1 | `GET /projects/{uuid}/environments` | Environment CRUD; core for multi-env Terraform workflows |
| 1 | `GET /projects/{uuid}/{environment_name_or_uuid}` | Environment details data source |
| 1 | `GET /services/{uuid}/envs` | Service env var listing; data source completeness |
| 1 | `PATCH /databases/{uuid}/envs` | Database env vars CRUD |
| 1 | `POST /applications/dockerfile` | Dockerfile app without git; common deployment pattern |
| 1 | `POST /applications/private-github-app` | GitHub App deployment; common in organizations |
| 1 | `POST /databases/{uuid}/envs` | Database env vars CRUD |
| 1 | `POST /projects/{uuid}/environments` | Environment CRUD |
| 2 | `DELETE /applications/{uuid}/scheduled-tasks/{task_uuid}` | Scheduled tasks CRUD |
| 2 | `DELETE /applications/{uuid}/storages/{storage_uuid}` | Persistent storage CRUD |
| 2 | `DELETE /databases/{uuid}/storages/{storage_uuid}` | Database persistent storage CRUD |
| 2 | `DELETE /services/{uuid}/scheduled-tasks/{task_uuid}` | Service scheduled tasks CRUD |
| 2 | `DELETE /services/{uuid}/storages/{storage_uuid}` | Service persistent storage CRUD |
| 2 | `GET /applications/{uuid}/scheduled-tasks` | Scheduled tasks CRUD |
| 2 | `GET /applications/{uuid}/start` | Application lifecycle; client method exists |
| 2 | `GET /applications/{uuid}/stop` | Application lifecycle; client method exists |
| 2 | `GET /applications/{uuid}/storages` | Persistent storage CRUD |
| 2 | `GET /databases/{uuid}/storages` | Database persistent storage CRUD |
| 2 | `GET /resources` | Global resource listing data source |
| 2 | `GET /services/{uuid}/scheduled-tasks` | Service scheduled tasks CRUD |
| 2 | `GET /services/{uuid}/storages` | Service persistent storage CRUD |
| 2 | `GET /teams` | Teams list data source |
| 2 | `GET /teams/current` | Current team data source |
| 2 | `GET /teams/current/members` | Current team members data source |
| 2 | `GET /teams/{id}/members` | Team members data source |
| 2 | `PATCH /applications/{uuid}/scheduled-tasks/{task_uuid}` | Scheduled tasks CRUD |
| 2 | `PATCH /applications/{uuid}/storages` | Persistent storage CRUD |
| 2 | `PATCH /databases/{uuid}/storages` | Database persistent storage CRUD |
| 2 | `PATCH /services/{uuid}/scheduled-tasks/{task_uuid}` | Service scheduled tasks CRUD |
| 2 | `PATCH /services/{uuid}/storages` | Service persistent storage CRUD |
| 2 | `POST /applications/{uuid}/scheduled-tasks` | Scheduled tasks CRUD |
| 2 | `POST /applications/{uuid}/storages` | Persistent storage CRUD |
| 2 | `POST /databases/{uuid}/storages` | Database persistent storage CRUD |
| 2 | `POST /deployments/{uuid}/cancel` | Cancel running deployment |
| 2 | `POST /servers/hetzner` | Hetzner cloud server provisioning |
| 2 | `POST /services/{uuid}/scheduled-tasks` | Service scheduled tasks CRUD |
| 2 | `POST /services/{uuid}/storages` | Service persistent storage CRUD |
| 3 | `DELETE /cloud-tokens/{uuid}` | Cloud provider token CRUD |
| 3 | `DELETE /databases/{uuid}/backups/{scheduled_backup_uuid}/executions/{execution_uuid}` | Delete individual backup execution |
| 3 | `DELETE /github-apps/{github_app_id}` | GitHub App integration CRUD |
| 3 | `GET /applications/{uuid}/scheduled-tasks/{task_uuid}/executions` | Read-only task execution history |
| 3 | `GET /cloud-tokens` | Cloud provider token CRUD |
| 3 | `GET /cloud-tokens/{uuid}` | Cloud provider token CRUD |
| 3 | `GET /databases/{uuid}/backups/{scheduled_backup_uuid}/executions` | Read-only backup execution history |
| 3 | `GET /github-apps` | GitHub App integration CRUD |
| 3 | `GET /github-apps/{github_app_id}/repositories` | Read-only GitHub repository lookup |
| 3 | `GET /github-apps/{github_app_id}/repositories/{owner}/{repo}/branches` | Read-only GitHub branch lookup |
| 3 | `GET /hetzner/images` | Hetzner cloud metadata lookup |
| 3 | `GET /hetzner/locations` | Hetzner cloud metadata lookup |
| 3 | `GET /hetzner/server-types` | Hetzner cloud metadata lookup |
| 3 | `GET /hetzner/ssh-keys` | Hetzner cloud metadata lookup |
| 3 | `GET /services/{uuid}/scheduled-tasks/{task_uuid}/executions` | Read-only task execution history |
| 3 | `PATCH /applications/{uuid}/envs/bulk` | Bulk env var update; convenience |
| 3 | `PATCH /cloud-tokens/{uuid}` | Cloud provider token CRUD |
| 3 | `PATCH /databases/{uuid}/envs/bulk` | Bulk env var update; convenience |
| 3 | `PATCH /github-apps/{github_app_id}` | GitHub App integration CRUD |
| 3 | `PATCH /services/{uuid}/envs/bulk` | Bulk env var update; convenience |
| 3 | `POST /cloud-tokens` | Cloud provider token CRUD |
| 3 | `POST /cloud-tokens/{uuid}/validate` | Cloud provider token validation |
| 3 | `POST /github-apps` | GitHub App integration CRUD |

## Intentionally Skipped

These endpoints are not appropriate for Terraform resource management.

| Endpoint | Reason |
|----------|--------|
| `DELETE /applications/{uuid}/previews/{pull_request_id}` | Preview deployment management, niche |
| `GET /applications/{uuid}/logs` | Streaming logs, not a Terraform resource |
| `GET /databases/{uuid}/restart` | Operational action, not a Terraform resource |
| `GET /databases/{uuid}/start` | Operational action, not a Terraform resource |
| `GET /databases/{uuid}/stop` | Operational action, not a Terraform resource |
| `GET /deploy` | Generic deploy trigger; use coolify_deployment resource |
| `GET /disable` | API lifecycle management, not a Terraform resource |
| `GET /enable` | API lifecycle management, not a Terraform resource |
| `GET /health` | Operational healthcheck, not a Terraform resource |
| `GET /servers/{uuid}/validate` | Operational validation, not a Terraform resource |
| `GET /services/{uuid}/restart` | Operational action, not a Terraform resource |
| `GET /services/{uuid}/start` | Operational action, not a Terraform resource |
| `GET /services/{uuid}/stop` | Operational action, not a Terraform resource |

## New in Spec (Unclassified)

_None. All spec endpoints are classified._

This section appears when the pinned spec is updated with new endpoints
that haven't been added to the coverage registry yet. The
`TestSpecCoverage_Completeness` test also fails in this case.
