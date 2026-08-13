package spectest

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// coverageStatus tracks a single API endpoint's provider coverage.
type coverageStatus struct {
	category string // "covered", "planned", "skipped"
	resource string // Terraform resource name or skip reason
	since    string // provider version that added support (covered only)
	priority int    // 1=high, 2=medium, 3=low (planned only)
	notes    string // human-readable context
}

// contractRoutePin is the pin contract whose routes[] define the coverage inventory.
const contractRoutePin = "coolify-v4.json"

// registryExtrasNotInContract are registry keys kept for historical OpenAPI-only
// aliases that Coolify no longer lists as first-class routes (or never did in
// the extract). Completeness still requires them to be classified.
var registryExtrasNotInContract = map[string]bool{
	// Deprecated Coolify alias: creates a Service, not an Application.
	"POST /applications/dockercompose": true,
}

// coveredEndpoints returns the provider's route coverage registry.
//
// Route inventory for completeness is the source-extracted contract
// (testdata/contracts/coolify-v4.json routes[]), not the partial OpenAPI
// path list. Field-level truth is the same contract. OpenAPI under
// testdata/specs/ is not authoritative for routes.
//
// TestSpecCoverage_Completeness fails if contract routes and this registry
// drift. TestSpecCoverage_GenerateDoc generates API_COVERAGE.md from this data.
func coveredEndpoints() map[string]coverageStatus {
	covered := func(resource, since string) coverageStatus {
		return coverageStatus{category: "covered", resource: resource, since: since}
	}
	skipped := func(reason string) coverageStatus {
		return coverageStatus{category: "skipped", resource: reason}
	}
	planned := func(priority int, notes string) coverageStatus {
		return coverageStatus{category: "planned", priority: priority, notes: notes}
	}

	return map[string]coverageStatus{
		// ── Projects ──
		"GET /projects":                      covered("data.coolify_projects", "v0.1.0"),
		"POST /projects":                     covered("coolify_project", "v0.1.0"),
		"GET /projects/{uuid}":               covered("data.coolify_project", "v0.1.0"),
		"PATCH /projects/{uuid}":             covered("coolify_project", "v0.1.0"),
		"DELETE /projects/{uuid}":            covered("coolify_project", "v0.1.0"),
		"GET /projects/{uuid}/environments":  covered("data.coolify_environments", "v0.2.0"),
		"POST /projects/{uuid}/environments": covered("coolify_environment", "v0.2.0"),
		"DELETE /projects/{uuid}/environments/{environment_name_or_uuid}": covered("coolify_environment", "v0.2.0"),
		"GET /projects/{uuid}/{environment_name_or_uuid}":                 covered("data.coolify_environment", "v0.2.0"),

		// ── Servers ──
		"GET /servers":                  covered("data.coolify_servers", "v0.1.0"),
		"POST /servers":                 covered("coolify_server", "v0.1.0"),
		"GET /servers/{uuid}":           covered("data.coolify_server", "v0.1.0"),
		"PATCH /servers/{uuid}":         covered("coolify_server", "v0.1.0"),
		"DELETE /servers/{uuid}":        covered("coolify_server", "v0.1.0"),
		"GET /servers/{uuid}/domains":   covered("data.coolify_server_domains", "v0.1.0"),
		"GET /servers/{uuid}/resources": covered("data.coolify_server_resources", "v0.1.0"),
		"POST /servers/hetzner":         covered("coolify_server_hetzner", "v0.2.0"),
		"GET /servers/{uuid}/validate":  covered("data.coolify_server_validation", "v0.2.0"),

		// ── Applications ──
		"GET /applications":                                               covered("data.coolify_applications", "v0.1.0"),
		"POST /applications/public":                                       covered("coolify_application", "v0.1.0"),
		"POST /applications/dockercompose":                                {category: "skipped", resource: "Deprecated alias: use POST /services instead because this flow creates a Service, not an Application"},
		"POST /applications/dockerimage":                                  covered("coolify_application_docker_image", "v0.1.0"),
		"POST /applications/private-deploy-key":                           covered("coolify_application_private_git", "v0.1.0"),
		"GET /applications/{uuid}":                                        covered("data.coolify_application", "v0.1.0"),
		"PATCH /applications/{uuid}":                                      covered("coolify_application + variants", "v0.1.0"),
		"DELETE /applications/{uuid}":                                     covered("coolify_application + variants", "v0.1.0"),
		"GET /applications/{uuid}/envs":                                   covered("data.coolify_environment_variables", "v0.1.0"),
		"POST /applications/{uuid}/envs":                                  covered("coolify_environment_variable", "v0.1.0"),
		"PATCH /applications/{uuid}/envs":                                 covered("coolify_environment_variable", "v0.1.0"),
		"DELETE /applications/{uuid}/envs/{env_uuid}":                     covered("coolify_environment_variable", "v0.1.0"),
		"GET /applications/{uuid}/restart":                                covered("coolify_deployment", "v0.1.0"),
		"POST /applications/dockerfile":                                   covered("coolify_application_dockerfile", "v0.2.0"),
		"POST /applications/private-github-app":                           covered("coolify_application_github_app", "v0.2.0"),
		"GET /applications/{uuid}/start":                                  covered("coolify_resource_action", "v0.3.0"),
		"GET /applications/{uuid}/stop":                                   covered("coolify_resource_action", "v0.3.0"),
		"GET /applications/{uuid}/scheduled-tasks":                        covered("data.coolify_scheduled_tasks", "v0.2.0"),
		"POST /applications/{uuid}/scheduled-tasks":                       covered("coolify_scheduled_task", "v0.2.0"),
		"PATCH /applications/{uuid}/scheduled-tasks/{task_uuid}":          covered("coolify_scheduled_task", "v0.2.0"),
		"DELETE /applications/{uuid}/scheduled-tasks/{task_uuid}":         covered("coolify_scheduled_task", "v0.2.0"),
		"GET /applications/{uuid}/scheduled-tasks/{task_uuid}/executions": covered("data.coolify_task_executions", "v0.2.0"),
		"GET /applications/{uuid}/storages":                               covered("data.coolify_storages", "v0.2.0"),
		"POST /applications/{uuid}/storages":                              covered("coolify_storage", "v0.2.0"),
		"PATCH /applications/{uuid}/storages":                             covered("coolify_storage", "v0.2.0"),
		"DELETE /applications/{uuid}/storages/{storage_uuid}":             covered("coolify_storage", "v0.2.0"),
		"PUT /applications/{uuid}/storages/{storage_uuid}/backups":        covered("coolify_storage_backup", "v0.1.9"),
		"DELETE /applications/{uuid}/storages/{storage_uuid}/backups":     covered("coolify_storage_backup", "v0.1.9"),
		"PATCH /applications/{uuid}/envs/bulk":                            covered("client.BulkUpdateEnvVars", "v0.2.0"),
		"GET /applications/{uuid}/logs":                                   covered("data.coolify_application_logs", "v0.2.0"),
		"DELETE /applications/{uuid}/previews/{pull_request_id}":          covered("client.DeletePreviewDeployment", "v0.2.0"),

		// ── Databases ──
		"GET /databases":                                                   covered("data.coolify_databases", "v0.1.0"),
		"POST /databases/postgresql":                                       covered("coolify_database_postgresql", "v0.1.0"),
		"POST /databases/mysql":                                            covered("coolify_database_mysql", "v0.1.0"),
		"POST /databases/mariadb":                                          covered("coolify_database_mariadb", "v0.1.0"),
		"POST /databases/mongodb":                                          covered("coolify_database_mongodb", "v0.1.0"),
		"POST /databases/redis":                                            covered("coolify_database_redis", "v0.1.0"),
		"POST /databases/clickhouse":                                       covered("coolify_database_clickhouse", "v0.1.0"),
		"POST /databases/keydb":                                            covered("coolify_database_keydb", "v0.1.0"),
		"POST /databases/dragonfly":                                        covered("coolify_database_dragonfly", "v0.1.0"),
		"GET /databases/{uuid}":                                            covered("data.coolify_database", "v0.1.0"),
		"PATCH /databases/{uuid}":                                          covered("coolify_*_database", "v0.1.0"),
		"DELETE /databases/{uuid}":                                         covered("coolify_*_database", "v0.1.0"),
		"GET /databases/{uuid}/backups":                                    covered("coolify_database_backup", "v0.1.0"),
		"POST /databases/{uuid}/backups":                                   covered("coolify_database_backup", "v0.1.0"),
		"PATCH /databases/{uuid}/backups/{scheduled_backup_uuid}":          covered("coolify_database_backup", "v0.1.0"),
		"DELETE /databases/{uuid}/backups/{scheduled_backup_uuid}":         covered("coolify_database_backup", "v0.1.0"),
		"GET /databases/{uuid}/envs":                                       covered("data.coolify_environment_variables", "v0.2.0"),
		"POST /databases/{uuid}/envs":                                      covered("coolify_environment_variable", "v0.2.0"),
		"PATCH /databases/{uuid}/envs":                                     covered("coolify_environment_variable", "v0.2.0"),
		"DELETE /databases/{uuid}/envs/{env_uuid}":                         covered("coolify_environment_variable", "v0.2.0"),
		"GET /databases/{uuid}/storages":                                   covered("data.coolify_storages", "v0.2.0"),
		"POST /databases/{uuid}/storages":                                  covered("coolify_storage", "v0.2.0"),
		"PATCH /databases/{uuid}/storages":                                 covered("coolify_storage", "v0.2.0"),
		"DELETE /databases/{uuid}/storages/{storage_uuid}":                 covered("coolify_storage", "v0.2.0"),
		"PUT /databases/{uuid}/storages/{storage_uuid}/backups":            covered("coolify_storage_backup", "v0.1.9"),
		"DELETE /databases/{uuid}/storages/{storage_uuid}/backups":         covered("coolify_storage_backup", "v0.1.9"),
		"GET /databases/{uuid}/backups/{scheduled_backup_uuid}/executions": covered("data.coolify_backup_executions", "v0.2.0"),
		"DELETE /databases/{uuid}/backups/{scheduled_backup_uuid}/executions/{execution_uuid}": covered("client.DeleteBackupExecution", "v0.2.0"),
		"PATCH /databases/{uuid}/envs/bulk":                                                    covered("client.BulkUpdateEnvVars", "v0.2.0"),
		"GET /databases/{uuid}/restart":                                                        covered("coolify_resource_action", "v0.3.0"),
		"GET /databases/{uuid}/start":                                                          covered("coolify_resource_action", "v0.3.0"),
		"GET /databases/{uuid}/stop":                                                           covered("coolify_resource_action", "v0.3.0"),

		// ── Services ──
		"GET /services":                                               covered("data.coolify_services", "v0.1.0"),
		"POST /services":                                              covered("coolify_service", "v0.1.0"),
		"GET /services/{uuid}":                                        covered("data.coolify_service", "v0.1.0"),
		"PATCH /services/{uuid}":                                      covered("coolify_service", "v0.1.0"),
		"DELETE /services/{uuid}":                                     covered("coolify_service", "v0.1.0"),
		"POST /services/{uuid}/envs":                                  covered("coolify_environment_variable", "v0.1.0"),
		"PATCH /services/{uuid}/envs":                                 covered("coolify_environment_variable", "v0.1.0"),
		"DELETE /services/{uuid}/envs/{env_uuid}":                     covered("coolify_environment_variable", "v0.1.0"),
		"GET /services/{uuid}/envs":                                   covered("data.coolify_environment_variables", "v0.2.0"),
		"GET /services/{uuid}/scheduled-tasks":                        covered("data.coolify_scheduled_tasks", "v0.2.0"),
		"POST /services/{uuid}/scheduled-tasks":                       covered("coolify_scheduled_task", "v0.2.0"),
		"PATCH /services/{uuid}/scheduled-tasks/{task_uuid}":          covered("coolify_scheduled_task", "v0.2.0"),
		"DELETE /services/{uuid}/scheduled-tasks/{task_uuid}":         covered("coolify_scheduled_task", "v0.2.0"),
		"GET /services/{uuid}/scheduled-tasks/{task_uuid}/executions": covered("data.coolify_task_executions", "v0.2.0"),
		"GET /services/{uuid}/storages":                               covered("data.coolify_storages", "v0.2.0"),
		"POST /services/{uuid}/storages":                              covered("coolify_storage", "v0.2.0"),
		"PATCH /services/{uuid}/storages":                             covered("coolify_storage", "v0.2.0"),
		"DELETE /services/{uuid}/storages/{storage_uuid}":             covered("coolify_storage", "v0.2.0"),
		"PUT /services/{uuid}/storages/{storage_uuid}/backups":        covered("coolify_storage_backup", "v0.1.9"),
		"DELETE /services/{uuid}/storages/{storage_uuid}/backups":     covered("coolify_storage_backup", "v0.1.9"),
		"PATCH /services/{uuid}/envs/bulk":                            covered("client.BulkUpdateEnvVars", "v0.2.0"),
		"GET /services/{uuid}/restart":                                covered("coolify_resource_action", "v0.3.0"),
		"GET /services/{uuid}/start":                                  covered("coolify_resource_action", "v0.3.0"),
		"GET /services/{uuid}/stop":                                   covered("coolify_resource_action", "v0.3.0"),

		// ── Security Keys ──
		"GET /security/keys":           covered("data.coolify_private_keys", "v0.1.0"),
		"POST /security/keys":          covered("coolify_private_key", "v0.1.0"),
		"PATCH /security/keys/{uuid}":  covered("coolify_private_key", "v0.1.0"),
		"GET /security/keys/{uuid}":    covered("data.coolify_private_key", "v0.1.0"),
		"DELETE /security/keys/{uuid}": covered("coolify_private_key", "v0.1.0"),

		// ── Deployments ──
		"GET /deployments/{uuid}":              covered("coolify_deployment", "v0.1.0"),
		"GET /deployments":                     covered("data.coolify_deployments", "v0.2.0"),
		"GET /deployments/applications/{uuid}": covered("data.coolify_deployments", "v0.2.0"),
		"POST /deployments/{uuid}/cancel":      covered("client.CancelDeployment", "v0.2.0"),
		"GET /deploy":                          covered("client.Deploy", "v0.2.0"),

		// ── Teams ──
		"GET /teams/{id}":            covered("data.coolify_team", "v0.1.0"),
		"GET /teams":                 covered("data.coolify_teams", "v0.2.0"),
		"GET /teams/{id}/members":    covered("data.coolify_team_members", "v0.2.0"),
		"GET /teams/current":         covered("data.coolify_team_members", "v0.2.0"),
		"GET /teams/current/members": covered("data.coolify_team_members", "v0.2.0"),

		// ── Cloud Tokens ──
		"GET /cloud-tokens":                  covered("data.coolify_cloud_tokens", "v0.2.0"),
		"POST /cloud-tokens":                 covered("coolify_cloud_token", "v0.2.0"),
		"GET /cloud-tokens/{uuid}":           covered("data.coolify_cloud_token", "v0.2.0"),
		"PATCH /cloud-tokens/{uuid}":         covered("coolify_cloud_token", "v0.2.0"),
		"DELETE /cloud-tokens/{uuid}":        covered("coolify_cloud_token", "v0.2.0"),
		"POST /cloud-tokens/{uuid}/validate": covered("client.ValidateCloudToken", "v0.2.0"),

		// ── GitHub Apps ──
		"GET /github-apps":                                                      covered("data.coolify_github_apps", "v0.2.0"),
		"POST /github-apps":                                                     covered("coolify_github_app", "v0.2.0"),
		"PATCH /github-apps/{github_app_id}":                                    covered("coolify_github_app", "v0.2.0"),
		"DELETE /github-apps/{github_app_id}":                                   covered("coolify_github_app", "v0.2.0"),
		"GET /github-apps/{github_app_id}/repositories":                         covered("data.coolify_github_app_repositories", "v0.2.0"),
		"GET /github-apps/{github_app_id}/repositories/{owner}/{repo}/branches": covered("data.coolify_github_app_branches", "v0.2.0"),

		// ── Hetzner ──
		"GET /hetzner/images":       covered("data.coolify_hetzner_images", "v0.2.0"),
		"GET /hetzner/locations":    covered("data.coolify_hetzner_locations", "v0.2.0"),
		"GET /hetzner/server-types": covered("data.coolify_hetzner_server_types", "v0.2.0"),
		"GET /hetzner/ssh-keys":     covered("data.coolify_hetzner_ssh_keys", "v0.2.0"),

		// ── Operational / Meta ──
		"GET /version":      covered("data.coolify_version", "v0.1.0"),
		"GET /resources":    covered("data.coolify_resources", "v0.2.0"),
		"GET /health":       covered("data.coolify_health", "v0.2.0"),
		"GET /enable":       covered("client.EnableAPI", "v0.2.0"),
		"GET /disable":      covered("client.DisableAPI", "v0.2.0"),
		"POST /mcp/enable":  covered("coolify_api_settings (mcp_enabled)", "v0.4.0"),
		"POST /mcp/disable": covered("coolify_api_settings (mcp_enabled)", "v0.4.0"),

		// ── Contract routes not present in legacy OpenAPI inventory ──
		// (classified so API_COVERAGE reflects Coolify source, not OpenAPI)
		"DELETE /applications/{uuid}/destinations/{destination_uuid}":                   skipped("Application multi-destination attach/detach; primary TF surface is coolify_destination"),
		"DELETE /applications/{uuid}/tags/{tag_uuid}":                                   skipped("Resource tags API; not modeled as first-class TF resources"),
		"DELETE /cloud-init-scripts/{uuid}":                                             skipped("Cloud-init script library; not a TF resource"),
		"DELETE /databases/{uuid}/tags/{tag_uuid}":                                      skipped("Resource tags API; not modeled as first-class TF resources"),
		"DELETE /destinations/{uuid}":                                                   covered("coolify_destination", "v0.2.0"),
		"DELETE /gitlab-apps/{gitlab_app_id}":                                           skipped("GitLab App integration; provider covers GitHub Apps only today"),
		"DELETE /projects/{uuid}/environments/{environment_name_or_uuid}/envs/{env_id}": skipped("Shared/project/server/team env vars; provider models resource-scoped envs (app/db/service)"),
		"DELETE /projects/{uuid}/envs/{env_id}":                                         skipped("Shared/project/server/team env vars; provider models resource-scoped envs (app/db/service)"),
		"DELETE /s3-storages/{uuid}":                                                    covered("coolify_s3_storage", "v0.1.13"),
		"DELETE /servers/{uuid}/envs/{env_id}":                                          skipped("Shared/project/server/team env vars; provider models resource-scoped envs (app/db/service)"),
		"DELETE /services/{uuid}/tags/{tag_uuid}":                                       skipped("Resource tags API; not modeled as first-class TF resources"),
		"DELETE /tags/{uuid}":                                                           skipped("Resource tags API; not modeled as first-class TF resources"),
		"DELETE /team/envs/{env_id}":                                                    skipped("Shared/project/server/team env vars; provider models resource-scoped envs (app/db/service)"),
		"GET /applications/{uuid}/destinations":                                         skipped("Application multi-destination attach/detach; primary TF surface is coolify_destination"),
		"GET /applications/{uuid}/rollback-images":                                      skipped("Application rollback/images; operational, not TF lifecycle"),
		"GET /applications/{uuid}/tags":                                                 skipped("Resource tags API; not modeled as first-class TF resources"),
		"GET /cloud-init-scripts":                                                       skipped("Cloud-init script library; not a TF resource"),
		"GET /cloud-init-scripts/{uuid}":                                                skipped("Cloud-init script library; not a TF resource"),
		"GET /databases/{uuid}/logs":                                                    skipped("Resource logs streaming; not durable TF state"),
		"GET /databases/{uuid}/tags":                                                    skipped("Resource tags API; not modeled as first-class TF resources"),
		"GET /destinations":                                                             covered("data.coolify_destinations", "v0.2.0"),
		"GET /destinations/{uuid}":                                                      covered("data.coolify_destination", "v0.2.0"),
		"GET /digitalocean/images":                                                      covered("data.coolify_digitalocean_images", "v0.2.0"),
		"GET /digitalocean/regions":                                                     covered("data.coolify_digitalocean_regions", "v0.2.0"),
		"GET /digitalocean/sizes":                                                       covered("data.coolify_digitalocean_sizes", "v0.2.0"),
		"GET /digitalocean/ssh-keys":                                                    covered("data.coolify_digitalocean_ssh_keys", "v0.2.0"),
		"GET /gitlab-apps":                                                              skipped("GitLab App integration; provider covers GitHub Apps only today"),
		"GET /hetzner/firewalls":                                                        skipped("Hetzner firewalls/networks list; not required for coolify_server_hetzner"),
		"GET /hetzner/networks":                                                         skipped("Hetzner firewalls/networks list; not required for coolify_server_hetzner"),
		"GET /notifications/discord":                                                    planned(2, "Notification channel resources (#394)"),
		"GET /notifications/email":                                                      planned(2, "Notification channel resources (#394)"),
		"GET /notifications/pushover":                                                   planned(2, "Notification channel resources (#394)"),
		"GET /notifications/slack":                                                      planned(2, "Notification channel resources (#394)"),
		"GET /notifications/telegram":                                                   planned(2, "Notification channel resources (#394)"),
		"GET /notifications/webhook":                                                    planned(2, "Notification channel resources (#394)"),
		"GET /projects/{uuid}/environments/{environment_name_or_uuid}/envs":             skipped("Shared/project/server/team env vars; provider models resource-scoped envs (app/db/service)"),
		"GET /projects/{uuid}/envs":                                                     skipped("Shared/project/server/team env vars; provider models resource-scoped envs (app/db/service)"),
		"GET /s3-storages":                                                              covered("data.coolify_s3_storages", "v0.1.13"),
		"GET /s3-storages/{uuid}":                                                       covered("data.coolify_s3_storage", "v0.1.13"),
		"GET /servers/{server_uuid}/destinations":                                       covered("data.coolify_destinations", "v0.2.0"),
		"GET /servers/{uuid}/cloudflare-tunnel":                                         skipped("Server operational/control-plane API; not modeled as TF resources"),
		"GET /servers/{uuid}/docker-cleanup":                                            skipped("Server operational/control-plane API; not modeled as TF resources"),
		"GET /servers/{uuid}/docker-cleanup/executions":                                 skipped("Server operational/control-plane API; not modeled as TF resources"),
		"GET /servers/{uuid}/envs":                                                      skipped("Shared/project/server/team env vars; provider models resource-scoped envs (app/db/service)"),
		"GET /servers/{uuid}/export":                                                    skipped("Server operational/control-plane API; not modeled as TF resources"),
		"GET /servers/{uuid}/log-drains":                                                skipped("Server operational/control-plane API; not modeled as TF resources"),
		"GET /servers/{uuid}/proxy":                                                     skipped("Server operational/control-plane API; not modeled as TF resources"),
		"GET /servers/{uuid}/sentinel":                                                  skipped("Server operational/control-plane API; not modeled as TF resources"),
		"GET /services/{uuid}/applications":                                             skipped("Nested service components; coolify_service manages the compose service unit"),
		"GET /services/{uuid}/applications/{app_uuid}":                                  skipped("Nested service components; coolify_service manages the compose service unit"),
		"GET /services/{uuid}/applications/{app_uuid}/logs":                             skipped("Resource logs streaming; not durable TF state"),
		"GET /services/{uuid}/applications/{app_uuid}/restart":                          skipped("Nested service components; coolify_service manages the compose service unit"),
		"GET /services/{uuid}/applications/{app_uuid}/start":                            skipped("Nested service components; coolify_service manages the compose service unit"),
		"GET /services/{uuid}/applications/{app_uuid}/stop":                             skipped("Nested service components; coolify_service manages the compose service unit"),
		"GET /services/{uuid}/databases":                                                skipped("Nested service components; coolify_service manages the compose service unit"),
		"GET /services/{uuid}/databases/{database_uuid}":                                skipped("Nested service components; coolify_service manages the compose service unit"),
		"GET /services/{uuid}/databases/{database_uuid}/logs":                           skipped("Resource logs streaming; not durable TF state"),
		"GET /services/{uuid}/logs":                                                     skipped("Resource logs streaming; not durable TF state"),
		"GET /services/{uuid}/tags":                                                     skipped("Resource tags API; not modeled as first-class TF resources"),
		"GET /tags":                                                                     skipped("Resource tags API; not modeled as first-class TF resources"),
		"GET /team":                                                                     skipped("Alias of /teams/current*; use data.coolify_team / team_members"),
		"GET /team/envs":                                                                skipped("Shared/project/server/team env vars; provider models resource-scoped envs (app/db/service)"),
		"GET /team/members":                                                             skipped("Alias of /teams/current*; use data.coolify_team / team_members"),
		"GET /vultr/os":                                                                 covered("data.coolify_vultr_os", "v0.2.0"),
		"GET /vultr/plans":                                                              covered("data.coolify_vultr_plans", "v0.2.0"),
		"GET /vultr/regions":                                                            covered("data.coolify_vultr_regions", "v0.2.0"),
		"GET /vultr/ssh-keys":                                                           covered("data.coolify_vultr_ssh_keys", "v0.2.0"),
		"PATCH /cloud-init-scripts/{uuid}":                                              skipped("Cloud-init script library; not a TF resource"),
		"PATCH /destinations/{uuid}":                                                    skipped("Destination fields RequireReplace; provider does not call PATCH"),
		"PATCH /gitlab-apps/{gitlab_app_id}":                                            skipped("GitLab App integration; provider covers GitHub Apps only today"),
		"PATCH /notifications/discord":                                                  planned(2, "Notification channel resources (#394)"),
		"PATCH /notifications/email":                                                    planned(2, "Notification channel resources (#394)"),
		"PATCH /notifications/pushover":                                                 planned(2, "Notification channel resources (#394)"),
		"PATCH /notifications/slack":                                                    planned(2, "Notification channel resources (#394)"),
		"PATCH /notifications/telegram":                                                 planned(2, "Notification channel resources (#394)"),
		"PATCH /notifications/webhook":                                                  planned(2, "Notification channel resources (#394)"),
		"PATCH /projects/{uuid}/environments/{environment_name_or_uuid}":                skipped("Environment description is TF state-only today; client has no Update call"),
		"PATCH /projects/{uuid}/environments/{environment_name_or_uuid}/envs/{env_id}": skipped("Shared/project/server/team env vars; provider models resource-scoped envs (app/db/service)"),
		"PATCH /projects/{uuid}/envs/{env_id}":                                         skipped("Shared/project/server/team env vars; provider models resource-scoped envs (app/db/service)"),
		"PATCH /s3-storages/{uuid}":                                                    covered("coolify_s3_storage", "v0.1.13"),
		"PATCH /servers/{uuid}/cloudflare-tunnel":                                      skipped("Server operational/control-plane API; not modeled as TF resources"),
		"PATCH /servers/{uuid}/docker-cleanup":                                         skipped("Server operational/control-plane API; not modeled as TF resources"),
		"PATCH /servers/{uuid}/envs/{env_id}":                                          skipped("Shared/project/server/team env vars; provider models resource-scoped envs (app/db/service)"),
		"PATCH /servers/{uuid}/log-drains":                                             skipped("Server operational/control-plane API; not modeled as TF resources"),
		"PATCH /servers/{uuid}/proxy":                                                  skipped("Server operational/control-plane API; not modeled as TF resources"),
		"PATCH /servers/{uuid}/sentinel":                                               skipped("Server operational/control-plane API; not modeled as TF resources"),
		"PATCH /services/{uuid}/applications/{app_uuid}":                               skipped("Nested service components; coolify_service manages the compose service unit"),
		"PATCH /services/{uuid}/databases/{database_uuid}":                             skipped("Nested service components; coolify_service manages the compose service unit"),
		"PATCH /tags/{uuid}":                                                           skipped("Resource tags API; not modeled as first-class TF resources"),
		"PATCH /team/envs/{env_id}":                                                    skipped("Shared/project/server/team env vars; provider models resource-scoped envs (app/db/service)"),
		"POST /applications/{uuid}/clone":                                              skipped("One-shot clone/migrate/move; not Terraform lifecycle"),
		"POST /applications/{uuid}/destinations":                                       skipped("Application multi-destination attach/detach; primary TF surface is coolify_destination"),
		"POST /applications/{uuid}/migrate":                                            skipped("One-shot clone/migrate/move; not Terraform lifecycle"),
		"POST /applications/{uuid}/move":                                               skipped("One-shot clone/migrate/move; not Terraform lifecycle"),
		"POST /applications/{uuid}/restart":                                            covered("coolify_resource_action", "v0.3.0"),
		"POST /applications/{uuid}/rollback":                                           skipped("Application rollback/images; operational, not TF lifecycle"),
		"POST /applications/{uuid}/scheduled-tasks/{task_uuid}/execute":                skipped("One-shot task execute; coolify_scheduled_task manages definition only"),
		"POST /applications/{uuid}/start":                                              covered("coolify_resource_action", "v0.3.0"),
		"POST /applications/{uuid}/stop":                                               covered("coolify_resource_action", "v0.3.0"),
		"POST /applications/{uuid}/storages/{storage_uuid}/backups/run":                skipped("One-shot volume backup run; coolify_storage_backup manages schedule only"),
		"POST /applications/{uuid}/tags":                                               skipped("Resource tags API; not modeled as first-class TF resources"),
		"POST /cloud-init-scripts":                                                     skipped("Cloud-init script library; not a TF resource"),
		"POST /databases/{uuid}/clone":                                                 skipped("One-shot clone/migrate/move; not Terraform lifecycle"),
		"POST /databases/{uuid}/migrate":                                               skipped("One-shot clone/migrate/move; not Terraform lifecycle"),
		"POST /databases/{uuid}/move":                                                  skipped("One-shot clone/migrate/move; not Terraform lifecycle"),
		"POST /databases/{uuid}/restart":                                               covered("coolify_resource_action", "v0.3.0"),
		"POST /databases/{uuid}/start":                                                 covered("coolify_resource_action", "v0.3.0"),
		"POST /databases/{uuid}/stop":                                                  covered("coolify_resource_action", "v0.3.0"),
		"POST /databases/{uuid}/storages/{storage_uuid}/backups/run":                   skipped("One-shot volume backup run; coolify_storage_backup manages schedule only"),
		"POST /databases/{uuid}/tags":                                                  skipped("Resource tags API; not modeled as first-class TF resources"),
		"POST /deploy":                                                                 covered("client.Deploy", "v0.2.0"),
		"POST /disable":                                                                covered("coolify_api_settings", "v0.2.0"),
		"POST /enable":                                                                 covered("coolify_api_settings", "v0.2.0"),
		"POST /feedback":                                                               skipped("Coolify product feedback endpoint; not TF"),
		"POST /gitlab-apps":                                                            skipped("GitLab App integration; provider covers GitHub Apps only today"),
		"POST /projects/{uuid}/environments/{environment_name_or_uuid}/envs":           skipped("Shared/project/server/team env vars; provider models resource-scoped envs (app/db/service)"),
		"POST /projects/{uuid}/envs":                                                   skipped("Shared/project/server/team env vars; provider models resource-scoped envs (app/db/service)"),
		"POST /s3-storages":                                                            covered("coolify_s3_storage", "v0.1.13"),
		"POST /s3-storages/{uuid}/validate":                                            covered("coolify_s3_storage_validate", "v0.1.14"),
		"POST /sentinel/push":                                                          skipped("Server operational/control-plane API; not modeled as TF resources"),
		"POST /servers/digitalocean":                                                   covered("coolify_server_digitalocean", "v0.2.0"),
		"POST /servers/import":                                                         skipped("Server operational/control-plane API; not modeled as TF resources"),
		"POST /servers/vultr":                                                          covered("coolify_server_vultr", "v0.2.0"),
		"POST /servers/{server_uuid}/destinations":                                     covered("coolify_destination", "v0.2.0"),
		"POST /servers/{uuid}/claim":                                                   skipped("Server operational/control-plane API; not modeled as TF resources"),
		"POST /servers/{uuid}/cloudflare-tunnel/disable":                               skipped("Server operational/control-plane API; not modeled as TF resources"),
		"POST /servers/{uuid}/cloudflare-tunnel/enable":                                skipped("Server operational/control-plane API; not modeled as TF resources"),
		"POST /servers/{uuid}/docker-cleanup/run":                                      skipped("Server operational/control-plane API; not modeled as TF resources"),
		"POST /servers/{uuid}/envs":                                                    skipped("Shared/project/server/team env vars; provider models resource-scoped envs (app/db/service)"),
		"POST /servers/{uuid}/export/mailbox":                                          skipped("Server operational/control-plane API; not modeled as TF resources"),
		"POST /servers/{uuid}/migrate":                                                 skipped("One-shot clone/migrate/move; not Terraform lifecycle"),
		"POST /servers/{uuid}/proxy/restart":                                           skipped("Server operational/control-plane API; not modeled as TF resources"),
		"POST /servers/{uuid}/transfer/complete":                                       skipped("Server operational/control-plane API; not modeled as TF resources"),
		"POST /servers/{uuid}/validate":                                                covered("coolify_server_validate", "v0.2.0"),
		"POST /services/{uuid}/applications/{app_uuid}/logs":                           skipped("Resource logs streaming; not durable TF state"),
		"POST /services/{uuid}/applications/{app_uuid}/restart":                        skipped("Nested service app/DB lifecycle; manage via parent coolify_service"),
		"POST /services/{uuid}/applications/{app_uuid}/start":                          skipped("Nested service app/DB lifecycle; manage via parent coolify_service"),
		"POST /services/{uuid}/applications/{app_uuid}/stop":                           skipped("Nested service app/DB lifecycle; manage via parent coolify_service"),
		"POST /services/{uuid}/clone":                                                  skipped("One-shot clone/migrate/move; not Terraform lifecycle"),
		"POST /services/{uuid}/databases/{database_uuid}/restart":                      skipped("Nested service app/DB lifecycle; manage via parent coolify_service"),
		"POST /services/{uuid}/databases/{database_uuid}/start":                        skipped("Nested service app/DB lifecycle; manage via parent coolify_service"),
		"POST /services/{uuid}/databases/{database_uuid}/stop":                         skipped("Nested service app/DB lifecycle; manage via parent coolify_service"),
		"POST /services/{uuid}/migrate":                                                skipped("One-shot clone/migrate/move; not Terraform lifecycle"),
		"POST /services/{uuid}/move":                                                   skipped("One-shot clone/migrate/move; not Terraform lifecycle"),
		"POST /services/{uuid}/restart":                                                covered("coolify_resource_action", "v0.3.0"),
		"POST /services/{uuid}/scheduled-tasks/{task_uuid}/execute":                    skipped("One-shot task execute; coolify_scheduled_task manages definition only"),
		"POST /services/{uuid}/start":                                                  covered("coolify_resource_action", "v0.3.0"),
		"POST /services/{uuid}/stop":                                                   covered("coolify_resource_action", "v0.3.0"),
		"POST /services/{uuid}/storages/{storage_uuid}/backups/run":                    skipped("One-shot volume backup run; coolify_storage_backup manages schedule only"),
		"POST /services/{uuid}/tags":                                                   skipped("Resource tags API; not modeled as first-class TF resources"),
		"POST /tags":                                                                   skipped("Resource tags API; not modeled as first-class TF resources"),
		"POST /team/envs":                                                              skipped("Shared/project/server/team env vars; provider models resource-scoped envs (app/db/service)"),
		"PUT /servers/{uuid}/proxy/configuration":                                      skipped("Server operational/control-plane API; not modeled as TF resources"),
	}
}

