package spectest

// Contract field skip tables (#622). deferred entries must cite an open issue.
// Umbrella backlog for remaining product fields: #626.

// applicationFieldSkips are Application model + ApplicationSetting fields not
// required on client.Application.
var applicationFieldSkips = skipMap(
	// Internal DB identifiers (not exposed via API)
	skipInternal("environment_id", "numeric FK"),
	skipInternal("destination_id", "numeric FK"),
	skipInternal("destination_type", "morph type"),
	skipInternal("source_id", "numeric FK"),
	skipInternal("source_type", "morph type"),
	skipInternal("private_key_id", "numeric FK"),
	skipInternal("repository_project_id", "numeric FK"),
	// Computed/internal fields not exposed by the API
	skipInternal("config_hash", "computed hash"),
	skipInternal("custom_healthcheck_found", "computed flag"),
	skipInternal("compose_parsing_version", "internal parser version"),
	skipInternal("last_online_at", "runtime status"),
	skipInternal("last_restart_at", "runtime status"),
	skipInternal("last_restart_type", "runtime status"),
	skipInternal("restart_count", "runtime status"),
	skipInternal("nixpkgsarchive", "internal build archive"),
	skipInternal("git_full_url", "computed URL"),
	// Docker compose PR fields (removed in later migrations)
	skipNA("docker_compose_pr_location", "removed from Coolify API surface"),
	skipNA("docker_compose_pr", "removed from Coolify API surface"),
	skipNA("docker_compose_pr_raw", "removed from Coolify API surface"),
	// Fields served from related models, not Application table
	skipNA("docker_compose", "served from related model, not Application row"),
	// Swarm-only fields (not commonly used)
	skipNA("swarm_replicas", "swarm-only; not in provider surface"),
	skipNA("swarm_placement_constraints", "swarm-only; not in provider surface"),
	// Internal-only ApplicationSetting fields not exposed by update API
	skipInternal("application_id", "settings FK"),
	skipInternal("custom_internal_name", "settings internal"),
	skipInternal("disable_build_cache", "settings not on update allow list"),
	skipInternal("docker_images_to_keep", "settings not on update allow list"),
	skipInternal("gpu_count", "settings not on update allow list"),
	skipInternal("gpu_device_ids", "settings not on update allow list"),
	skipInternal("gpu_driver", "settings not on update allow list"),
	skipInternal("gpu_options", "settings not on update allow list"),
	skipInternal("include_source_commit_in_build", "settings not on update allow list"),
	skipInternal("inject_build_args_to_dockerfile", "settings not on update allow list"),
	skipInternal("is_consistent_container_name_enabled", "settings not on update allow list"),
	skipInternal("is_container_label_readonly_enabled", "settings not on update allow list"),
	skipInternal("is_custom_ssl", "settings not on update allow list"),
	skipInternal("is_debug_enabled", "settings not on update allow list"),
	skipInternal("is_dual_cert", "settings not on update allow list"),
	skipInternal("is_env_sorting_enabled", "settings not on update allow list"),
	skipInternal("is_git_lfs_enabled", "settings not on update allow list"),
	skipInternal("is_git_shallow_clone_enabled", "settings not on update allow list"),
	skipInternal("is_git_submodules_enabled", "settings not on update allow list"),
	skipInternal("is_gpu_enabled", "settings not on update allow list"),
	skipInternal("is_gzip_enabled", "settings not on update allow list"),
	skipInternal("is_http2", "settings not on update allow list"),
	skipInternal("is_include_timestamps", "settings not on update allow list"),
	skipInternal("is_log_drain_enabled", "settings not on update allow list"),
	skipInternal("is_pr_deployments_public_enabled", "settings not on update allow list"),
	skipInternal("is_preview_deployments_enabled", "settings; version-gated separately where exposed"),
	skipInternal("is_raw_compose_deployment_enabled", "settings not on update allow list"),
	skipInternal("is_stripprefix_enabled", "settings not on update allow list"),
	skipInternal("is_swarm_only_worker_nodes", "settings not on update allow list"),
	skipInternal("use_build_secrets", "settings; version-gated separately where exposed"),
	// is_build_server_enabled is the setting name; the API field is use_build_server
	skipNA("is_build_server_enabled", "API field is use_build_server on Application"),
	skipDeferred("stop_grace_period", 626, "not yet exposed on application resource"),
)

