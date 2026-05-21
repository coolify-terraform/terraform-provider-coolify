---
page_title: "API Contract Accuracy"
subcategory: "Guides"
description: "Comparison of Coolify OpenAPI spec vs real source code contract."
---

# API Contract Accuracy

This page compares the pinned reusable OpenAPI schemas with the source-derived
Coolify contract extracted from the real application code.

> The source-derived contract is the field-level source of truth. The pinned OpenAPI spec is useful for reusable public schemas and route inventory, but some contract models only exist as internal implementation details or inline request bodies.
> `reviewed drift` means the pinned spec and source contract disagree on nullability, but the provider already handles the field safely and no runtime fix is needed.
> `mapped` means the field name appears in the provider's internal client JSON structs. It does not guarantee Terraform schema exposure, read-after-write round trips, or full CRUD behavior.

Contract version: `v4.0.1` | Extracted from: `coollabsio/coolify@v4.0.1`

## Summary

| Metric | Count |
|--------|------:|
| Public schema fields compared | 299 |
| Public schema type matches | 299/299 |
| Public schema nullable matches | 269/299 |
| Public schema client JSON mappings | 143/299 |
| Reusable public schemas compared | 9 |
| Contract-only / inline-only models documented | 13 |

---

## Reusable Public Schemas

## Application

Fields: 133 | Type matches: 133/133 | Nullable matches: 120/133 | Client JSON mappings: 78/133

