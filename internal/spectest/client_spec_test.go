package spectest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestClientEndpoints_SpecCompliance validates that every API endpoint
// our client uses stays compatible with the pinned OpenAPI route spec.
// This is a route and payload-shape audit only. The source-derived contract
// remains the field-level source of truth for provider behavior.
func TestClientEndpoints_SpecCompliance(t *testing.T) {
	t.Parallel()

	v, err := newValidator("coolify-v4")
	if err != nil {
		t.Fatalf("creating validator: %v", err)
	}

	type endpoint struct {
		name       string
		method     string
		path       string
		body       interface{}
		wantStatus int
		response   interface{}
	}

	endpoints := []endpoint{
		// Projects
		{"CreateProject", "POST", "/api/v1/projects",
			map[string]string{"name": "test", "description": "desc"},
			201, map[string]string{"uuid": "proj-1"}},
		{"GetProject", "GET", "/api/v1/projects/proj-1",
			nil, 200, map[string]interface{}{"uuid": "proj-1", "name": "test", "description": "desc"}},
		{"UpdateProject", "PATCH", "/api/v1/projects/proj-1",
			map[string]string{"name": "updated"},
			201, map[string]interface{}{"uuid": "proj-1", "name": "updated", "description": "desc"}},
		{"DeleteProject", "DELETE", "/api/v1/projects/proj-1",
			nil, 200, map[string]string{"message": "deleted"}},
		{"ListProjects", "GET", "/api/v1/projects",
			nil, 200, []map[string]interface{}{{"uuid": "proj-1", "name": "test"}}},

		// Servers
		{"CreateServer", "POST", "/api/v1/servers",
			map[string]interface{}{"name": "srv", "ip": "10.0.0.1", "port": 22, "private_key_uuid": "pk-1"},
			201, map[string]string{"uuid": "srv-1"}},
		{"GetServer", "GET", "/api/v1/servers/srv-1",
			nil, 200, map[string]interface{}{"uuid": "srv-1", "name": "srv", "ip": "10.0.0.1", "port": 22}},
		{"DeleteServer", "DELETE", "/api/v1/servers/srv-1",
			nil, 200, map[string]string{"message": "deleted"}},

		// Applications
		{"CreatePublicApp", "POST", "/api/v1/applications/public",
			map[string]interface{}{
				"project_uuid": "proj-1", "server_uuid": "srv-1",
				"environment_name": "production", "environment_uuid": "env-1",
				"git_repository": "https://github.com/ex/repo", "git_branch": "main",
				"build_pack": "nixpacks", "ports_exposes": "3000",
			}, 201, map[string]string{"uuid": "app-1"}},
		{"GetApplication", "GET", "/api/v1/applications/app-1",
			nil, 200, map[string]interface{}{"uuid": "app-1", "name": "my-app"}},
		{"DeleteApplication", "DELETE", "/api/v1/applications/app-1",
			nil, 200, map[string]string{"message": "deleted"}},

		// Databases
		{"CreatePostgresql", "POST", "/api/v1/databases/postgresql",
			map[string]interface{}{
				"server_uuid": "srv-1", "project_uuid": "proj-1",
				"environment_name": "production", "environment_uuid": "env-1",
			}, 201, map[string]string{"uuid": "db-1"}},
		{"GetDatabase", "GET", "/api/v1/databases/db-1",
			nil, 200, map[string]interface{}{"uuid": "db-1", "name": "pg-db"}},
		{"DeleteDatabase", "DELETE", "/api/v1/databases/db-1",
			nil, 200, map[string]string{"message": "deleted"}},

		// Private Keys
		{"CreatePrivateKey", "POST", "/api/v1/security/keys",
			map[string]interface{}{"name": "my-key", "private_key": "ssh-ed25519 AAAA"},
			201, map[string]string{"uuid": "pk-1"}},
		{"GetPrivateKey", "GET", "/api/v1/security/keys/pk-1",
			nil, 200, map[string]interface{}{"uuid": "pk-1", "name": "my-key", "private_key": "ssh-ed25519 AAAA", "is_git_related": false}},
		{"DeletePrivateKey", "DELETE", "/api/v1/security/keys/pk-1",
			nil, 200, map[string]string{"message": "deleted"}},

		// Services
		{"CreateService", "POST", "/api/v1/services",
			map[string]interface{}{
				"type": "plausible", "server_uuid": "srv-1",
				"project_uuid": "proj-1", "environment_name": "production",
				"environment_uuid": "env-1",
			}, 201, map[string]interface{}{"uuid": "svc-1", "domains": []string{}}},
		{"GetService", "GET", "/api/v1/services/svc-1",
			nil, 200, map[string]interface{}{"uuid": "svc-1", "name": "plausible"}},
		{"DeleteService", "DELETE", "/api/v1/services/svc-1",
			nil, 200, map[string]string{"message": "deleted"}},

		// Environment Variables (application)
		{"CreateAppEnvVar", "POST", "/api/v1/applications/app-1/envs",
			map[string]interface{}{"key": "DB_HOST", "value": "localhost", "is_preview": false},
			201, map[string]string{"uuid": "env-1"}},
		{"ListAppEnvVars", "GET", "/api/v1/applications/app-1/envs",
			nil, 200, []map[string]interface{}{{"uuid": "env-1", "key": "DB_HOST", "value": "localhost"}}},
		{"DeleteAppEnvVar", "DELETE", "/api/v1/applications/app-1/envs/env-1",
			nil, 200, map[string]string{"message": "deleted"}},

		// Database Backups
		{"CreateBackup", "POST", "/api/v1/databases/db-1/backups",
			map[string]interface{}{"frequency": "0 2 * * *", "enabled": true},
			201, map[string]interface{}{"uuid": "bak-1", "message": "created"}},

		// Teams
		{"GetTeam", "GET", "/api/v1/teams/1",
			nil, 200, map[string]interface{}{"id": 1, "name": "team-1"}},

		// Deployments
		{"ListDeployments", "GET", "/api/v1/deployments",
			nil, 200, []map[string]interface{}{{"uuid": "dep-1", "status": "queued"}}},
		{"ListAppDeployments", "GET", "/api/v1/deployments/applications/app-1",
			nil, 200, []map[string]interface{}{{"uuid": "dep-1", "status": "queued"}}},
		{"CancelDeployment", "POST", "/api/v1/deployments/dep-1/cancel",
			nil, 200, map[string]string{"message": "cancelled"}},
		{"Deploy", "GET", "/api/v1/deploy",
			nil, 200, map[string]string{"message": "deploy queued"}},

		// Environments
		{"ListEnvironments", "GET", "/api/v1/projects/proj-1/environments",
			nil, 200, []map[string]interface{}{{"id": 1, "name": "production"}}},
		{"CreateEnvironment", "POST", "/api/v1/projects/proj-1/environments",
			map[string]string{"name": "staging"},
			201, map[string]interface{}{"name": "staging"}},
		{"DeleteEnvironment", "DELETE", "/api/v1/projects/proj-1/environments/staging",
			nil, 200, map[string]string{"message": "deleted"}},
		{"GetEnvironment", "GET", "/api/v1/projects/proj-1/staging",
			nil, 200, map[string]interface{}{"id": 1, "name": "staging"}},

		// Applications (additional types)
		{"CreateDockerfileApp", "POST", "/api/v1/applications/dockerfile",
			map[string]interface{}{
				"project_uuid": "proj-1", "server_uuid": "srv-1",
				"environment_name": "production", "environment_uuid": "env-1",
				"dockerfile": "FROM nginx", "ports_exposes": "80",
			}, 201, map[string]string{"uuid": "app-1"}},
		{"CreateGitHubAppApp", "POST", "/api/v1/applications/private-github-app",
			map[string]interface{}{
				"project_uuid": "proj-1", "server_uuid": "srv-1",
				"environment_name": "production", "environment_uuid": "env-1",
				"github_app_uuid": "gh-1", "git_repository": "owner/repo",
				"git_branch": "main", "build_pack": "nixpacks", "ports_exposes": "3000",
			}, 201, map[string]string{"uuid": "app-1"}},
		{"StartApplication", "GET", "/api/v1/applications/app-1/start",
			nil, 200, map[string]string{"message": "started"}},
		{"StopApplication", "GET", "/api/v1/applications/app-1/stop",
			nil, 200, map[string]string{"message": "stopped"}},
		{"GetApplicationLogs", "GET", "/api/v1/applications/app-1/logs",
			nil, 200, []map[string]interface{}{{"timestamp": "2024-01-01T00:00:00Z", "message": "started"}}},
		{"DeletePreview", "DELETE", "/api/v1/applications/app-1/previews/123",
			nil, 200, map[string]string{"message": "deleted"}},

		// Scheduled Tasks (application)
		{"ListAppTasks", "GET", "/api/v1/applications/app-1/scheduled-tasks",
			nil, 200, []map[string]interface{}{{"uuid": "task-1", "name": "cleanup"}}},
		{"CreateAppTask", "POST", "/api/v1/applications/app-1/scheduled-tasks",
			map[string]interface{}{"name": "cleanup", "command": "rm -rf /tmp/*", "frequency": "0 * * * *"},
			201, map[string]interface{}{"uuid": "task-1", "message": "created"}},
		{"UpdateAppTask", "PATCH", "/api/v1/applications/app-1/scheduled-tasks/task-1",
			map[string]interface{}{"name": "cleanup-v2"},
			200, map[string]interface{}{"uuid": "task-1", "message": "updated"}},
		{"DeleteAppTask", "DELETE", "/api/v1/applications/app-1/scheduled-tasks/task-1",
			nil, 200, map[string]string{"message": "deleted"}},
		{"ListAppTaskExecs", "GET", "/api/v1/applications/app-1/scheduled-tasks/task-1/executions",
			nil, 200, []map[string]interface{}{{"id": 1, "status": "success"}}},

		// Storages (application)
		{"ListAppStorages", "GET", "/api/v1/applications/app-1/storages",
			nil, 200, []map[string]interface{}{{"id": 1, "name": "data"}}},
		{"CreateAppStorage", "POST", "/api/v1/applications/app-1/storages",
			map[string]interface{}{"name": "data", "mount_path": "/data", "host_path": "/host/data"},
			201, map[string]interface{}{"id": 1, "message": "created"}},
		{"UpdateAppStorage", "PATCH", "/api/v1/applications/app-1/storages",
			map[string]interface{}{"id": 1, "name": "data-v2"},
			200, map[string]interface{}{"id": 1, "message": "updated"}},
		{"DeleteAppStorage", "DELETE", "/api/v1/applications/app-1/storages/stor-1",
			nil, 200, map[string]string{"message": "deleted"}},

		// App env bulk
		{"BulkUpdateAppEnvs", "PATCH", "/api/v1/applications/app-1/envs/bulk",
			map[string]interface{}{"data": []map[string]interface{}{{"key": "K1", "value": "V1"}}},
			200, map[string]string{"message": "updated"}},

		// Database environment variables
		{"ListDatabaseEnvVars", "GET", "/api/v1/databases/db-1/envs",
			nil, 200, []map[string]interface{}{{"uuid": "env-1", "key": "PG_HOST", "value": "localhost"}}},
		{"CreateDatabaseEnvVar", "POST", "/api/v1/databases/db-1/envs",
			map[string]interface{}{"key": "PG_HOST", "value": "localhost", "is_preview": false},
			201, map[string]string{"uuid": "env-1"}},
		{"UpdateDatabaseEnvVar", "PATCH", "/api/v1/databases/db-1/envs",
			map[string]interface{}{"key": "PG_HOST", "value": "db.local"},
			200, map[string]string{"message": "updated"}},
		{"DeleteDatabaseEnvVar", "DELETE", "/api/v1/databases/db-1/envs/env-1",
			nil, 200, map[string]string{"message": "deleted"}},
		{"BulkUpdateDbEnvs", "PATCH", "/api/v1/databases/db-1/envs/bulk",
			map[string]interface{}{"data": []map[string]interface{}{{"key": "K1", "value": "V1"}}},
			200, map[string]string{"message": "updated"}},

		// Database lifecycle
		{"RestartDatabase", "GET", "/api/v1/databases/db-1/restart",
			nil, 200, map[string]string{"message": "restarted"}},
		{"StartDatabase", "GET", "/api/v1/databases/db-1/start",
			nil, 200, map[string]string{"message": "started"}},
		{"StopDatabase", "GET", "/api/v1/databases/db-1/stop",
			nil, 200, map[string]string{"message": "stopped"}},

		// Database storages
		{"ListDbStorages", "GET", "/api/v1/databases/db-1/storages",
			nil, 200, []map[string]interface{}{{"id": 1, "name": "pgdata"}}},
		{"CreateDbStorage", "POST", "/api/v1/databases/db-1/storages",
			map[string]interface{}{"name": "pgdata", "mount_path": "/var/lib/postgresql", "host_path": "/host/pgdata"},
			201, map[string]interface{}{"id": 1, "message": "created"}},
		{"UpdateDbStorage", "PATCH", "/api/v1/databases/db-1/storages",
			map[string]interface{}{"id": 1, "name": "pgdata-v2"},
			200, map[string]interface{}{"id": 1, "message": "updated"}},
		{"DeleteDbStorage", "DELETE", "/api/v1/databases/db-1/storages/stor-1",
			nil, 200, map[string]string{"message": "deleted"}},

		// Database backup executions
		{"ListBackupExecs", "GET", "/api/v1/databases/db-1/backups/bak-1/executions",
			nil, 200, []map[string]interface{}{{"id": 1, "status": "success"}}},
		{"DeleteBackupExec", "DELETE", "/api/v1/databases/db-1/backups/bak-1/executions/exec-1",
			nil, 200, map[string]string{"message": "deleted"}},

		// Services extras
		{"ListServiceEnvVars", "GET", "/api/v1/services/svc-1/envs",
			nil, 200, []map[string]interface{}{{"uuid": "env-1", "key": "SECRET", "value": "val"}}},
		{"BulkUpdateSvcEnvs", "PATCH", "/api/v1/services/svc-1/envs/bulk",
			map[string]interface{}{"data": []map[string]interface{}{{"key": "K1", "value": "V1"}}},
			200, map[string]string{"message": "updated"}},
		{"RestartService", "GET", "/api/v1/services/svc-1/restart",
			nil, 200, map[string]string{"message": "restarted"}},
		{"StartService", "GET", "/api/v1/services/svc-1/start",
			nil, 200, map[string]string{"message": "started"}},
		{"StopService", "GET", "/api/v1/services/svc-1/stop",
			nil, 200, map[string]string{"message": "stopped"}},

		// Scheduled Tasks (service)
		{"ListSvcTasks", "GET", "/api/v1/services/svc-1/scheduled-tasks",
			nil, 200, []map[string]interface{}{{"uuid": "task-1", "name": "cleanup"}}},
		{"CreateSvcTask", "POST", "/api/v1/services/svc-1/scheduled-tasks",
			map[string]interface{}{"name": "cleanup", "command": "rm -rf /tmp/*", "frequency": "0 * * * *"},
			201, map[string]interface{}{"uuid": "task-1", "message": "created"}},
		{"UpdateSvcTask", "PATCH", "/api/v1/services/svc-1/scheduled-tasks/task-1",
			map[string]interface{}{"name": "cleanup-v2"},
			200, map[string]interface{}{"uuid": "task-1", "message": "updated"}},
		{"DeleteSvcTask", "DELETE", "/api/v1/services/svc-1/scheduled-tasks/task-1",
			nil, 200, map[string]string{"message": "deleted"}},
		{"ListSvcTaskExecs", "GET", "/api/v1/services/svc-1/scheduled-tasks/task-1/executions",
			nil, 200, []map[string]interface{}{{"id": 1, "status": "success"}}},

		// Storages (service)
		{"ListSvcStorages", "GET", "/api/v1/services/svc-1/storages",
			nil, 200, []map[string]interface{}{{"id": 1, "name": "svc-data"}}},
		{"CreateSvcStorage", "POST", "/api/v1/services/svc-1/storages",
			map[string]interface{}{"name": "svc-data", "mount_path": "/data", "host_path": "/host/svc-data"},
			201, map[string]interface{}{"id": 1, "message": "created"}},
		{"UpdateSvcStorage", "PATCH", "/api/v1/services/svc-1/storages",
			map[string]interface{}{"id": 1, "name": "svc-data-v2"},
			200, map[string]interface{}{"id": 1, "message": "updated"}},
		{"DeleteSvcStorage", "DELETE", "/api/v1/services/svc-1/storages/stor-1",
			nil, 200, map[string]string{"message": "deleted"}},

		// Cloud Tokens
		{"ListCloudTokens", "GET", "/api/v1/cloud-tokens",
			nil, 200, []map[string]interface{}{{"id": 1, "name": "my-token"}}},
		{"CreateCloudToken", "POST", "/api/v1/cloud-tokens",
			map[string]interface{}{"name": "my-token", "provider": "hetzner", "token": "secret-token"},
			201, map[string]interface{}{"uuid": "ct-1"}},
		{"GetCloudToken", "GET", "/api/v1/cloud-tokens/ct-1",
			nil, 200, map[string]interface{}{"uuid": "ct-1", "name": "my-token", "provider": "hetzner"}},
		{"UpdateCloudToken", "PATCH", "/api/v1/cloud-tokens/ct-1",
			map[string]interface{}{"name": "updated-token"},
			200, map[string]interface{}{"uuid": "ct-1"}},
		{"DeleteCloudToken", "DELETE", "/api/v1/cloud-tokens/ct-1",
			nil, 200, map[string]string{"message": "deleted"}},
		{"ValidateCloudToken", "POST", "/api/v1/cloud-tokens/ct-1/validate",
			nil, 200, map[string]interface{}{"valid": true}},

		// GitHub Apps
		{"ListGitHubApps", "GET", "/api/v1/github-apps",
			nil, 200, []map[string]interface{}{{"id": 1, "uuid": "gh-1", "name": "my-gh-app", "api_url": "https://api.github.com", "html_url": "https://github.com", "app_id": 12345, "installation_id": 67890, "client_id": "Iv1.abc123", "private_key_id": 1, "is_system_wide": false, "team_id": 1}}},
		{"CreateGitHubApp", "POST", "/api/v1/github-apps",
			map[string]interface{}{
				"name": "my-gh-app", "api_url": "https://api.github.com", "html_url": "https://github.com",
				"app_id": 12345, "installation_id": 67890, "client_id": "Iv1.abc123",
				"client_secret": "secret", "private_key_uuid": "pk-1",
			},
			201, map[string]interface{}{"id": 1, "uuid": "gh-1", "name": "my-gh-app"}},
		{"UpdateGitHubApp", "PATCH", "/api/v1/github-apps/1",
			map[string]interface{}{"name": "updated-gh-app", "client_secret": "updated-secret"},
			200, map[string]interface{}{"message": "GitHub app updated successfully", "data": map[string]interface{}{"id": 1, "name": "updated-gh-app"}}},
		{"DeleteGitHubApp", "DELETE", "/api/v1/github-apps/1",
			nil, 200, map[string]string{"message": "deleted"}},
		{"ListRepos", "GET", "/api/v1/github-apps/1/repositories",
			nil, 200, []map[string]interface{}{{"name": "repo", "full_name": "owner/repo"}}},
		{"ListBranches", "GET", "/api/v1/github-apps/1/repositories/owner/repo/branches",
			nil, 200, []map[string]interface{}{{"name": "main"}}},

		// Hetzner
		{"HetznerImages", "GET", "/api/v1/hetzner/images?cloud_provider_token_uuid=tok-1",
			nil, 200, []map[string]interface{}{{"id": 1, "name": "ubuntu-22.04"}}},
		{"HetznerLocations", "GET", "/api/v1/hetzner/locations?cloud_provider_token_uuid=tok-1",
			nil, 200, []map[string]interface{}{{"id": 1, "name": "fsn1"}}},
		{"HetznerServerTypes", "GET", "/api/v1/hetzner/server-types?cloud_provider_token_uuid=tok-1",
			nil, 200, []map[string]interface{}{{"id": 1, "name": "cx11"}}},
		{"HetznerSSHKeys", "GET", "/api/v1/hetzner/ssh-keys?cloud_provider_token_uuid=tok-1",
			nil, 200, []map[string]interface{}{{"id": 1, "name": "my-key"}}},

		// Servers (additional)
		{"CreateHetznerServer", "POST", "/api/v1/servers/hetzner",
			map[string]interface{}{
				"name": "hetzner-srv", "private_key_uuid": "pk-1",
				"cloud_provider_token_uuid": "tok-1", "server_type": "cx11",
				"location": "fsn1", "image": "ubuntu-22.04",
			}, 201, map[string]string{"uuid": "srv-1"}},
		{"ValidateServer", "GET", "/api/v1/servers/srv-1/validate",
			nil, 200, map[string]interface{}{"valid": true}},

		// Teams (additional)
		{"ListTeams", "GET", "/api/v1/teams",
			nil, 200, []map[string]interface{}{{"id": 1, "name": "team-1"}}},
		{"ListTeamMembers", "GET", "/api/v1/teams/1/members",
			nil, 200, []map[string]interface{}{{"id": 1, "name": "user-1"}}},
		{"GetCurrentTeam", "GET", "/api/v1/teams/current",
			nil, 200, map[string]interface{}{"id": 1, "name": "team-1"}},
		{"GetCurrentTeamMembers", "GET", "/api/v1/teams/current/members",
			nil, 200, []map[string]interface{}{{"id": 1, "name": "user-1"}}},

		// Other
		{"ListResources", "GET", "/api/v1/resources",
			nil, 200, []map[string]interface{}{{"uuid": "res-1", "type": "application"}}},
		{"Health", "GET", "/api/v1/health",
			nil, 200, "OK"},
		{"Enable", "GET", "/api/v1/enable",
			nil, 200, map[string]string{"message": "enabled"}},
		{"Disable", "GET", "/api/v1/disable",
			nil, 200, map[string]string{"message": "disabled"}},
	}

	for _, ep := range endpoints {
		t.Run(ep.name, func(t *testing.T) {
			// Not parallel: libopenapi-validator has internal state
			// that races when shared across goroutines.
			mux := http.NewServeMux()
			// Strip query string for mux pattern (ServeMux matches on path only).
			handlerPath := ep.path
			if idx := strings.IndexByte(handlerPath, '?'); idx != -1 {
				handlerPath = handlerPath[:idx]
			}
			pattern := fmt.Sprintf("%s %s", ep.method, handlerPath)
			mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(ep.wantStatus)
				json.NewEncoder(w).Encode(ep.response)
			})
			// Version endpoint.
			mux.HandleFunc("GET /api/v1/version", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/html")
				w.Write([]byte("v4.0.0"))
			})

			srv := httptest.NewServer(withSpecValidation(t, "coolify-v4", mux, false, v))
			defer srv.Close()

			var reqBody *bytes.Buffer
			if ep.body != nil {
				data, _ := json.Marshal(ep.body)
				reqBody = bytes.NewBuffer(data)
			} else {
				reqBody = &bytes.Buffer{}
			}

			req, err := http.NewRequest(ep.method, srv.URL+ep.path, reqBody)
			if err != nil {
				t.Fatalf("creating request: %v", err)
			}
			if ep.body != nil {
				req.Header.Set("Content-Type", "application/json")
			}
			req.Header.Set("Authorization", "Bearer test-token")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != ep.wantStatus {
				t.Errorf("expected status %d, got %d", ep.wantStatus, resp.StatusCode)
			}
		})
	}
}