var serverCoverageSkips = skipMap(
	skipInternal("team_id", "FK"),
	skipInternal("private_key_id", "numeric FK"),
	skipInternal("proxy", "complex JSON object"),
	skipInternal("sentinel_token", "hidden by middleware"),
	skipInternal("sentinel_custom_url", "sentinel config not on base server entity"),
	skipInternal("sentinel_metrics_token", "sentinel metrics secret"),
	skipInternal("sentinel_metrics_history", "sentinel metrics config"),
	skipInternal("sentinel_metrics_interval", "sentinel metrics config"),
	skipInternal("started_at", "runtime status"),
	skipInternal("last_online_at", "runtime status"),
	skipInternal("last_restart_at", "runtime status"),
	skipInternal("last_restart_type", "runtime status"),
	skipInternal("restart_count", "runtime status"),
	skipInternal("unreachable_notification", "runtime flag"),
	skipInternal("unreachable_count", "runtime counter"),
	skipInternal("log_drain_notification", "notification flag"),
	skipInternal("swarm_cluster", "swarm cluster blob"),
	skipInternal("cloud_provider_token_id", "numeric FK"),
	skipInternal("detected_traefik_version", "ephemeral status"),
	skipInternal("hetzner_server_id", "Hetzner-specific tracking"),
	skipInternal("hetzner_server_status", "Hetzner-specific status"),
	skipInternal("ip_previous", "internal tracking"),
	skipInternal("is_validating", "ephemeral status"),
	skipInternal("server_metadata", "internal metadata"),
	skipInternal("traefik_outdated_info", "ephemeral status"),
)

var serviceCoverageSkips = skipMap(
	skipInternal("team_id", "FK"),
	skipInternal("environment_id", "numeric FK"),
	skipInternal("destination_id", "numeric FK"),
	skipInternal("destination_type", "morph type"),
	skipInternal("server_id", "numeric FK"),
	skipInternal("is_container_label_readonly_enabled", "internal label flag"),
	skipInternal("is_readonly", "internal flag"),
	skipInternal("compose_parsing_version", "internal config"),
	skipNA("service_type", "mapped to type in client struct"),
)

var privateKeyCoverageSkips = skipMap(
	skipInternal("team_id", "FK"),
)

// environmentVariableCoverageSkips: after #619, deploy flags are on the client.
// Remaining product/UI fields track under #626.
var environmentVariableCoverageSkips = skipMap(
	skipInternal("resourceable_id", "polymorphic FK"),
	skipInternal("resourceable_type", "polymorphic FK"),
	skipInternal("team_id", "FK"),
	skipInternal("real_value", "computed accessor"),
	skipInternal("version", "Coolify internal version stamp"),
	skipDeferred("is_required", 626, "product surface not yet managed in Terraform"),
	skipDeferred("is_shared", 626, "shared-var surface not managed as resource field"),
	skipDeferred("is_shown_once", 626, "UI reveal-once; not deploy semantics"),
	skipDeferred("order", 626, "UI ordering not managed in Terraform"),
)

var scheduledTaskCoverageSkips = skipMap(
	skipInternal("application_id", "numeric FK"),
	skipInternal("service_id", "numeric FK"),
	skipInternal("team_id", "FK"),
	skipInternal("container", "not user-facing container name"),
	skipDeferred("timeout", 626, "not yet exposed on scheduled_task resource"),
)

var projectCoverageSkips = skipMap(
	skipInternal("team_id", "FK"),
)

var githubAppCoverageSkips = skipMap(
	skipInternal("team_id", "FK"),
	skipInternal("private_key_id", "numeric FK"),
	skipInternal("client_secret", "sensitive write-only; hidden on read"),
	skipInternal("is_system_wide", "system flag"),
	skipInternal("administration", "GitHub permission scope"),
	skipInternal("contents", "GitHub permission scope"),
	skipInternal("custom_port", "internal git config"),
	skipInternal("custom_user", "internal git config"),
	skipInternal("is_public", "internal flag"),
	skipInternal("metadata", "GitHub permission scope"),
	skipInternal("pull_requests", "GitHub permission scope"),
)