| Field | Contract Type | Spec Type | Type Match | Nullable Match | Default | Client JSON Mapping |
|-------|:---:|:---:|:---:|:---:|---------|:---:|
| additional_destinations | string | string | yes | **WRONG** | - | n/a |
| application_id | integer | integer | yes | yes | - | n/a |
| base_directory | string | string | yes | yes | / | mapped |
| build_command | string | string | yes | reviewed drift | - | mapped |
| build_pack | string | string | yes | yes | - | mapped |
| compose_parsing_version | string | string | yes | yes | - | n/a |
| config_hash | string | string | yes | **WRONG** | - | n/a |
| connect_to_docker_network | string | string | yes | yes | - | mapped |
| custom_docker_run_options | string | string | yes | yes | - | mapped |
| custom_healthcheck_found | string | string | yes | yes | - | n/a |
| custom_internal_name | string | string | yes | yes | - | n/a |
| custom_labels | string | string | yes | reviewed drift | - | mapped |
| custom_network_aliases | string | string | yes | reviewed drift | - | mapped |
| custom_nginx_configuration | string | string | yes | reviewed drift | - | mapped |
| description | string | string | yes | reviewed drift | - | mapped |
| destination_id | integer | integer | yes | **WRONG** | - | n/a |
| destination_type | string | string | yes | **WRONG** | - | n/a |
| disable_build_cache | string | string | yes | yes | - | n/a |
| docker_compose | string | string | yes | **WRONG** | - | n/a |
| docker_compose_custom_build_command | string | string | yes | reviewed drift | - | mapped |
| docker_compose_custom_start_command | string | string | yes | reviewed drift | - | mapped |
| docker_compose_domains | string | string | yes | reviewed drift | - | mapped |
| docker_compose_location | string | string | yes | yes | - | mapped |
| docker_compose_pr | string | string | yes | **WRONG** | - | n/a |
| docker_compose_pr_location | string | string | yes | **WRONG** | /docker-compose.yaml | n/a |
| docker_compose_pr_raw | string | string | yes | **WRONG** | - | n/a |
| docker_compose_raw | string | string | yes | reviewed drift | - | mapped |
| docker_images_to_keep | string | string | yes | yes | - | n/a |
| docker_registry_image_name | string | string | yes | yes | - | mapped |
| docker_registry_image_tag | string | string | yes | yes | - | mapped |
| dockerfile | string | string | yes | reviewed drift | - | mapped |
| dockerfile_location | string | string | yes | yes | - | mapped |
| dockerfile_target_build | string | string | yes | reviewed drift | - | mapped |
| environment_id | integer | integer | yes | yes | - | n/a |
| force_domain_override | string | string | yes | yes | - | mapped |
| domains | string | string | yes | yes | - | mapped |
| git_branch | string | string | yes | yes | - | mapped |
| git_commit_sha | string | string | yes | yes | HEAD | mapped |
| git_full_url | string | string | yes | yes | - | n/a |
| git_repository | string | string | yes | yes | - | mapped |
| gpu_count | string | string | yes | yes | - | n/a |
| gpu_device_ids | string | string | yes | yes | - | n/a |
| gpu_driver | string | string | yes | yes | - | n/a |
| gpu_options | string | string | yes | yes | - | n/a |
| health_check_command | string | string | yes | reviewed drift | - | mapped |
| health_check_enabled | boolean | boolean | yes | yes | true | mapped |
| health_check_host | string | string | yes | reviewed drift | localhost | mapped |
| health_check_interval | integer | integer | yes | yes | 5 | mapped |
| health_check_method | string | string | yes | yes | GET | mapped |
| health_check_path | string | string | yes | yes | / | mapped |
| health_check_port | string | string | yes | yes | - | mapped |
| health_check_response_text | string | string | yes | yes | - | mapped |
| health_check_retries | integer | integer | yes | yes | 10 | mapped |
| health_check_return_code | integer | integer | yes | yes | 200 | mapped |
| health_check_scheme | string | string | yes | yes | http | mapped |
| health_check_start_period | integer | integer | yes | yes | 5 | mapped |
| health_check_timeout | integer | integer | yes | yes | 5 | mapped |
| health_check_type | string | string | yes | yes | - | mapped |
| http_basic_auth_password | string | string | yes | reviewed drift | - | mapped |
| http_basic_auth_username | string | string | yes | reviewed drift | - | mapped |
| include_source_commit_in_build | string | string | yes | yes | - | n/a |
| inject_build_args_to_dockerfile | string | string | yes | yes | - | n/a |
| install_command | string | string | yes | reviewed drift | - | mapped |
| is_auto_deploy_enabled | boolean | boolean | yes | yes | true | mapped |
| is_build_server_enabled | string | string | yes | yes | - | n/a |
| is_consistent_container_name_enabled | string | string | yes | yes | - | n/a |
| is_container_label_escape_enabled | string | string | yes | yes | - | mapped |
| is_container_label_readonly_enabled | boolean | boolean | yes | yes | false | n/a |
| is_custom_ssl | boolean | boolean | yes | yes | false | n/a |
| is_debug_enabled | boolean | boolean | yes | yes | false | n/a |
| is_dual_cert | boolean | boolean | yes | yes | false | n/a |
| is_env_sorting_enabled | boolean | boolean | yes | yes | false | n/a |
| is_force_https_enabled | boolean | boolean | yes | yes | true | mapped |
| is_git_lfs_enabled | boolean | boolean | yes | yes | true | n/a |
| is_git_shallow_clone_enabled | string | string | yes | yes | - | n/a |
| is_git_submodules_enabled | boolean | boolean | yes | yes | true | n/a |
| is_gpu_enabled | string | string | yes | yes | - | n/a |
| is_gzip_enabled | string | string | yes | yes | - | n/a |
| is_http2 | boolean | boolean | yes | yes | false | n/a |
| is_http_basic_auth_enabled | string | string | yes | yes | - | mapped |
| is_include_timestamps | string | string | yes | yes | - | n/a |
| is_log_drain_enabled | string | string | yes | yes | - | n/a |
| is_pr_deployments_public_enabled | string | string | yes | yes | - | n/a |
| is_preserve_repository_enabled | string | string | yes | yes | - | mapped |
| is_preview_deployments_enabled | boolean | boolean | yes | yes | false | n/a |
| is_raw_compose_deployment_enabled | string | string | yes | yes | - | n/a |
| is_spa | string | string | yes | yes | - | mapped |
| is_static | boolean | boolean | yes | yes | false | mapped |
| is_stripprefix_enabled | string | string | yes | yes | - | n/a |
| is_swarm_only_worker_nodes | string | string | yes | yes | - | n/a |
| last_online_at | string | string | yes | yes | - | n/a |
| last_restart_at | string | string | yes | **WRONG** | - | n/a |
| last_restart_type | string | string | yes | **WRONG** | - | n/a |
| limits_cpu_shares | integer | integer | yes | yes | 1024 | mapped |
| limits_cpus | string | string | yes | yes | 0 | mapped |
| limits_cpuset | string | string | yes | yes | 0 | mapped |
| limits_memory | string | string | yes | yes | 0 | mapped |
| limits_memory_reservation | string | string | yes | yes | 0 | mapped |
| limits_memory_swap | string | string | yes | yes | 0 | mapped |
| limits_memory_swappiness | integer | integer | yes | yes | 60 | mapped |
| manual_webhook_secret_bitbucket | string | string | yes | reviewed drift | - | mapped |
| manual_webhook_secret_gitea | string | string | yes | reviewed drift | - | mapped |
| manual_webhook_secret_github | string | string | yes | reviewed drift | - | mapped |
| manual_webhook_secret_gitlab | string | string | yes | reviewed drift | - | mapped |
| name | string | string | yes | yes | - | mapped |
| nixpkgsarchive | string | string | yes | yes | - | n/a |
| ports_exposes | string | string | yes | yes | - | mapped |
| ports_mappings | string | string | yes | yes | - | mapped |
| post_deployment_command | string | string | yes | yes | - | mapped |
| post_deployment_command_container | string | string | yes | reviewed drift | - | mapped |
| pre_deployment_command | string | string | yes | yes | - | mapped |
| pre_deployment_command_container | string | string | yes | reviewed drift | - | mapped |
| preview_url_template | string | string | yes | yes | { {pr_id} }.{ {domain} } | mapped |
| private_key_id | integer | integer | yes | yes | - | n/a |
| publish_directory | string | string | yes | reviewed drift | - | mapped |
| redirect | string | string | yes | reviewed drift | - | mapped |
| repository_project_id | integer | integer | yes | yes | - | n/a |
| restart_count | integer | integer | yes | yes | 0 | n/a |
| source_id | integer | integer | yes | yes | - | n/a |
| source_type | string | string | yes | **WRONG** | - | n/a |
| start_command | string | string | yes | reviewed drift | - | mapped |
| static_image | string | string | yes | yes | nginx:alpine | mapped |
| status | string | string | yes | yes | exited | mapped |
| swarm_placement_constraints | string | string | yes | **WRONG** | - | n/a |
| swarm_replicas | string | string | yes | **WRONG** | - | n/a |
| use_build_secrets | string | string | yes | yes | - | n/a |
| use_build_server | string | string | yes | yes | - | mapped |
| uuid | string | string | yes | yes | - | mapped |
| watch_paths | string | string | yes | reviewed drift | - | mapped |
| created_at | - | string | - | - | - | mapped |
| deleted_at | - | string | - | - | - | n/a |
| id | - | integer | - | - | - | mapped |
| updated_at | - | string | - | - | - | mapped |

## Environment

Fields: 7 | Type matches: 7/7 | Nullable matches: 7/7 | Client JSON mappings: 6/7

| Field | Contract Type | Spec Type | Type Match | Nullable Match | Default | Client JSON Mapping |
|-------|:---:|:---:|:---:|:---:|---------|:---:|
| description | string | string | yes | yes | - | mapped |
| name | string | string | yes | yes | - | mapped |
| project_id | integer | integer | yes | yes | - | n/a |
| uuid | string | string | yes | yes | - | mapped |
| created_at | - | string | - | - | - | mapped |
| id | - | integer | - | - | - | mapped |
| updated_at | - | string | - | - | - | mapped |

## EnvironmentVariable

Fields: 33 | Type matches: 33/33 | Nullable matches: 21/33 | Client JSON mappings: 8/33

