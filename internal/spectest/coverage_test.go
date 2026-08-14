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
	skipped := func(kind string) coverageStatus {
		return coverageStatus{category: "skipped", resource: kind}
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
		"POST /applications/dockercompose":                                skipped(skipDeprecated),
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
		"DELETE /applications/{uuid}/destinations/{destination_uuid}":                   covered("coolify_application_destination", "v0.1.15"),
		"DELETE /applications/{uuid}/tags/{tag_uuid}":                                   covered("coolify_tag + coolify_resource_tag", "v0.1.15"),
		"DELETE /cloud-init-scripts/{uuid}":                                             covered("coolify_cloud_init_script", "v0.1.15"),
		"DELETE /databases/{uuid}/tags/{tag_uuid}":                                      covered("coolify_tag + coolify_resource_tag", "v0.1.15"),
		"DELETE /destinations/{uuid}":                                                   covered("coolify_destination", "v0.2.0"),
		"DELETE /gitlab-apps/{gitlab_app_id}":                                           covered("coolify_gitlab_app", "v0.1.15"),
		"DELETE /projects/{uuid}/environments/{environment_name_or_uuid}/envs/{env_id}": covered("coolify_shared_environment_variable", "v0.1.15"),
		"DELETE /projects/{uuid}/envs/{env_id}":                                         covered("coolify_shared_environment_variable", "v0.1.15"),
		"DELETE /s3-storages/{uuid}":                                                    covered("coolify_s3_storage", "v0.1.13"),
		"DELETE /servers/{uuid}/envs/{env_id}":                                          covered("coolify_shared_environment_variable", "v0.1.15"),
		"DELETE /services/{uuid}/tags/{tag_uuid}":                                       covered("coolify_tag + coolify_resource_tag", "v0.1.15"),
		"DELETE /tags/{uuid}":                                                           covered("coolify_tag + coolify_resource_tag", "v0.1.15"),
		"DELETE /team/envs/{env_id}":                                                    covered("coolify_shared_environment_variable", "v0.1.15"),
		"GET /applications/{uuid}/destinations":                                         covered("coolify_application_destination", "v0.1.15"),
		"GET /applications/{uuid}/rollback-images":                                      skipped(skipRollback),
		"GET /applications/{uuid}/tags":                                                 covered("coolify_tag + coolify_resource_tag", "v0.1.15"),
		"GET /cloud-init-scripts":                                                       covered("coolify_cloud_init_script", "v0.1.15"),
		"GET /cloud-init-scripts/{uuid}":                                                covered("coolify_cloud_init_script", "v0.1.15"),
		"GET /databases/{uuid}/logs":                                                    skipped(skipLogs),
		"GET /databases/{uuid}/tags":                                                    covered("coolify_tag + coolify_resource_tag", "v0.1.15"),
		"GET /destinations":                                                             covered("data.coolify_destinations", "v0.2.0"),
		"GET /destinations/{uuid}":                                                      covered("data.coolify_destination", "v0.2.0"),
		"GET /digitalocean/images":                                                      covered("data.coolify_digitalocean_images", "v0.2.0"),
		"GET /digitalocean/regions":                                                     covered("data.coolify_digitalocean_regions", "v0.2.0"),
		"GET /digitalocean/sizes":                                                       covered("data.coolify_digitalocean_sizes", "v0.2.0"),
		"GET /digitalocean/ssh-keys":                                                    covered("data.coolify_digitalocean_ssh_keys", "v0.2.0"),
		"GET /gitlab-apps":                                                              covered("coolify_gitlab_app", "v0.1.15"),
		"GET /hetzner/firewalls":                                                        covered("data.coolify_hetzner_firewalls", "v0.1.16"),
		"GET /hetzner/networks":                                                         covered("data.coolify_hetzner_networks", "v0.1.16"),
		"GET /notifications/discord":                                                    covered("coolify_notification_discord", "v0.1.14"),
		"GET /notifications/email":                                                      covered("coolify_notification_email", "v0.1.14"),
		"GET /notifications/pushover":                                                   covered("coolify_notification_pushover", "v0.1.14"),
		"GET /notifications/slack":                                                      covered("coolify_notification_slack", "v0.1.14"),
		"GET /notifications/telegram":                                                   covered("coolify_notification_telegram", "v0.1.14"),
		"GET /notifications/webhook":                                                    covered("coolify_notification_webhook", "v0.1.14"),
		"GET /projects/{uuid}/environments/{environment_name_or_uuid}/envs":             covered("coolify_shared_environment_variable", "v0.1.15"),
		"GET /projects/{uuid}/envs":                                                     covered("coolify_shared_environment_variable", "v0.1.15"),
		"GET /s3-storages":                                                              covered("data.coolify_s3_storages", "v0.1.13"),
		"GET /s3-storages/{uuid}":                                                       covered("data.coolify_s3_storage", "v0.1.13"),
		"GET /servers/{server_uuid}/destinations":                                       covered("data.coolify_destinations", "v0.2.0"),
		"GET /servers/{uuid}/cloudflare-tunnel":                                         covered("coolify_server_cloudflare_tunnel", "v0.1.15"),
		"GET /servers/{uuid}/docker-cleanup":                                            covered("coolify_server_docker_cleanup", "v0.1.15"),
		"GET /servers/{uuid}/docker-cleanup/executions":                                 skipped(skipControlPlane),
		"GET /servers/{uuid}/envs":                                                      covered("coolify_shared_environment_variable", "v0.1.15"),
		"GET /servers/{uuid}/export":                                                    skipped(skipControlPlane),
		"GET /servers/{uuid}/log-drains":                                                covered("coolify_server_log_drain", "v0.1.15"),
		"GET /servers/{uuid}/proxy":                                                     covered("coolify_server_proxy", "v0.1.15"),
		"GET /servers/{uuid}/sentinel":                                                  covered("coolify_server_sentinel", "v0.1.15"),
		"GET /services/{uuid}/applications":                                             skipped(skipNestedService),
		"GET /services/{uuid}/applications/{app_uuid}":                                  skipped(skipNestedService),
		"GET /services/{uuid}/applications/{app_uuid}/logs":                             skipped(skipLogs),
		"GET /services/{uuid}/applications/{app_uuid}/restart":                          skipped(skipNestedService),
		"GET /services/{uuid}/applications/{app_uuid}/start":                            skipped(skipNestedService),
		"GET /services/{uuid}/applications/{app_uuid}/stop":                             skipped(skipNestedService),
		"GET /services/{uuid}/databases":                                                skipped(skipNestedService),
		"GET /services/{uuid}/databases/{database_uuid}":                                skipped(skipNestedService),
		"GET /services/{uuid}/databases/{database_uuid}/logs":                           skipped(skipLogs),
		"GET /services/{uuid}/logs":                                                     skipped(skipLogs),
		"GET /services/{uuid}/tags":                                                     covered("coolify_tag + coolify_resource_tag", "v0.1.15"),
		"GET /tags":                                                                     covered("coolify_tag + coolify_resource_tag", "v0.1.15"),
		"GET /team":                                                                     skipped(skipAlias),
		"GET /team/envs":                                                                covered("coolify_shared_environment_variable", "v0.1.15"),
		"GET /team/members":                                                             skipped(skipAlias),
		"GET /vultr/os":                                                                 covered("data.coolify_vultr_os", "v0.2.0"),
		"GET /vultr/plans":                                                              covered("data.coolify_vultr_plans", "v0.2.0"),
		"GET /vultr/regions":                                                            covered("data.coolify_vultr_regions", "v0.2.0"),
		"GET /vultr/ssh-keys":                                                           covered("data.coolify_vultr_ssh_keys", "v0.2.0"),
		"PATCH /cloud-init-scripts/{uuid}":                                              covered("coolify_cloud_init_script", "v0.1.15"),
		"PATCH /destinations/{uuid}":                                                    covered("coolify_destination", "v0.1.15"),
		"PATCH /gitlab-apps/{gitlab_app_id}":                                            covered("coolify_gitlab_app", "v0.1.15"),
		"PATCH /notifications/discord":                                                  covered("coolify_notification_discord", "v0.1.14"),
		"PATCH /notifications/email":                                                    covered("coolify_notification_email", "v0.1.14"),
		"PATCH /notifications/pushover":                                                 covered("coolify_notification_pushover", "v0.1.14"),
		"PATCH /notifications/slack":                                                    covered("coolify_notification_slack", "v0.1.14"),
		"PATCH /notifications/telegram":                                                 covered("coolify_notification_telegram", "v0.1.14"),
		"PATCH /notifications/webhook":                                                  covered("coolify_notification_webhook", "v0.1.14"),
		"PATCH /projects/{uuid}/environments/{environment_name_or_uuid}":                covered("coolify_environment", "v0.1.15"),
		"PATCH /projects/{uuid}/environments/{environment_name_or_uuid}/envs/{env_id}": covered("coolify_shared_environment_variable", "v0.1.15"),
		"PATCH /projects/{uuid}/envs/{env_id}":                                         covered("coolify_shared_environment_variable", "v0.1.15"),
		"PATCH /s3-storages/{uuid}":                                                    covered("coolify_s3_storage", "v0.1.13"),
		"PATCH /servers/{uuid}/cloudflare-tunnel":                                      covered("coolify_server_cloudflare_tunnel", "v0.1.15"),
		"PATCH /servers/{uuid}/docker-cleanup":                                         covered("coolify_server_docker_cleanup", "v0.1.15"),
		"PATCH /servers/{uuid}/envs/{env_id}":                                          covered("coolify_shared_environment_variable", "v0.1.15"),
		"PATCH /servers/{uuid}/log-drains":                                             covered("coolify_server_log_drain", "v0.1.15"),
		"PATCH /servers/{uuid}/proxy":                                                  covered("coolify_server_proxy", "v0.1.15"),
		"PATCH /servers/{uuid}/sentinel":                                               covered("coolify_server_sentinel", "v0.1.15"),
		"PATCH /services/{uuid}/applications/{app_uuid}":                               skipped(skipNestedService),
		"PATCH /services/{uuid}/databases/{database_uuid}":                             skipped(skipNestedService),
		"PATCH /tags/{uuid}":                                                           covered("coolify_tag + coolify_resource_tag", "v0.1.15"),
		"PATCH /team/envs/{env_id}":                                                    covered("coolify_shared_environment_variable", "v0.1.15"),
		"POST /applications/{uuid}/clone":                                              skipped(skipCloneMove),
		"POST /applications/{uuid}/destinations":                                       covered("coolify_application_destination", "v0.1.15"),
		"POST /applications/{uuid}/migrate":                                            skipped(skipCloneMove),
		"POST /applications/{uuid}/move":                                               skipped(skipCloneMove),
		"POST /applications/{uuid}/restart":                                            covered("coolify_resource_action", "v0.3.0"),
		"POST /applications/{uuid}/rollback":                                           skipped(skipRollback),
		"POST /applications/{uuid}/scheduled-tasks/{task_uuid}/execute":                skipped(skipRunNow),
		"POST /applications/{uuid}/start":                                              covered("coolify_resource_action", "v0.3.0"),
		"POST /applications/{uuid}/stop":                                               covered("coolify_resource_action", "v0.3.0"),
		"POST /applications/{uuid}/storages/{storage_uuid}/backups/run":                skipped(skipRunNow),
		"POST /applications/{uuid}/tags":                                               covered("coolify_tag + coolify_resource_tag", "v0.1.15"),
		"POST /cloud-init-scripts":                                                     covered("coolify_cloud_init_script", "v0.1.15"),
		"POST /databases/{uuid}/clone":                                                 skipped(skipCloneMove),
		"POST /databases/{uuid}/migrate":                                               skipped(skipCloneMove),
		"POST /databases/{uuid}/move":                                                  skipped(skipCloneMove),
		"POST /databases/{uuid}/restart":                                               covered("coolify_resource_action", "v0.3.0"),
		"POST /databases/{uuid}/start":                                                 covered("coolify_resource_action", "v0.3.0"),
		"POST /databases/{uuid}/stop":                                                  covered("coolify_resource_action", "v0.3.0"),
		"POST /databases/{uuid}/storages/{storage_uuid}/backups/run":                   skipped(skipRunNow),
		"POST /databases/{uuid}/tags":                                                  covered("coolify_tag + coolify_resource_tag", "v0.1.15"),
		"POST /deploy":                                                                 covered("client.Deploy", "v0.2.0"),
		"POST /disable":                                                                covered("coolify_api_settings", "v0.2.0"),
		"POST /enable":                                                                 covered("coolify_api_settings", "v0.2.0"),
		"POST /feedback":                                                               skipped(skipFeedback),
		"POST /gitlab-apps":                                                            covered("coolify_gitlab_app", "v0.1.15"),
		"POST /projects/{uuid}/environments/{environment_name_or_uuid}/envs":           covered("coolify_shared_environment_variable", "v0.1.15"),
		"POST /projects/{uuid}/envs":                                                   covered("coolify_shared_environment_variable", "v0.1.15"),
		"POST /s3-storages":                                                            covered("coolify_s3_storage", "v0.1.13"),
		"POST /s3-storages/{uuid}/validate":                                            covered("coolify_s3_storage_validate", "v0.1.14"),
		"POST /sentinel/push":                                                          skipped(skipControlPlane),
		"POST /servers/digitalocean":                                                   covered("coolify_server_digitalocean", "v0.2.0"),
		"POST /servers/import":                                                         skipped(skipControlPlane),
		"POST /servers/vultr":                                                          covered("coolify_server_vultr", "v0.2.0"),
		"POST /servers/{server_uuid}/destinations":                                     covered("coolify_destination", "v0.2.0"),
		"POST /servers/{uuid}/claim":                                                   skipped(skipControlPlane),
		"POST /servers/{uuid}/cloudflare-tunnel/disable":                               skipped(skipEnableDisable),
		"POST /servers/{uuid}/cloudflare-tunnel/enable":                                skipped(skipEnableDisable),
		"POST /servers/{uuid}/docker-cleanup/run":                                      skipped(skipControlPlane),
		"POST /servers/{uuid}/envs":                                                    covered("coolify_shared_environment_variable", "v0.1.15"),
		"POST /servers/{uuid}/export/mailbox":                                          skipped(skipControlPlane),
		"POST /servers/{uuid}/migrate":                                                 skipped(skipCloneMove),
		"POST /servers/{uuid}/proxy/restart":                                           skipped(skipControlPlane),
		"POST /servers/{uuid}/transfer/complete":                                       skipped(skipControlPlane),
		"POST /servers/{uuid}/validate":                                                covered("coolify_server_validate", "v0.2.0"),
		"POST /services/{uuid}/applications/{app_uuid}/logs":                           skipped(skipLogs),
		"POST /services/{uuid}/applications/{app_uuid}/restart":                        skipped(skipNestedService),
		"POST /services/{uuid}/applications/{app_uuid}/start":                          skipped(skipNestedService),
		"POST /services/{uuid}/applications/{app_uuid}/stop":                           skipped(skipNestedService),
		"POST /services/{uuid}/clone":                                                  skipped(skipCloneMove),
		"POST /services/{uuid}/databases/{database_uuid}/restart":                      skipped(skipNestedService),
		"POST /services/{uuid}/databases/{database_uuid}/start":                        skipped(skipNestedService),
		"POST /services/{uuid}/databases/{database_uuid}/stop":                         skipped(skipNestedService),
		"POST /services/{uuid}/migrate":                                                skipped(skipCloneMove),
		"POST /services/{uuid}/move":                                                   skipped(skipCloneMove),
		"POST /services/{uuid}/restart":                                                covered("coolify_resource_action", "v0.3.0"),
		"POST /services/{uuid}/scheduled-tasks/{task_uuid}/execute":                    skipped(skipRunNow),
		"POST /services/{uuid}/start":                                                  covered("coolify_resource_action", "v0.3.0"),
		"POST /services/{uuid}/stop":                                                   covered("coolify_resource_action", "v0.3.0"),
		"POST /services/{uuid}/storages/{storage_uuid}/backups/run":                    skipped(skipRunNow),
		"POST /services/{uuid}/tags":                                                   covered("coolify_tag + coolify_resource_tag", "v0.1.15"),
		"POST /tags":                                                                   covered("coolify_tag + coolify_resource_tag", "v0.1.15"),
		"POST /team/envs":                                                              covered("coolify_shared_environment_variable", "v0.1.15"),
		"PUT /servers/{uuid}/proxy/configuration":                                      covered("coolify_server_proxy", "v0.1.15"),
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

	md, err := generateCoverageMarkdown(coveredEndpoints())
	if err != nil {
		t.Fatalf("generating API_COVERAGE.md: %v", err)
	}

	outPath := filepath.Join(testdataDir(), "..", "API_COVERAGE.md")
	if err := os.WriteFile(outPath, []byte(md), 0644); err != nil {
		t.Fatalf("writing API_COVERAGE.md: %v", err)
	}
	t.Logf("Generated %s (%d bytes)", outPath, len(md))
}