var databaseBackupCoverageSkips = skipMap(
	skipInternal("team_id", "FK"),
	skipInternal("database_id", "numeric FK"),
	skipDeferred("description", 626, "not yet exposed on database_backup"),
	skipDeferred("disable_local_backup", 626, "not yet on client DatabaseBackup entity"),
	skipNA("s3_storage_id", "numeric FK; provider uses s3_storage_uuid"),
)

var cloudTokenCoverageSkips = skipMap(
	skipInternal("team_id", "FK"),
)

var storageCoverageSkips = skipMap(
	skipInternal("container_id", "internal Docker container ID"),
	skipNA("resource_id", "numeric FK; provider uses resource_uuid"),
	skipDeferred("is_preview_suffix_enabled", 626, "not yet exposed on storage resource"),
)

var serverSettingCoverageSkips = skipMap(
	skipInternal("server_id", "FK"),
	skipNA("is_build_server", "on Server entity, not Settings"),
	skipDeferred("is_force_disabled", 626, "not yet exposed on server settings"),
	skipNA("is_reachable", "on Server entity, not Settings"),
	skipNA("is_usable", "on Server entity, not Settings"),
	skipDeferred("is_swarm_manager", 626, "not yet exposed on server settings"),
	skipDeferred("is_swarm_worker", 626, "not yet exposed on server settings"),
	skipDeferred("sentinel_custom_url", 626, "not yet exposed on server settings"),
	skipNA("sentinel_metrics_refresh_rate_in_seconds", "old contract name; superseded by sentinel_metrics_refresh_rate_seconds"),
	skipNA("sentinel_push_interval_in_seconds", "old contract name; superseded by sentinel_push_interval_seconds"),
	skipDeferred("sentinel_token", 626, "sensitive; not yet exposed"),
	skipDeferred("is_logdrain_axiom_enabled", 626, "not yet exposed"),
	skipDeferred("logdrain_axiom_api_key", 626, "sensitive; not yet exposed"),
	skipDeferred("logdrain_axiom_dataset_name", 626, "not yet exposed"),
	skipDeferred("is_logdrain_custom_enabled", 626, "not yet exposed"),
	skipDeferred("logdrain_custom_config", 626, "not yet exposed"),
	skipDeferred("logdrain_custom_config_parser", 626, "not yet exposed"),
	skipDeferred("is_logdrain_highlight_enabled", 626, "not yet exposed"),
	skipDeferred("logdrain_highlight_project_id", 626, "not yet exposed"),
	skipDeferred("is_logdrain_newrelic_enabled", 626, "not yet exposed"),
	skipDeferred("logdrain_newrelic_base_uri", 626, "not yet exposed"),
	skipDeferred("logdrain_newrelic_license_key", 626, "sensitive; not yet exposed"),
	skipDeferred("disable_application_image_retention", 626, "not yet exposed"),
	skipDeferred("force_disabled", 626, "not yet exposed"),
	skipDeferred("is_jump_server", 626, "not yet exposed"),
	skipDeferred("is_sentinel_debug_enabled", 626, "not yet exposed"),
)

var environmentCoverageSkips = skipMap(
	skipNA("project_id", "numeric FK; provider uses project_uuid"),
	skipNA("uuid", "provider identifies environments by name"),
)

var databaseModelSkips = skipMap(
	skipInternal("environment_id", "numeric FK"),
	skipInternal("destination_id", "numeric FK"),
	skipInternal("destination_type", "morph type"),
	skipInternal("started_at", "runtime status"),
	skipInternal("last_online_at", "runtime status"),
	skipInternal("last_restart_at", "runtime status"),
	skipInternal("last_restart_type", "runtime status"),
	skipInternal("restart_count", "runtime status"),
)

// appEnvWriteSkips: allowed_fields on application create_env / update_env that
// are intentionally not on applicationEnvVarInput (#623).
var appEnvWriteSkips = skipMap(
	skipDeferred("is_shown_once", 626, "UI reveal-once; not sent on write path yet"),
)