| Field | Contract Type | Spec Type | Type Match | Nullable Match | Default | Client JSON Mapping |
|-------|:---:|:---:|:---:|:---:|---------|:---:|
| application_id | integer | integer | yes | **WRONG** | - | n/a |
| comment | string | string | yes | **WRONG** | - | n/a |
| database_id | integer | integer | yes | **WRONG** | - | n/a |
| is_build_time | boolean | boolean | yes | yes | false | n/a |
| is_buildtime | string | string | yes | yes | true | mapped |
| is_buildtime_only | boolean | boolean | yes | yes | false | n/a |
| is_literal | string | string | yes | yes | - | n/a |
| is_multiline | string | string | yes | yes | - | n/a |
| is_preview | boolean | boolean | yes | yes | false | mapped |
| is_required | string | string | yes | yes | - | n/a |
| is_runtime | string | string | yes | yes | true | n/a |
| is_shared | string | string | yes | yes | - | n/a |
| is_shown_once | string | string | yes | yes | - | n/a |
| key | string | string | yes | yes | - | mapped |
| order | string | string | yes | yes | - | n/a |
| resourceable_id | string | string | yes | yes | - | n/a |
| resourceable_type | string | string | yes | yes | - | n/a |
| service_id | integer | integer | yes | **WRONG** | - | n/a |
| standalone_clickhouse_id | integer | integer | yes | **WRONG** | - | n/a |
| standalone_dragonfly_id | integer | integer | yes | **WRONG** | - | n/a |
| standalone_keydb_id | integer | integer | yes | **WRONG** | - | n/a |
| standalone_mariadb_id | integer | integer | yes | **WRONG** | - | n/a |
| standalone_mongodb_id | integer | integer | yes | **WRONG** | - | n/a |
| standalone_mysql_id | integer | integer | yes | **WRONG** | - | n/a |
| standalone_postgresql_id | integer | integer | yes | **WRONG** | - | n/a |
| standalone_redis_id | integer | integer | yes | **WRONG** | - | n/a |
| value | string | string | yes | reviewed drift | - | mapped |
| version | string | string | yes | yes | - | n/a |
| created_at | - | string | - | - | - | mapped |
| id | - | integer | - | - | - | mapped |
| real_value | - | string | - | - | - | n/a |
| updated_at | - | string | - | - | - | mapped |
| uuid | - | string | - | - | - | mapped |

## PrivateKey

Fields: 11 | Type matches: 11/11 | Nullable matches: 11/11 | Client JSON mappings: 10/11

| Field | Contract Type | Spec Type | Type Match | Nullable Match | Default | Client JSON Mapping |
|-------|:---:|:---:|:---:|:---:|---------|:---:|
| description | string | string | yes | reviewed drift | - | mapped |
| fingerprint | string | string | yes | yes | - | mapped |
| is_git_related | boolean | boolean | yes | yes | false | mapped |
| name | string | string | yes | yes | - | mapped |
| private_key | string | string | yes | yes | - | mapped |
| uuid | string | string | yes | yes | - | mapped |
| created_at | - | string | - | - | - | mapped |
| id | - | integer | - | - | - | mapped |
| public_key | - | string | - | - | - | mapped |
| team_id | - | integer | - | - | - | n/a |
| updated_at | - | string | - | - | - | mapped |

## Project

Fields: 4 | Type matches: 4/4 | Nullable matches: 4/4 | Client JSON mappings: 4/4

| Field | Contract Type | Spec Type | Type Match | Nullable Match | Default | Client JSON Mapping |
|-------|:---:|:---:|:---:|:---:|---------|:---:|
| description | string | string | yes | reviewed drift | - | mapped |
| name | string | string | yes | yes | - | mapped |
| uuid | string | string | yes | yes | - | mapped |
| id | - | integer | - | - | - | mapped |

## ScheduledTask

Fields: 12 | Type matches: 12/12 | Nullable matches: 10/12 | Client JSON mappings: 9/12

| Field | Contract Type | Spec Type | Type Match | Nullable Match | Default | Client JSON Mapping |
|-------|:---:|:---:|:---:|:---:|---------|:---:|
| application_id | integer | integer | yes | **WRONG** | - | n/a |
| command | string | string | yes | yes | - | mapped |
| container | string | string | yes | yes | - | n/a |
| enabled | boolean | boolean | yes | yes | true | mapped |
| frequency | string | string | yes | yes | - | mapped |
| name | string | string | yes | yes | - | mapped |
| service_id | integer | integer | yes | **WRONG** | - | n/a |
| timeout | string | string | yes | yes | - | mapped |
| uuid | string | string | yes | yes | - | mapped |
| created_at | - | string | - | - | - | mapped |
| id | - | integer | - | - | - | mapped |
| updated_at | - | string | - | - | - | mapped |

## Server

Fields: 26 | Type matches: 26/26 | Nullable matches: 26/26 | Client JSON mappings: 8/26

| Field | Contract Type | Spec Type | Type Match | Nullable Match | Default | Client JSON Mapping |
|-------|:---:|:---:|:---:|:---:|---------|:---:|
| cloud_provider_token_id | string | string | yes | yes | - | n/a |
| description | string | string | yes | reviewed drift | - | mapped |
| detected_traefik_version | string | string | yes | yes | - | n/a |
| hetzner_server_id | string | string | yes | yes | - | n/a |
| hetzner_server_status | string | string | yes | yes | - | n/a |
| ip | string | string | yes | yes | - | mapped |
| ip_previous | string | string | yes | yes | - | n/a |
| is_metrics_enabled | boolean | boolean | yes | yes | true | n/a |
| is_validating | string | string | yes | yes | - | n/a |
| name | string | string | yes | yes | - | mapped |
| port | integer | integer | yes | yes | 22 | mapped |
| private_key_id | integer | integer | yes | yes | - | n/a |
| server_metadata | string | string | yes | yes | - | n/a |
| traefik_outdated_info | string | string | yes | yes | - | n/a |
| user | string | string | yes | yes | root | mapped |
| uuid | string | string | yes | yes | - | mapped |
| high_disk_usage_notification_sent | - | boolean | - | - | - | n/a |
| id | - | integer | - | - | - | mapped |
| log_drain_notification_sent | - | boolean | - | - | - | n/a |
| proxy | - | object | - | - | - | n/a |
| proxy_type | - | string | - | - | - | n/a |
| settings | - |  | - | - | - | mapped |
| swarm_cluster | - | string | - | - | - | n/a |
| unreachable_count | - | integer | - | - | - | n/a |
| unreachable_notification_sent | - | boolean | - | - | - | n/a |
| validation_logs | - | string | - | - | - | n/a |