func TestGenerateCoverageMarkdown_Readable(t *testing.T) {
	t.Parallel()
	md, err := generateCoverageMarkdown(coveredEndpoints())
	if err != nil {
		t.Fatal(err)
	}
	needles := []string{
		"docs/guides/coolify-version-support.md",
		"**Use this instead:**",
		"## What Terraform does not wrap",
		"## Routes by Terraform resource",
		"## Appendix: all classified routes",
	}
	for _, n := range needles {
		if !strings.Contains(md, n) {
			t.Errorf("generated doc missing %q", n)
		}
	}
	for ep, s := range coveredEndpoints() {
		if s.category != "skipped" {
			continue
		}
		if !strings.Contains(md, "`"+ep+"`") {
			t.Errorf("skipped route %s missing from generated doc", ep)
		}
	}
}

func TestSkipKinds_CoverAllSkipped(t *testing.T) {
	t.Parallel()
	kinds := skipKindByID()
	for ep, s := range coveredEndpoints() {
		if s.category != "skipped" {
			continue
		}
		k, ok := kinds[s.resource]
		if !ok {
			t.Errorf("%s: unknown skip kind %q", ep, s.resource)
			continue
		}
		if k.instead == "" {
			t.Errorf("skip kind %q has empty instead", s.resource)
		}
	}
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
