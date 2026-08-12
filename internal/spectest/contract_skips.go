package spectest

// Contract field skip tables (#622). deferred entries must cite an open issue.
// Deferred public product fields use skipDeferred + issue number.
// #626 was cleared when all remaining skips were either implemented or reclassified as n/a.

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
	// Internal-only ApplicationSetting fields not on ApplicationsController
	// APPLICATION_SETTING_FIELDS / update allow list (UI-only or other surfaces).
	// Note: disable_build_cache, is_gzip_enabled, is_git_*, stop_grace_period, and
	// Coolify v4.3.0 additions (log drain, GPU, consistent container name,
	// custom_internal_name, noindex_domains) ARE on the public allow list via
	// ...self::APPLICATION_SETTING_FIELDS (extractor expands spreads, #661);
	// they must not be skipInternal.
	skipInternal("application_id", "settings FK"),
	skipInternal("is_container_label_readonly_enabled", "settings UI-only; not in APPLICATION_SETTING_FIELDS"),
	skipInternal("is_custom_ssl", "settings UI-only; not in APPLICATION_SETTING_FIELDS"),
	skipInternal("is_debug_enabled", "settings UI-only; not in APPLICATION_SETTING_FIELDS"),
	skipInternal("is_dual_cert", "settings UI-only; not in APPLICATION_SETTING_FIELDS"),
	skipInternal("is_http2", "settings UI-only; not in APPLICATION_SETTING_FIELDS"),
	skipInternal("is_include_timestamps", "settings UI-only; not in APPLICATION_SETTING_FIELDS"),
	skipInternal("is_swarm_only_worker_nodes", "settings UI-only; not in APPLICATION_SETTING_FIELDS"),
	// Runtime DNS probe state on Application; not Terraform-managed.
	skipNA("domain_dns_statuses", "runtime DNS probe state; not Terraform-managed"),
	// is_build_server_enabled is the setting name; the API field is use_build_server
	skipNA("is_build_server_enabled", "API field is use_build_server on Application"),
	// is_preview_deployments_enabled, use_build_secrets, stop_grace_period and
	// APPLICATION_SETTING_FIELDS are public allow-list fields covered on client (#628, #661)
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
	// Public on cloud create-server allow lists; provider maps tokens via UUID fields.
	skipNA("cloud_provider_token_id", "numeric FK; cloud server resources use token UUID fields"),
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
var environmentVariableCoverageSkips = skipMap(
	skipInternal("resourceable_id", "polymorphic FK"),
	skipInternal("resourceable_type", "polymorphic FK"),
	skipInternal("team_id", "FK"),
	skipInternal("real_value", "computed accessor"),
	skipInternal("version", "Coolify internal version stamp"),
	// is_shown_once covered on client + coolify_environment_variable schema (app write path).
	skipNA("is_required", "not on ApplicationsController create_env/update_env $allowedFields"),
	skipNA("is_shared", "not on ApplicationsController create_env/update_env $allowedFields"),
	skipNA("order", "not on ApplicationsController create_env/update_env $allowedFields"),
)

var scheduledTaskCoverageSkips = skipMap(
	skipInternal("application_id", "numeric FK"),
	skipInternal("service_id", "numeric FK"),
	skipInternal("team_id", "FK"),
	// container, timeout covered on client (#629)
)

var projectCoverageSkips = skipMap(
	skipInternal("team_id", "FK"),
)

var githubAppCoverageSkips = skipMap(
	skipInternal("team_id", "FK"),
	skipInternal("private_key_id", "numeric FK"),
	// Write-only on create/update; API never returns it on the entity read struct.
	// Schema+create/update already manage client_secret; entity coverage is n/a.
	skipNA("client_secret", "write-only sensitive; intentionally omitted from GitHubApp read entity"),
	skipInternal("administration", "GitHub permission scope blob"),
	skipInternal("contents", "GitHub permission scope blob"),
	skipNA("is_public", "GitHub App visibility flag; not mapped on provider github_app resource"),
	skipInternal("metadata", "GitHub permission scope blob"),
	skipInternal("pull_requests", "GitHub permission scope blob"),
)

var databaseBackupCoverageSkips = skipMap(
	skipInternal("team_id", "FK"),
	skipInternal("database_id", "numeric FK"),
	// description + disable_local_backup covered on client entity and schema (Computed read-only).
	skipNA("s3_storage_id", "numeric FK; provider uses s3_storage_uuid"),
)

var cloudTokenCoverageSkips = skipMap(
	skipInternal("team_id", "FK"),
)

var storageCoverageSkips = skipMap(
	skipInternal("container_id", "internal Docker container ID"),
	skipNA("resource_id", "numeric FK; provider uses resource_uuid"),
	// is_preview_suffix_enabled covered on client + coolify_storage schema.
)

var serverSettingCoverageSkips = skipMap(
	skipInternal("server_id", "FK"),
	skipNA("is_build_server", "on Server entity, not Settings"),
	skipNA("is_force_disabled", "alias of force_disabled on ServerSetting model"),
	skipNA("is_reachable", "on Server entity, not Settings"),
	skipNA("is_usable", "on Server entity, not Settings"),
	skipNA("sentinel_metrics_refresh_rate_in_seconds", "old contract name; superseded by sentinel_metrics_refresh_rate_seconds"),
	skipNA("sentinel_push_interval_in_seconds", "old contract name; superseded by sentinel_push_interval_seconds"),
	// Remaining settings fields covered on client.ServerSettings + read-only server schema.
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
// is_shown_once covered on applicationEnvVarInput write path.
)

// applicationCreateWriteSkips: create allow-list fields that are intentionally
// not on Create*AppInput (or not schema) for silent-default coverage (#643).
// Prefer implementing the field over skipping; use skipDeferred with an issue
// when product work is tracked.
var applicationCreateWriteSkips = skipMap(
// empty: autogenerate_domain is required on create inputs + schema (#642).
)