## ServerSetting

> For server resources, the Terraform write surface is intentionally smaller than the read surface. The provider only sends the shared `UpdateServerInput` fields on PATCH: `name`, `description`, `ip`, `port`, `user`, `private_key_uuid`, `is_build_server`, `concurrent_builds`, `dynamic_timeout`, `deployment_queue_limit`, `server_disk_usage_notification_threshold`, `server_disk_usage_check_frequency`, and `connection_timeout`. Other `ServerSetting` fields can still appear on GET responses without being valid Terraform inputs or public PATCH fields.

Fields: 54 | Type matches: 54/54 | Nullable matches: 53/54 | Client JSON mappings: 11/54

| Field | Contract Type | Spec Type | Type Match | Nullable Match | Default | Client JSON Mapping |
|-------|:---:|:---:|:---:|:---:|---------|:---:|
| cleanup_after_percentage | integer | integer | yes | yes | 80 | n/a |
| concurrent_builds | string | string | yes | yes | - | mapped |
| delete_unused_networks | string | string | yes | yes | - | n/a |
| delete_unused_volumes | string | string | yes | yes | - | n/a |
| deployment_queue_limit | string | string | yes | yes | - | mapped |
| disable_application_image_retention | string | string | yes | yes | - | n/a |
| docker_cleanup_frequency | string | string | yes | yes | */10 * * * * | n/a |
| docker_cleanup_threshold | string | string | yes | yes | - | n/a |
| dynamic_timeout | string | string | yes | yes | - | mapped |
| force_disabled | string | string | yes | yes | - | n/a |
| force_docker_cleanup | boolean | boolean | yes | yes | false | n/a |
| force_server_cleanup | boolean | boolean | yes | yes | false | n/a |
| generate_exact_labels | string | string | yes | yes | - | n/a |
| is_build_server | boolean | boolean | yes | yes | false | mapped |
| is_cloudflare_tunnel | string | string | yes | yes | - | n/a |
| is_force_cleanup_enabled | boolean | boolean | yes | yes | false | n/a |
| is_jump_server | boolean | boolean | yes | yes | false | n/a |
| is_logdrain_axiom_enabled | string | string | yes | yes | - | n/a |
| is_logdrain_custom_enabled | string | string | yes | yes | - | n/a |
| is_logdrain_highlight_enabled | string | string | yes | yes | - | n/a |
| is_logdrain_newrelic_enabled | string | string | yes | yes | - | n/a |
| is_metrics_enabled | string | string | yes | yes | - | n/a |
| is_part_of_swarm | boolean | boolean | yes | yes | false | n/a |
| is_reachable | boolean | boolean | yes | yes | false | mapped |
| is_sentinel_debug_enabled | string | string | yes | yes | - | n/a |
| is_sentinel_enabled | boolean | boolean | yes | yes | false | n/a |
| is_server_api_enabled | boolean | boolean | yes | yes | false | n/a |
| is_swarm_manager | string | string | yes | yes | - | n/a |
| is_swarm_worker | string | string | yes | yes | - | n/a |
| is_terminal_enabled | string | string | yes | yes | - | n/a |
| is_usable | boolean | boolean | yes | yes | false | mapped |
| logdrain_axiom_api_key | string | string | yes | yes | - | n/a |
| logdrain_axiom_dataset_name | string | string | yes | yes | - | n/a |
| logdrain_custom_config | string | string | yes | yes | - | n/a |
| logdrain_custom_config_parser | string | string | yes | yes | - | n/a |
| logdrain_highlight_project_id | string | string | yes | yes | - | n/a |
| logdrain_newrelic_base_uri | string | string | yes | yes | - | n/a |
| logdrain_newrelic_license_key | string | string | yes | yes | - | n/a |
| metrics_history_days | integer | integer | yes | yes | 30 | n/a |
| metrics_refresh_rate_seconds | integer | integer | yes | yes | 5 | n/a |
| metrics_token | string | string | yes | **WRONG** | - | n/a |
| sentinel_custom_url | string | string | yes | yes | - | n/a |
| sentinel_metrics_history_days | string | string | yes | yes | - | n/a |
| sentinel_metrics_refresh_rate_seconds | string | string | yes | yes | - | n/a |
| sentinel_push_interval_seconds | string | string | yes | yes | - | n/a |
| sentinel_token | string | string | yes | yes | - | n/a |
| server_disk_usage_check_frequency | string | string | yes | yes | - | mapped |
| server_disk_usage_notification_threshold | string | string | yes | yes | - | mapped |
| server_id | integer | integer | yes | yes | - | n/a |
| server_timezone | string | string | yes | yes |  | n/a |
| wildcard_domain | string | string | yes | yes | - | n/a |
| created_at | - | string | - | - | - | mapped |
| id | - | integer | - | - | - | mapped |
| updated_at | - | string | - | - | - | mapped |

## Service

Fields: 19 | Type matches: 19/19 | Nullable matches: 17/19 | Client JSON mappings: 9/19

