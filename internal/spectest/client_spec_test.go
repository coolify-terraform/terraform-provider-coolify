package spectest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
)

// TestClientEndpoints_SpecCompliance validates that every API endpoint
// our client uses stays compatible with the pinned OpenAPI route spec.
// This is a route and payload-shape audit only. The source-derived contract
// remains the field-level source of truth for provider behavior. Request
// bodies use real client input structs for cases where JSON tags do not
// obviously match the Go field names.
func TestClientEndpoints_SpecCompliance(t *testing.T) {
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

	updatedProjectName := "updated"
	updatedTaskName := "cleanup-v2"
	appStorageUUID := "app-stor-1"
	updatedAppStorageName := "data-v2"
	dbStorageUUID := "db-stor-1"
	updatedDBStorageName := "pgdata-v2"
	svcStorageUUID := "svc-stor-1"
	updatedSvcStorageName := "svc-data-v2"
	updatedCloudTokenName := "updated-token"
	updatedGitHubAppName := "updated-gh-app"
	updatedGitHubAppSecret := "updated-secret"

	endpoints := []endpoint{
		// Projects
		{"CreateProject", "POST", "/api/v1/projects",
			client.CreateProjectInput{Name: "test", Description: "desc"},
			201, map[string]string{"uuid": "proj-1"}},
		{"GetProject", "GET", "/api/v1/projects/proj-1",
			nil, 200, map[string]interface{}{"uuid": "proj-1", "name": "test", "description": "desc"}},
		{"UpdateProject", "PATCH", "/api/v1/projects/proj-1",
			client.UpdateProjectInput{Name: &updatedProjectName},
			201, map[string]interface{}{"uuid": "proj-1", "name": "updated", "description": "desc"}},
		{"DeleteProject", "DELETE", "/api/v1/projects/proj-1",
			nil, 200, map[string]string{"message": "deleted"}},
		{"ListProjects", "GET", "/api/v1/projects",
			nil, 200, []map[string]interface{}{{"uuid": "proj-1", "name": "test"}}},

		// Servers
		{"CreateServer", "POST", "/api/v1/servers",
			client.CreateServerInput{Name: "srv", IP: "10.0.0.1", Port: 22, PrivateKeyUUID: "pk-1"},
			201, map[string]string{"uuid": "srv-1"}},
		{"GetServer", "GET", "/api/v1/servers/srv-1",
			nil, 200, map[string]interface{}{"uuid": "srv-1", "name": "srv", "ip": "10.0.0.1", "port": 22}},
		{"DeleteServer", "DELETE", "/api/v1/servers/srv-1",
			nil, 200, map[string]string{"message": "deleted"}},

		// Applications
		{"CreatePublicApp", "POST", "/api/v1/applications/public",
			client.CreatePublicAppInput{
				ProjectUUID:     "proj-1",
				ServerUUID:      "srv-1",
				EnvironmentName: "production",
				EnvironmentUUID: "env-1",
				GitRepository:   "https://github.com/ex/repo",
				GitBranch:       "main",
				BuildPack:       "nixpacks",
				PortsExposes:    "3000",
			}, 201, map[string]string{"uuid": "app-1"}},
		{"GetApplication", "GET", "/api/v1/applications/app-1",
			nil, 200, map[string]interface{}{"uuid": "app-1", "name": "my-app"}},
		{"DeleteApplication", "DELETE", "/api/v1/applications/app-1",
			nil, 200, map[string]string{"message": "deleted"}},

		// Databases
		{"CreatePostgresql", "POST", "/api/v1/databases/postgresql",
			client.CreatePostgresqlInput{
				CreateDatabaseBaseInput: client.CreateDatabaseBaseInput{
					ServerUUID:      "srv-1",
					ProjectUUID:     "proj-1",
					EnvironmentName: "production",
					EnvironmentUUID: "env-1",
				},
			}, 201, map[string]string{"uuid": "db-1"}},
		{"GetDatabase", "GET", "/api/v1/databases/db-1",
			nil, 200, map[string]interface{}{"uuid": "db-1", "name": "pg-db"}},
		{"DeleteDatabase", "DELETE", "/api/v1/databases/db-1",
			nil, 200, map[string]string{"message": "deleted"}},

		// Private Keys
		{"CreatePrivateKey", "POST", "/api/v1/security/keys",
			client.CreatePrivateKeyInput{Name: "my-key", PrivateKey: "ssh-ed25519 AAAA"},
			201, map[string]string{"uuid": "pk-1"}},
		{"GetPrivateKey", "GET", "/api/v1/security/keys/pk-1",
			nil, 200, map[string]interface{}{"uuid": "pk-1", "name": "my-key", "private_key": "ssh-ed25519 AAAA", "is_git_related": false}},
		{"DeletePrivateKey", "DELETE", "/api/v1/security/keys/pk-1",
			nil, 200, map[string]string{"message": "deleted"}},

		// Services
		{"CreateService", "POST", "/api/v1/services",
			client.CreateServiceInput{
				Type:            "plausible",
				ServerUUID:      "srv-1",
				ProjectUUID:     "proj-1",
				EnvironmentName: "production",
				EnvironmentUUID: "env-1",
			}, 201, map[string]interface{}{"uuid": "svc-1", "domains": []string{}}},
		{"GetService", "GET", "/api/v1/services/svc-1",
			nil, 200, map[string]interface{}{"uuid": "svc-1", "name": "plausible"}},
		{"DeleteService", "DELETE", "/api/v1/services/svc-1",
			nil, 200, map[string]string{"message": "deleted"}},

		// Environment Variables (application)
		{"CreateEnvVar_Application", "POST", "/api/v1/applications/app-1/envs",
			client.EnvironmentVariable{Key: "DB_HOST", Value: "localhost", IsPreview: false},
			201, map[string]string{"uuid": "env-1"}},
		{"ListEnvVars_Application", "GET", "/api/v1/applications/app-1/envs",
			nil, 200, []map[string]interface{}{{"uuid": "env-1", "key": "DB_HOST", "value": "localhost"}}},
		{"DeleteEnvVar_Application", "DELETE", "/api/v1/applications/app-1/envs/env-1",
			nil, 200, map[string]string{"message": "deleted"}},

		// Database Backups
		{"CreateBackup", "POST", "/api/v1/databases/db-1/backups",
			client.CreateDatabaseBackupInput{Frequency: "0 2 * * *", Enabled: true},
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
			client.CreateEnvironmentInput{Name: "staging"},
			201, map[string]interface{}{"name": "staging"}},
		{"DeleteEnvironment", "DELETE", "/api/v1/projects/proj-1/environments/staging",
			nil, 200, map[string]string{"message": "deleted"}},
		{"GetEnvironment", "GET", "/api/v1/projects/proj-1/staging",
			nil, 200, map[string]interface{}{"id": 1, "name": "staging"}},

		// Applications (additional types)
		{"CreateDockerfileApp", "POST", "/api/v1/applications/dockerfile",
			client.CreateDockerfileAppInput{
				ProjectUUID:        "proj-1",
				ServerUUID:         "srv-1",
				EnvironmentName:    "production",
				EnvironmentUUID:    "env-1",
				DockerfileLocation: "FROM nginx",
				PortsExposes:       "80",
			}, 201, map[string]string{"uuid": "app-1"}},
		{"CreateGitHubAppApp", "POST", "/api/v1/applications/private-github-app",
			client.CreateGitHubAppInput{
				ProjectUUID:     "proj-1",
				ServerUUID:      "srv-1",
				EnvironmentName: "production",
				EnvironmentUUID: "env-1",
				GitHubAppUUID:   "gh-1",
				GitRepository:   "owner/repo",
				GitBranch:       "main",
				BuildPack:       "nixpacks",
				PortsExposes:    "3000",
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
			client.CreateScheduledTaskInput{Name: "cleanup", Command: "rm -rf /tmp/*", Frequency: "0 * * * *"},
			201, map[string]interface{}{"uuid": "task-1", "message": "created"}},
		{"UpdateAppTask", "PATCH", "/api/v1/applications/app-1/scheduled-tasks/task-1",
			client.UpdateScheduledTaskInput{Name: &updatedTaskName},
			200, map[string]interface{}{"uuid": "task-1", "message": "updated"}},
		{"DeleteAppTask", "DELETE", "/api/v1/applications/app-1/scheduled-tasks/task-1",
			nil, 200, map[string]string{"message": "deleted"}},
		{"ListAppTaskExecs", "GET", "/api/v1/applications/app-1/scheduled-tasks/task-1/executions",
			nil, 200, []map[string]interface{}{{"id": 1, "status": "success"}}},

		// Storages (application)
		{"ListAppStorages", "GET", "/api/v1/applications/app-1/storages",
			nil, 200, []map[string]interface{}{{"id": 1, "name": "data"}}},
		{"CreateAppStorage", "POST", "/api/v1/applications/app-1/storages",
			client.CreateStorageInput{Type: "persistent", Name: "data", MountPath: "/data", HostPath: "/host/data"},
			201, map[string]interface{}{"id": 1, "message": "created"}},
		{"UpdateAppStorage", "PATCH", "/api/v1/applications/app-1/storages",
			client.UpdateStorageInput{UUID: &appStorageUUID, Type: "persistent", Name: &updatedAppStorageName},
			200, map[string]interface{}{"id": 1, "message": "updated"}},
		{"DeleteAppStorage", "DELETE", "/api/v1/applications/app-1/storages/stor-1",
			nil, 200, map[string]string{"message": "deleted"}},

		// App env bulk
		{"BulkUpdateEnvVars_Application", "PATCH", "/api/v1/applications/app-1/envs/bulk",
			client.BulkEnvVarInput{Variables: []client.EnvVarEntry{{Key: "K1", Value: "V1"}}},
			200, map[string]string{"message": "updated"}},

		// Database environment variables
		{"ListEnvVars_Database", "GET", "/api/v1/databases/db-1/envs",
			nil, 200, []map[string]interface{}{{"uuid": "env-1", "key": "PG_HOST", "value": "localhost"}}},
		{"CreateEnvVar_Database", "POST", "/api/v1/databases/db-1/envs",
			client.EnvironmentVariable{Key: "PG_HOST", Value: "localhost", IsPreview: false},
			201, map[string]string{"uuid": "env-1"}},
		{"UpdateEnvVar_Database", "PATCH", "/api/v1/databases/db-1/envs",
			client.EnvironmentVariable{Key: "PG_HOST", Value: "db.local"},
			200, map[string]string{"message": "updated"}},
		{"DeleteEnvVar_Database", "DELETE", "/api/v1/databases/db-1/envs/env-1",
			nil, 200, map[string]string{"message": "deleted"}},
		{"BulkUpdateEnvVars_Database", "PATCH", "/api/v1/databases/db-1/envs/bulk",
			client.BulkEnvVarInput{Variables: []client.EnvVarEntry{{Key: "K1", Value: "V1"}}},
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
			client.CreateStorageInput{Type: "persistent", Name: "pgdata", MountPath: "/var/lib/postgresql", HostPath: "/host/pgdata"},
			201, map[string]interface{}{"id": 1, "message": "created"}},
		{"UpdateDbStorage", "PATCH", "/api/v1/databases/db-1/storages",
			client.UpdateStorageInput{UUID: &dbStorageUUID, Type: "persistent", Name: &updatedDBStorageName},
			200, map[string]interface{}{"id": 1, "message": "updated"}},
		{"DeleteDbStorage", "DELETE", "/api/v1/databases/db-1/storages/stor-1",
			nil, 200, map[string]string{"message": "deleted"}},

		// Database backup executions
		{"ListBackupExecs", "GET", "/api/v1/databases/db-1/backups/bak-1/executions",
			nil, 200, []map[string]interface{}{{"id": 1, "status": "success"}}},
		{"DeleteBackupExec", "DELETE", "/api/v1/databases/db-1/backups/bak-1/executions/exec-1",
			nil, 200, map[string]string{"message": "deleted"}},

		// Services extras
		{"ListEnvVars_Service", "GET", "/api/v1/services/svc-1/envs",
			nil, 200, []map[string]interface{}{{"uuid": "env-1", "key": "SECRET", "value": "val"}}},
		{"BulkUpdateEnvVars_Service", "PATCH", "/api/v1/services/svc-1/envs/bulk",
			client.BulkEnvVarInput{Variables: []client.EnvVarEntry{{Key: "K1", Value: "V1"}}},
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
			client.CreateScheduledTaskInput{Name: "cleanup", Command: "rm -rf /tmp/*", Frequency: "0 * * * *"},
			201, map[string]interface{}{"uuid": "task-1", "message": "created"}},
		{"UpdateSvcTask", "PATCH", "/api/v1/services/svc-1/scheduled-tasks/task-1",
			client.UpdateScheduledTaskInput{Name: &updatedTaskName},
			200, map[string]interface{}{"uuid": "task-1", "message": "updated"}},
		{"DeleteSvcTask", "DELETE", "/api/v1/services/svc-1/scheduled-tasks/task-1",
			nil, 200, map[string]string{"message": "deleted"}},
		{"ListSvcTaskExecs", "GET", "/api/v1/services/svc-1/scheduled-tasks/task-1/executions",
			nil, 200, []map[string]interface{}{{"id": 1, "status": "success"}}},

		// Storages (service)
		{"ListSvcStorages", "GET", "/api/v1/services/svc-1/storages",
			nil, 200, []map[string]interface{}{{"id": 1, "name": "svc-data"}}},
		{"CreateSvcStorage", "POST", "/api/v1/services/svc-1/storages",
			client.CreateStorageInput{Type: "persistent", Name: "svc-data", MountPath: "/data", HostPath: "/host/svc-data"},
			201, map[string]interface{}{"id": 1, "message": "created"}},
		{"UpdateSvcStorage", "PATCH", "/api/v1/services/svc-1/storages",
			client.UpdateStorageInput{UUID: &svcStorageUUID, Type: "persistent", Name: &updatedSvcStorageName},
			200, map[string]interface{}{"id": 1, "message": "updated"}},
		{"DeleteSvcStorage", "DELETE", "/api/v1/services/svc-1/storages/stor-1",
			nil, 200, map[string]string{"message": "deleted"}},

		// Cloud Tokens
		{"ListCloudTokens", "GET", "/api/v1/cloud-tokens",
			nil, 200, []map[string]interface{}{{"id": 1, "name": "my-token"}}},
		{"CreateCloudToken", "POST", "/api/v1/cloud-tokens",
			client.CreateCloudTokenInput{Name: "my-token", Provider: "hetzner", Token: "secret-token"},
			201, map[string]interface{}{"uuid": "ct-1"}},
		{"GetCloudToken", "GET", "/api/v1/cloud-tokens/ct-1",
			nil, 200, map[string]interface{}{"uuid": "ct-1", "name": "my-token", "provider": "hetzner"}},
		{"UpdateCloudToken", "PATCH", "/api/v1/cloud-tokens/ct-1",
			client.UpdateCloudTokenInput{Name: &updatedCloudTokenName},
			200, map[string]interface{}{"uuid": "ct-1"}},
		{"DeleteCloudToken", "DELETE", "/api/v1/cloud-tokens/ct-1",
			nil, 200, map[string]string{"message": "deleted"}},
		{"ValidateCloudToken", "POST", "/api/v1/cloud-tokens/ct-1/validate",
			nil, 200, map[string]interface{}{"valid": true}},

		// GitHub Apps
		{"ListGitHubApps", "GET", "/api/v1/github-apps",
			nil, 200, []map[string]interface{}{{"id": 1, "uuid": "gh-1", "name": "my-gh-app", "api_url": "https://api.github.com", "html_url": "https://github.com", "app_id": 12345, "installation_id": 67890, "client_id": "Iv1.abc123", "private_key_id": 1, "is_system_wide": false, "team_id": 1}}},
		{"CreateGitHubApp", "POST", "/api/v1/github-apps",
			client.CreateGitHubAppIntegrationInput{
				Name:           "my-gh-app",
				APIURL:         "https://api.github.com",
				HTMLURL:        "https://github.com",
				AppID:          12345,
				InstallationID: 67890,
				ClientID:       "Iv1.abc123",
				ClientSecret:   "secret",
				PrivateKeyUUID: "pk-1",
			},
			201, map[string]interface{}{"id": 1, "uuid": "gh-1", "name": "my-gh-app"}},
		{"UpdateGitHubApp", "PATCH", "/api/v1/github-apps/1",
			client.UpdateGitHubAppIntegrationInput{Name: &updatedGitHubAppName, ClientSecret: &updatedGitHubAppSecret},
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
			client.CreateHetznerServerInput{
				Name:                   "hetzner-srv",
				PrivateKeyUUID:         "pk-1",
				CloudProviderTokenUUID: "tok-1",
				ServerType:             "cx11",
				Location:               "fsn1",
				Image:                  "ubuntu-22.04",
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
				w.Write([]byte("v4.1.0"))
			})

			srv := httptest.NewServer(withSpecValidation(t, "coolify-v4", mux, false, v))
			defer srv.Close()

			var reqBody io.Reader = http.NoBody
			if ep.body != nil {
				data, err := json.Marshal(ep.body)
				if err != nil {
					t.Fatalf("marshaling request body: %v", err)
				}
				reqBody = bytes.NewReader(data)
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