// TestSpecCoverage_Completeness verifies every contract route is classified in
// coveredEndpoints(), and the registry has no unexpected keys outside the
// contract (except registryExtrasNotInContract).
func TestSpecCoverage_Completeness(t *testing.T) {
	t.Parallel()

	coverage := coveredEndpoints()
	contractOps, err := loadContractRoutes(contractRoutePin)
	if err != nil {
		t.Fatalf("loading contract routes: %v", err)
	}

	var missing []string
	for _, op := range contractOps {
		if _, ok := coverage[op]; !ok {
			missing = append(missing, op)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		t.Errorf("contract has %d routes not tracked in coveredEndpoints():\n  %s",
			len(missing), strings.Join(missing, "\n  "))
	}

	contractSet := make(map[string]bool, len(contractOps))
	for _, op := range contractOps {
		contractSet[op] = true
	}
	var stale []string
	for op := range coverage {
		if contractSet[op] || registryExtrasNotInContract[op] {
			continue
		}
		stale = append(stale, op)
	}
	if len(stale) > 0 {
		sort.Strings(stale)
		t.Errorf("coveredEndpoints() has %d entries not in the contract and not allowlisted:\n  %s",
			len(stale), strings.Join(stale, "\n  "))
	}
}

// TestLoadContractRoutes_Pin loads the pin and expects a non-empty route set.
func TestLoadContractRoutes_Pin(t *testing.T) {
	t.Parallel()
	ops, err := loadContractRoutes(contractRoutePin)
	if err != nil {
		t.Fatal(err)
	}
	if len(ops) < 100 {
		t.Fatalf("expected many contract routes, got %d", len(ops))
	}
	// Sanity: known Coolify routes from source extract
	want := []string{
		"GET /s3-storages",
		"POST /s3-storages/{uuid}/validate",
		"POST /servers/{uuid}/validate",
		"GET /destinations",
	}
	set := make(map[string]bool, len(ops))
	for _, o := range ops {
		set[o] = true
	}
	for _, w := range want {
		if !set[w] {
			t.Errorf("pin contract missing expected route %s", w)
		}
	}
}

// TestSpecCoverage_Report prints a coverage summary. Run with -v to see it.
func TestSpecCoverage_Report(t *testing.T) {
	t.Parallel()

	coverage := coveredEndpoints()
	var coveredN, plannedN, skippedN int
	for _, s := range coverage {
		switch s.category {
		case "covered":
			coveredN++
		case "planned":
			plannedN++
		case "skipped":
			skippedN++
		}
	}

	total := coveredN + plannedN + skippedN
	pct := float64(coveredN) / float64(total) * 100

	t.Logf("\n=== API Coverage Report ===")
	t.Logf("Total endpoints: %d", total)
	t.Logf("Covered:         %d (%.1f%%)", coveredN, pct)
	t.Logf("Planned:         %d", plannedN)
	t.Logf("Skipped:         %d", skippedN)
}

// TestSpecCoverage_GenerateDoc generates API_COVERAGE.md at the repo root.
// Run: go test ./internal/spectest/ -run TestSpecCoverage_GenerateDoc -v
func TestSpecCoverage_GenerateDoc(t *testing.T) {
	if os.Getenv("GENERATE_COVERAGE_DOC") == "" {
		t.Skip("set GENERATE_COVERAGE_DOC=1 to regenerate API_COVERAGE.md")
	}

	coverage := coveredEndpoints()

	type entry struct {
		endpoint string
		status   coverageStatus
	}

	var coveredList, plannedList, skippedList []entry
	for ep, s := range coverage {
		e := entry{endpoint: ep, status: s}
		switch s.category {
		case "covered":
			coveredList = append(coveredList, e)
		case "planned":
			plannedList = append(plannedList, e)
		case "skipped":
			skippedList = append(skippedList, e)
		}
	}

	sort.Slice(coveredList, func(i, j int) bool {
		return coveredList[i].endpoint < coveredList[j].endpoint
	})
	sort.Slice(plannedList, func(i, j int) bool {
		if plannedList[i].status.priority != plannedList[j].status.priority {
			return plannedList[i].status.priority < plannedList[j].status.priority
		}
		return plannedList[i].endpoint < plannedList[j].endpoint
	})
	sort.Slice(skippedList, func(i, j int) bool {
		return skippedList[i].endpoint < skippedList[j].endpoint
	})

	total := len(coveredList) + len(plannedList) + len(skippedList)
	pct := float64(len(coveredList)) / float64(total) * 100

	var b strings.Builder
	b.WriteString("# API Coverage\n\n")
	b.WriteString("<!-- Auto-generated from internal/spectest/coverage_test.go. Do not edit manually. -->\n")
	b.WriteString("<!-- Run: make api-coverage -->\n\n")
	b.WriteString("**Route inventory**: source-derived contract `testdata/contracts/coolify-v4.json` (`routes[]`)  \n")
	b.WriteString("**Field source of truth**: same contract (`models` / endpoint allow lists)  \n")
	b.WriteString("**Not inventory**: OpenAPI under `testdata/specs/` (partial upstream path list; do not treat as Coolify API completeness)  \n")
	fmt.Fprintf(&b, "**Coverage**: %d covered / %d registry entries (%.1f%%)  \n", len(coveredList), total, pct)
	fmt.Fprintf(&b, "**Planned**: %d | **Skipped**: %d  \n", len(plannedList), len(skippedList))
	fmt.Fprintf(&b, "**Registry size**: %d (contract routes + allowlisted extras)\n", total)

	// Covered
	b.WriteString("\n## Covered\n\n")
	b.WriteString("| Endpoint | Terraform Resource / Data Source | Since |\n")
	b.WriteString("|----------|----------------------------------|-------|\n")
	for _, e := range coveredList {
		fmt.Fprintf(&b, "| `%s` | `%s` | %s |\n", e.endpoint, e.status.resource, e.status.since)
	}

	// Planned
	b.WriteString("\n## Planned\n\n")
	b.WriteString("Ordered by priority (1 = most needed by users).\n\n")
	b.WriteString("| Priority | Endpoint | Notes |\n")
	b.WriteString("|----------|----------|-------|\n")
	for _, e := range plannedList {
		fmt.Fprintf(&b, "| %d | `%s` | %s |\n", e.status.priority, e.endpoint, e.status.notes)
	}

	// Skipped
	b.WriteString("\n## Intentionally Skipped\n\n")
	b.WriteString("These endpoints are intentionally not modeled directly in Terraform.\n\n")
	b.WriteString("| Endpoint | Reason |\n")
	b.WriteString("|----------|--------|\n")
	for _, e := range skippedList {
		fmt.Fprintf(&b, "| `%s` | %s |\n", e.endpoint, e.status.resource)
	}

	b.WriteString("\n## Unclassified contract routes\n\n")
	b.WriteString("_None. All pin contract routes are classified in `coveredEndpoints()`._\n\n")
	b.WriteString("When `make contract-extract` adds routes, classify them in\n")
	b.WriteString("`internal/spectest/coverage_test.go` or `TestSpecCoverage_Completeness` fails.\n")

	outPath := filepath.Join(testdataDir(), "..", "API_COVERAGE.md")
	if err := os.WriteFile(outPath, []byte(b.String()), 0644); err != nil {
		t.Fatalf("writing API_COVERAGE.md: %v", err)
	}
	t.Logf("Generated %s (%d bytes)", outPath, len(b.String()))
}

// loadContractRoutes returns sorted unique "METHOD /path" keys from a contract JSON file
// under testdata/contracts/.
func loadContractRoutes(filename string) ([]string, error) {
	path := filepath.Join(testdataDir(), "contracts", filename)
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc struct {
		Routes []struct {
			Method string `json:"method"`
			Path   string `json:"path"`
		} `json:"routes"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("parse %s: %w", filename, err)
	}
	seen := make(map[string]bool, len(doc.Routes))
	var ops []string
	for _, r := range doc.Routes {
		method := strings.ToUpper(strings.TrimSpace(r.Method))
		p := strings.TrimSpace(r.Path)
		if method == "" || p == "" {
			return nil, fmt.Errorf("%s: route entry with empty method or path", filename)
		}
		if !strings.HasPrefix(p, "/") {
			p = "/" + p
		}
		key := method + " " + p
		if seen[key] {
			continue
		}
		seen[key] = true
		ops = append(ops, key)
	}
	sort.Strings(ops)
	return ops, nil
}