| Field | Contract Type | Spec Type | Type Match | Nullable Match | Default | Client JSON Mapping |
|-------|:---:|:---:|:---:|:---:|---------|:---:|
| compose_parsing_version | string | string | yes | yes | - | n/a |
| config_hash | string | string | yes | yes | - | n/a |
| connect_to_docker_network | string | string | yes | yes | - | mapped |
| description | string | string | yes | yes | - | mapped |
| destination_id | integer | integer | yes | **WRONG** | - | n/a |
| destination_type | string | string | yes | **WRONG** | - | n/a |
| docker_compose | string | string | yes | yes | - | n/a |
| docker_compose_raw | string | string | yes | yes | - | mapped |
| environment_id | integer | integer | yes | yes | - | n/a |
| is_container_label_escape_enabled | string | string | yes | yes | - | mapped |
| name | string | string | yes | yes | - | mapped |
| server_id | string | string | yes | yes | - | n/a |
| service_type | string | string | yes | yes | - | n/a |
| uuid | string | string | yes | yes | - | mapped |
| created_at | - | string | - | - | - | mapped |
| deleted_at | - | string | - | - | - | n/a |
| id | - | integer | - | - | - | mapped |
| is_container_label_readonly_enabled | - | boolean | - | - | - | n/a |
| updated_at | - | string | - | - | - | mapped |

## Contract-Only or Inline-Only Models

These sections document source-derived models that do not map cleanly to reusable public OpenAPI component schemas.

## ScheduledDatabaseBackup

This section compares the internal source-derived backup model against the public backup request bodies in the pinned spec.
Coolify stores the relation as `s3_storage_id` internally, while the public API accepts `s3_storage_uuid` on request bodies.
That identifier translation is expected and does not imply a missing top-level S3 CRUD API.

Fields: 19 | Type matches: 19/19 | Nullable matches: 19/19 | Client JSON mappings: 15/19

| Field | Contract Type | Spec Type | Type Match | Nullable Match | Default | Client JSON Mapping |
|-------|:---:|:---:|:---:|:---:|---------|:---:|
| database_backup_retention_amount_locally | integer | - | - | - | 0 | mapped |
| database_backup_retention_amount_s3 | string | - | - | - | - | mapped |
| database_backup_retention_days_locally | string | - | - | - | - | mapped |
| database_backup_retention_days_s3 | string | - | - | - | - | mapped |
| database_backup_retention_max_storage_locally | string | - | - | - | - | mapped |
| database_backup_retention_max_storage_s3 | string | - | - | - | - | mapped |
| database_id | integer | - | - | - | - | n/a |
| database_type | string | - | - | - | - | mapped |
| databases_to_backup | string | - | - | - | - | mapped |
| description | string | - | - | - | - | mapped |
| disable_local_backup | string | - | - | - | - | n/a |
| dump_all | string | - | - | - | - | mapped |
| enabled | boolean | - | - | - | true | mapped |
| frequency | string | - | - | - | - | mapped |
| number_of_backups_locally | integer | - | - | - | 7 | n/a |
| s3_storage_id | integer | - | - | - | - | n/a |
| save_s3 | boolean | - | - | - | true | mapped |
| timeout | integer | - | - | - | 3600 | mapped |
| uuid | string | - | - | - | - | mapped |

## CloudProviderToken

This model exists in the extracted source contract but not as a reusable public OpenAPI schema.
Treat it as implementation detail coverage, not proof of a standalone public API surface.

Fields: 3 | Type matches: 3/3 | Nullable matches: 3/3 | Client JSON mappings: 3/3

| Field | Contract Type | Spec Type | Type Match | Nullable Match | Default | Client JSON Mapping |
|-------|:---:|:---:|:---:|:---:|---------|:---:|
| name | string | - | - | - | - | mapped |
| provider | string | - | - | - | - | mapped |
| token | string | - | - | - | - | mapped |

## GithubApp

This model exists in the extracted source contract but not as a reusable public OpenAPI schema.
Treat it as implementation detail coverage, not proof of a standalone public API surface.

Fields: 19 | Type matches: 19/19 | Nullable matches: 19/19 | Client JSON mappings: 11/19

| Field | Contract Type | Spec Type | Type Match | Nullable Match | Default | Client JSON Mapping |
|-------|:---:|:---:|:---:|:---:|---------|:---:|
| administration | string | - | - | - | - | n/a |
| api_url | string | - | - | - | - | mapped |
| app_id | integer | - | - | - | - | mapped |
| client_id | string | - | - | - | - | mapped |
| client_secret | string | - | - | - | - | mapped |
| contents | string | - | - | - | - | n/a |
| custom_port | integer | - | - | - | 22 | n/a |
| custom_user | string | - | - | - | git | n/a |
| html_url | string | - | - | - | - | mapped |
| installation_id | integer | - | - | - | - | mapped |
| is_public | boolean | - | - | - | false | mapped |
| is_system_wide | boolean | - | - | - | false | n/a |
| metadata | string | - | - | - | - | n/a |
| name | string | - | - | - | - | mapped |
| organization | string | - | - | - | - | mapped |
| private_key_id | integer | - | - | - | - | n/a |
| pull_requests | string | - | - | - | - | n/a |
| uuid | string | - | - | - | - | mapped |
| webhook_secret | string | - | - | - | - | mapped |

## LocalPersistentVolume

Fields: 8 | Type matches: 8/8 | Nullable matches: 8/8 | Client JSON mappings: 3/8

| Field | Contract Type | Spec Type | Type Match | Nullable Match | Default | Client JSON Mapping |
|-------|:---:|:---:|:---:|:---:|---------|:---:|
| container_id | string | - | - | - | - | n/a |
| host_path | string | - | - | - | - | mapped |
| is_preview_suffix_enabled | string | - | - | - | - | n/a |
| is_readonly | boolean | - | - | - | false | n/a |
| mount_path | string | - | - | - | - | mapped |
| name | string | - | - | - | - | mapped |
| resource_id | integer | - | - | - | - | n/a |
| resource_type | string | - | - | - | - | n/a |

## S3Storage

This model exists in the extracted source contract but not as a reusable public OpenAPI schema.
Treat it as implementation detail coverage, not proof of a standalone public API surface.

Fields: 10 | Type matches: 10/10 | Nullable matches: 10/10 | Client JSON mappings: 0/10

