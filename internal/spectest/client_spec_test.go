package spectest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestClientEndpoints_SpecCompliance validates that every API endpoint
// our client uses matches the OpenAPI spec. This catches wrong methods,
// wrong paths, wrong request bodies, and wrong response shapes.
func TestClientEndpoints_SpecCompliance(t *testing.T) {
	t.Parallel()

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
			}, 200, map[string]string{"uuid": "db-1"}},
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
	}

	for _, ep := range endpoints {
		ep := ep // capture range variable
		t.Run(ep.name, func(t *testing.T) {
			// Not parallel: libopenapi-validator has internal state
			// that races when shared across goroutines.

			// Create a mock server that returns the expected response.
			mux := http.NewServeMux()
			pattern := fmt.Sprintf("%s %s", ep.method, ep.path)
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

			srv := httptest.NewServer(WithSpecAudit(t, "coolify-v4", mux))
			defer srv.Close()

			// Send the request.
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