| Field | Contract Type | Spec Type | Type Match | Nullable Match | Default | Client JSON Mapping |
|-------|:---:|:---:|:---:|:---:|---------|:---:|
| bucket | string | - | - | - | - | n/a |
| description | string | - | - | - | - | n/a |
| endpoint | string | - | - | - | - | n/a |
| is_usable | string | - | - | - | - | n/a |
| key | string | - | - | - | - | n/a |
| name | string | - | - | - | - | n/a |
| region | string | - | - | - | us-east-1 | n/a |
| secret | string | - | - | - | - | n/a |
| unusable_email_sent | string | - | - | - | - | n/a |
| uuid | string | - | - | - | - | n/a |

## StandaloneClickhouse

Fields: 30 | Type matches: 30/30 | Nullable matches: 30/30 | Client JSON mappings: 20/30

| Field | Contract Type | Spec Type | Type Match | Nullable Match | Default | Client JSON Mapping |
|-------|:---:|:---:|:---:|:---:|---------|:---:|
| clickhouse_admin_password | string | - | - | - | - | mapped |
| clickhouse_admin_user | string | - | - | - | default | mapped |
| clickhouse_db | string | - | - | - | default | mapped |
| custom_docker_run_options | string | - | - | - | - | mapped |
| description | string | - | - | - | - | mapped |
| destination_id | integer | - | - | - | - | n/a |
| destination_type | string | - | - | - | - | n/a |
| environment_id | integer | - | - | - | - | n/a |
| image | string | - | - | - | bitnamilegacy/clickhouse | mapped |
| is_include_timestamps | boolean | - | - | - | false | n/a |
| is_log_drain_enabled | boolean | - | - | - | false | n/a |
| is_public | boolean | - | - | - | false | mapped |
| last_online_at | string | - | - | - | - | n/a |
| last_restart_at | string | - | - | - | - | n/a |
| last_restart_type | string | - | - | - | - | n/a |
| limits_cpu_shares | integer | - | - | - | 1024 | mapped |
| limits_cpus | string | - | - | - | 0 | mapped |
| limits_cpuset | string | - | - | - | - | mapped |
| limits_memory | string | - | - | - | 0 | mapped |
| limits_memory_reservation | string | - | - | - | 0 | mapped |
| limits_memory_swap | string | - | - | - | 0 | mapped |
| limits_memory_swappiness | integer | - | - | - | 60 | mapped |
| name | string | - | - | - | - | mapped |
| ports_mappings | string | - | - | - | - | mapped |
| public_port | integer | - | - | - | - | mapped |
| public_port_timeout | string | - | - | - | - | mapped |
| restart_count | string | - | - | - | - | n/a |
| started_at | string | - | - | - | - | n/a |
| status | string | - | - | - | exited | mapped |
| uuid | string | - | - | - | - | mapped |

## StandaloneDragonfly

Fields: 29 | Type matches: 29/29 | Nullable matches: 29/29 | Client JSON mappings: 18/29

| Field | Contract Type | Spec Type | Type Match | Nullable Match | Default | Client JSON Mapping |
|-------|:---:|:---:|:---:|:---:|---------|:---:|
| custom_docker_run_options | string | - | - | - | - | mapped |
| description | string | - | - | - | - | mapped |
| destination_id | integer | - | - | - | - | n/a |
| destination_type | string | - | - | - | - | n/a |
| dragonfly_password | string | - | - | - | - | mapped |
| enable_ssl | string | - | - | - | - | n/a |
| environment_id | integer | - | - | - | - | n/a |
| image | string | - | - | - | docker.dragonflydb.io/dragonflydb/dragonfly | mapped |
| is_include_timestamps | boolean | - | - | - | false | n/a |
| is_log_drain_enabled | boolean | - | - | - | false | n/a |
| is_public | boolean | - | - | - | false | mapped |
| last_online_at | string | - | - | - | - | n/a |
| last_restart_at | string | - | - | - | - | n/a |
| last_restart_type | string | - | - | - | - | n/a |
| limits_cpu_shares | integer | - | - | - | 1024 | mapped |
| limits_cpus | string | - | - | - | 0 | mapped |
| limits_cpuset | string | - | - | - | - | mapped |
| limits_memory | string | - | - | - | 0 | mapped |
| limits_memory_reservation | string | - | - | - | 0 | mapped |
| limits_memory_swap | string | - | - | - | 0 | mapped |
| limits_memory_swappiness | integer | - | - | - | 60 | mapped |
| name | string | - | - | - | - | mapped |
| ports_mappings | string | - | - | - | - | mapped |
| public_port | integer | - | - | - | - | mapped |
| public_port_timeout | string | - | - | - | - | mapped |
| restart_count | string | - | - | - | - | n/a |
| started_at | string | - | - | - | - | n/a |
| status | string | - | - | - | exited | mapped |
| uuid | string | - | - | - | - | mapped |

## StandaloneKeydb

Fields: 30 | Type matches: 30/30 | Nullable matches: 30/30 | Client JSON mappings: 19/30

| Field | Contract Type | Spec Type | Type Match | Nullable Match | Default | Client JSON Mapping |
|-------|:---:|:---:|:---:|:---:|---------|:---:|
| custom_docker_run_options | string | - | - | - | - | mapped |
| description | string | - | - | - | - | mapped |
| destination_id | integer | - | - | - | - | n/a |
| destination_type | string | - | - | - | - | n/a |
| enable_ssl | string | - | - | - | - | n/a |
| environment_id | integer | - | - | - | - | n/a |
| image | string | - | - | - | eqalpha/keydb:latest | mapped |
| is_include_timestamps | boolean | - | - | - | false | n/a |
| is_log_drain_enabled | boolean | - | - | - | false | n/a |
| is_public | boolean | - | - | - | false | mapped |
| keydb_conf | string | - | - | - | - | mapped |
| keydb_password | string | - | - | - | - | mapped |
| last_online_at | string | - | - | - | - | n/a |
| last_restart_at | string | - | - | - | - | n/a |
| last_restart_type | string | - | - | - | - | n/a |
| limits_cpu_shares | integer | - | - | - | 1024 | mapped |
| limits_cpus | string | - | - | - | 0 | mapped |
| limits_cpuset | string | - | - | - | - | mapped |
| limits_memory | string | - | - | - | 0 | mapped |
| limits_memory_reservation | string | - | - | - | 0 | mapped |
| limits_memory_swap | string | - | - | - | 0 | mapped |
| limits_memory_swappiness | integer | - | - | - | 60 | mapped |
| name | string | - | - | - | - | mapped |
| ports_mappings | string | - | - | - | - | mapped |
| public_port | integer | - | - | - | - | mapped |
| public_port_timeout | string | - | - | - | - | mapped |
| restart_count | string | - | - | - | - | n/a |
| started_at | string | - | - | - | - | n/a |
| status | string | - | - | - | exited | mapped |
| uuid | string | - | - | - | - | mapped |

## StandaloneMariadb

Fields: 32 | Type matches: 32/32 | Nullable matches: 32/32 | Client JSON mappings: 22/32

| Field | Contract Type | Spec Type | Type Match | Nullable Match | Default | Client JSON Mapping |
|-------|:---:|:---:|:---:|:---:|---------|:---:|
| custom_docker_run_options | string | - | - | - | - | mapped |
| description | string | - | - | - | - | mapped |
| destination_id | integer | - | - | - | - | n/a |
| destination_type | string | - | - | - | - | n/a |
| enable_ssl | string | - | - | - | - | n/a |
| environment_id | integer | - | - | - | - | n/a |
| image | string | - | - | - | mariadb:11 | mapped |
| is_log_drain_enabled | string | - | - | - | - | n/a |
| is_public | boolean | - | - | - | false | mapped |
| last_online_at | string | - | - | - | - | n/a |
| last_restart_at | string | - | - | - | - | n/a |
| last_restart_type | string | - | - | - | - | n/a |
| limits_cpu_shares | integer | - | - | - | 1024 | mapped |
| limits_cpus | string | - | - | - | 0 | mapped |
| limits_cpuset | string | - | - | - | 0 | mapped |
| limits_memory | string | - | - | - | 0 | mapped |
| limits_memory_reservation | string | - | - | - | 0 | mapped |
| limits_memory_swap | string | - | - | - | 0 | mapped |
| limits_memory_swappiness | integer | - | - | - | 60 | mapped |
| mariadb_conf | string | - | - | - | - | mapped |
| mariadb_database | string | - | - | - | default | mapped |
| mariadb_password | string | - | - | - | - | mapped |
| mariadb_root_password | string | - | - | - | - | mapped |
| mariadb_user | string | - | - | - | mariadb | mapped |
| name | string | - | - | - | - | mapped |
| ports_mappings | string | - | - | - | - | mapped |
| public_port | integer | - | - | - | - | mapped |
| public_port_timeout | string | - | - | - | - | mapped |
| restart_count | string | - | - | - | - | n/a |
| started_at | string | - | - | - | - | n/a |
| status | string | - | - | - | exited | mapped |
| uuid | string | - | - | - | - | mapped |

## StandaloneMongodb

Fields: 33 | Type matches: 33/33 | Nullable matches: 33/33 | Client JSON mappings: 21/33

| Field | Contract Type | Spec Type | Type Match | Nullable Match | Default | Client JSON Mapping |
|-------|:---:|:---:|:---:|:---:|---------|:---:|
| custom_docker_run_options | string | - | - | - | - | mapped |
| description | string | - | - | - | - | mapped |
| destination_id | integer | - | - | - | - | n/a |
| destination_type | string | - | - | - | - | n/a |
| enable_ssl | boolean | - | - | - | true | n/a |
| environment_id | integer | - | - | - | - | n/a |
| image | string | - | - | - | mongo:7 | mapped |
| is_include_timestamps | string | - | - | - | - | n/a |
| is_log_drain_enabled | string | - | - | - | - | n/a |
| is_public | boolean | - | - | - | false | mapped |
| last_online_at | string | - | - | - | - | n/a |
| last_restart_at | string | - | - | - | - | n/a |
| last_restart_type | string | - | - | - | - | n/a |
| limits_cpu_shares | integer | - | - | - | 1024 | mapped |
| limits_cpus | string | - | - | - | 0 | mapped |
| limits_cpuset | string | - | - | - | 0 | mapped |
| limits_memory | string | - | - | - | 0 | mapped |
| limits_memory_reservation | string | - | - | - | 0 | mapped |
| limits_memory_swap | string | - | - | - | 0 | mapped |
| limits_memory_swappiness | integer | - | - | - | 60 | mapped |
| mongo_conf | string | - | - | - | - | mapped |
| mongo_initdb_database | string | - | - | - | default | mapped |
| mongo_initdb_root_password | string | - | - | - | - | mapped |
| mongo_initdb_root_username | string | - | - | - | root | mapped |
| name | string | - | - | - | - | mapped |
| ports_mappings | string | - | - | - | - | mapped |
| public_port | integer | - | - | - | - | mapped |
| public_port_timeout | string | - | - | - | - | mapped |
| restart_count | string | - | - | - | - | n/a |
| ssl_mode | string | - | - | - | - | n/a |
| started_at | string | - | - | - | - | n/a |
| status | string | - | - | - | exited | mapped |
| uuid | string | - | - | - | - | mapped |

## StandaloneMysql

Fields: 34 | Type matches: 34/34 | Nullable matches: 34/34 | Client JSON mappings: 22/34

| Field | Contract Type | Spec Type | Type Match | Nullable Match | Default | Client JSON Mapping |
|-------|:---:|:---:|:---:|:---:|---------|:---:|
| custom_docker_run_options | string | - | - | - | - | mapped |
| description | string | - | - | - | - | mapped |
| destination_id | integer | - | - | - | - | n/a |
| destination_type | string | - | - | - | - | n/a |
| enable_ssl | string | - | - | - | - | n/a |
| environment_id | integer | - | - | - | - | n/a |
| image | string | - | - | - | mysql:8 | mapped |
| is_include_timestamps | string | - | - | - | - | n/a |
| is_log_drain_enabled | string | - | - | - | - | n/a |
| is_public | boolean | - | - | - | false | mapped |
| last_online_at | string | - | - | - | - | n/a |
| last_restart_at | string | - | - | - | - | n/a |
| last_restart_type | string | - | - | - | - | n/a |
| limits_cpu_shares | integer | - | - | - | 1024 | mapped |
| limits_cpus | string | - | - | - | 0 | mapped |
| limits_cpuset | string | - | - | - | 0 | mapped |
| limits_memory | string | - | - | - | 0 | mapped |
| limits_memory_reservation | string | - | - | - | 0 | mapped |
| limits_memory_swap | string | - | - | - | 0 | mapped |
| limits_memory_swappiness | integer | - | - | - | 60 | mapped |
| mysql_conf | string | - | - | - | - | mapped |
| mysql_database | string | - | - | - | default | mapped |
| mysql_password | string | - | - | - | - | mapped |
| mysql_root_password | string | - | - | - | - | mapped |
| mysql_user | string | - | - | - | mysql | mapped |
| name | string | - | - | - | - | mapped |
| ports_mappings | string | - | - | - | - | mapped |
| public_port | integer | - | - | - | - | mapped |
| public_port_timeout | string | - | - | - | - | mapped |
| restart_count | string | - | - | - | - | n/a |
| ssl_mode | string | - | - | - | - | n/a |
| started_at | string | - | - | - | - | n/a |
| status | string | - | - | - | exited | mapped |
| uuid | string | - | - | - | - | mapped |

## StandalonePostgresql

Fields: 36 | Type matches: 36/36 | Nullable matches: 36/36 | Client JSON mappings: 24/36

| Field | Contract Type | Spec Type | Type Match | Nullable Match | Default | Client JSON Mapping |
|-------|:---:|:---:|:---:|:---:|---------|:---:|
| custom_docker_run_options | string | - | - | - | - | mapped |
| description | string | - | - | - | - | mapped |
| destination_id | integer | - | - | - | - | n/a |
| destination_type | string | - | - | - | - | n/a |
| enable_ssl | string | - | - | - | - | n/a |
| environment_id | integer | - | - | - | - | n/a |
| image | string | - | - | - | postgres:16-alpine | mapped |
| init_scripts | object | - | - | - | - | mapped |
| is_include_timestamps | string | - | - | - | - | n/a |
| is_log_drain_enabled | string | - | - | - | - | n/a |
| is_public | boolean | - | - | - | false | mapped |
| last_online_at | string | - | - | - | - | n/a |
| last_restart_at | string | - | - | - | - | n/a |
| last_restart_type | string | - | - | - | - | n/a |
| limits_cpu_shares | integer | - | - | - | 1024 | mapped |
| limits_cpus | string | - | - | - | 0 | mapped |
| limits_cpuset | string | - | - | - | 0 | mapped |
| limits_memory | string | - | - | - | 0 | mapped |
| limits_memory_reservation | string | - | - | - | 0 | mapped |
| limits_memory_swap | string | - | - | - | 0 | mapped |
| limits_memory_swappiness | integer | - | - | - | 60 | mapped |
| name | string | - | - | - | - | mapped |
| ports_mappings | string | - | - | - | - | mapped |
| postgres_conf | string | - | - | - | - | mapped |
| postgres_db | string | - | - | - | postgres | mapped |
| postgres_host_auth_method | string | - | - | - | - | mapped |
| postgres_initdb_args | string | - | - | - | - | mapped |
| postgres_password | string | - | - | - | - | mapped |
| postgres_user | string | - | - | - | postgres | mapped |
| public_port | integer | - | - | - | - | mapped |
| public_port_timeout | string | - | - | - | - | mapped |
| restart_count | string | - | - | - | - | n/a |
| ssl_mode | string | - | - | - | - | n/a |
| started_at | string | - | - | - | - | n/a |
| status | string | - | - | - | exited | mapped |
| uuid | string | - | - | - | - | mapped |

## StandaloneRedis

Fields: 29 | Type matches: 29/29 | Nullable matches: 29/29 | Client JSON mappings: 18/29

| Field | Contract Type | Spec Type | Type Match | Nullable Match | Default | Client JSON Mapping |
|-------|:---:|:---:|:---:|:---:|---------|:---:|
| custom_docker_run_options | string | - | - | - | - | mapped |
| description | string | - | - | - | - | mapped |
| destination_id | integer | - | - | - | - | n/a |
| destination_type | string | - | - | - | - | n/a |
| enable_ssl | string | - | - | - | - | n/a |
| environment_id | integer | - | - | - | - | n/a |
| image | string | - | - | - | redis:7.2 | mapped |
| is_include_timestamps | string | - | - | - | - | n/a |
| is_log_drain_enabled | string | - | - | - | - | n/a |
| is_public | boolean | - | - | - | false | mapped |
| last_online_at | string | - | - | - | - | n/a |
| last_restart_at | string | - | - | - | - | n/a |
| last_restart_type | string | - | - | - | - | n/a |
| limits_cpu_shares | integer | - | - | - | 1024 | mapped |
| limits_cpus | string | - | - | - | 0 | mapped |
| limits_cpuset | string | - | - | - | 0 | mapped |
| limits_memory | string | - | - | - | 0 | mapped |
| limits_memory_reservation | string | - | - | - | 0 | mapped |
| limits_memory_swap | string | - | - | - | 0 | mapped |
| limits_memory_swappiness | integer | - | - | - | 60 | mapped |
| name | string | - | - | - | - | mapped |
| ports_mappings | string | - | - | - | - | mapped |
| public_port | integer | - | - | - | - | mapped |
| public_port_timeout | string | - | - | - | - | mapped |
| redis_conf | string | - | - | - | - | mapped |
| restart_count | string | - | - | - | - | n/a |
| started_at | string | - | - | - | - | n/a |
| status | string | - | - | - | exited | mapped |
| uuid | string | - | - | - | - | mapped |

