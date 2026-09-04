package client

import (
	"context"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_RetryOn429(t *testing.T) {
	t.Parallel()
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&attempts, 1)
		if n <= 2 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	err := c.do(context.Background(), http.MethodGet, "/test", nil, nil)
	if err != nil {
		t.Fatalf("expected success after retries, got: %v", err)
	}
	if got := atomic.LoadInt32(&attempts); got != 3 {
		t.Fatalf("expected 3 attempts, got %d", got)
	}
}

func TestClient_NotFoundError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	err := c.do(context.Background(), http.MethodGet, "/resource", nil, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsNotFound(err) {
		t.Fatalf("expected NotFoundError, got %T: %v", err, err)
	}
}

func TestClient_Timeout(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Block longer than the client timeout; stop when the client disconnects
		// so the test server shuts down quickly.
		select {
		case <-time.After(60 * time.Second):
		case <-r.Context().Done():
		}
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	// Shorten the timeout so the test finishes quickly.
	c.HTTPClient.Timeout = 2 * time.Second

	start := time.Now()
	err := c.do(context.Background(), http.MethodGet, "/slow", nil, nil)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if elapsed > 30*time.Second {
		t.Fatalf("expected timeout within a few seconds, took %s", elapsed)
	}
}

func TestClient_AuthHeader(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := r.Header.Get("Authorization")
		if got != "Bearer my-secret-token" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(srv.URL, "my-secret-token")
	err := c.do(context.Background(), http.MethodGet, "/auth-check", nil, nil)
	if err != nil {
		t.Fatalf("expected success with correct Bearer token, got: %v", err)
	}
}

func TestClient_UserAgent(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := r.Header.Get("User-Agent")
		if got != "terraform-provider-coolify" {
			http.Error(w, "bad user-agent: "+got, http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	err := c.do(context.Background(), http.MethodGet, "/ua-check", nil, nil)
	if err != nil {
		t.Fatalf("expected success with correct User-Agent, got: %v", err)
	}
}

func TestClient_ListProjects(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/projects" {
			t.Errorf("expected path /api/v1/projects, got %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("expected Authorization Bearer test-token, got %s", got)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]Project{
			{UUID: "proj-1", Name: "Alpha", Description: "First project"},
			{UUID: "proj-2", Name: "Beta", Description: "Second project"},
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	projects, err := c.ListProjects(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(projects) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(projects))
	}
	if projects[0].Name != "Alpha" {
		t.Errorf("expected first project name Alpha, got %s", projects[0].Name)
	}
	if projects[1].Name != "Beta" {
		t.Errorf("expected second project name Beta, got %s", projects[1].Name)
	}
}

func TestClient_GetProject(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/projects/proj-uuid-123" {
			t.Errorf("expected path /api/v1/projects/proj-uuid-123, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Project{
			UUID:        "proj-uuid-123",
			Name:        "My Project",
			Description: "A test project",
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	proj, err := c.GetProject(context.Background(), "proj-uuid-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if proj.UUID != "proj-uuid-123" {
		t.Errorf("expected UUID proj-uuid-123, got %s", proj.UUID)
	}
	if proj.Name != "My Project" {
		t.Errorf("expected Name 'My Project', got %s", proj.Name)
	}
	if proj.Description != "A test project" {
		t.Errorf("expected Description 'A test project', got %s", proj.Description)
	}
}

func TestClient_CreateProject(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/projects" {
			t.Errorf("expected path /api/v1/projects, got %s", r.URL.Path)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", ct)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}
		var input CreateProjectInput
		if err := json.Unmarshal(body, &input); err != nil {
			t.Fatalf("failed to unmarshal request body: %v", err)
		}
		if input.Name != "New Project" {
			t.Errorf("expected body name 'New Project', got %s", input.Name)
		}
		if input.Description != "Created via test" {
			t.Errorf("expected body description 'Created via test', got %s", input.Description)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(Project{UUID: "new-proj-uuid"})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	proj, err := c.CreateProject(context.Background(), CreateProjectInput{
		Name:        "New Project",
		Description: "Created via test",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if proj.UUID != "new-proj-uuid" {
		t.Errorf("expected UUID new-proj-uuid, got %s", proj.UUID)
	}
}

func TestClient_DeleteProject(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/projects/proj-del-uuid" {
			t.Errorf("expected path /api/v1/projects/proj-del-uuid, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	err := c.DeleteProject(context.Background(), "proj-del-uuid")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestClient_GetProject_NotFound(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	_, err := c.GetProject(context.Background(), "nonexistent-uuid")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsNotFound(err) {
		t.Fatalf("expected NotFoundError, got %T: %v", err, err)
	}
}

func TestClient_CreateProject_ServerError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	// Shorten retry delays so the test does not wait too long.
	c.HTTPClient.Timeout = 10 * time.Second
	_, err := c.CreateProject(context.Background(), CreateProjectInput{
		Name: "Will Fail",
	})
	if err == nil {
		t.Fatal("expected error on 500 response, got nil")
	}
	// The retryable client retries on 500 and eventually gives up.
	// The error message should indicate the retry exhaustion.
	got := err.Error()
	assert.True(t, strings.Contains(got, "giving up") || strings.Contains(got, "500"),
		"expected error message to mention retry exhaustion or status 500, got: %s", got)
}

func TestClient_ListServers(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/servers" {
			t.Errorf("expected path /api/v1/servers, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]Server{
			{
				UUID:           "srv-1",
				Name:           "Server One",
				Description:    "Primary server",
				IP:             "10.0.0.1",
				Port:           22,
				User:           "root",
				PrivateKeyUUID: "pk-1",
				IsBuildServer:  false,
				IsReachable:    true,
				IsUsable:       true,
			},
			{
				UUID:           "srv-2",
				Name:           "Server Two",
				Description:    "Build server",
				IP:             "10.0.0.2",
				Port:           2222,
				User:           "deploy",
				PrivateKeyUUID: "pk-2",
				IsBuildServer:  true,
				IsReachable:    true,
				IsUsable:       false,
			},
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	servers, err := c.ListServers(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(servers) != 2 {
		t.Fatalf("expected 2 servers, got %d", len(servers))
	}
	s1 := servers[0]
	if s1.UUID != "srv-1" {
		t.Errorf("expected UUID srv-1, got %s", s1.UUID)
	}
	if s1.Name != "Server One" {
		t.Errorf("expected Name 'Server One', got %s", s1.Name)
	}
	if s1.IP != "10.0.0.1" {
		t.Errorf("expected IP 10.0.0.1, got %s", s1.IP)
	}
	if s1.Port != 22 {
		t.Errorf("expected Port 22, got %d", s1.Port)
	}
	if s1.IsBuildServer {
		t.Error("expected IsBuildServer false for server 1")
	}
	if !s1.IsReachable {
		t.Error("expected IsReachable true for server 1")
	}
	if !s1.IsUsable {
		t.Error("expected IsUsable true for server 1")
	}
	s2 := servers[1]
	if s2.UUID != "srv-2" {
		t.Errorf("expected UUID srv-2, got %s", s2.UUID)
	}
	if !s2.IsBuildServer {
		t.Error("expected IsBuildServer true for server 2")
	}
	if s2.Port != 2222 {
		t.Errorf("expected Port 2222, got %d", s2.Port)
	}
	if s2.IsUsable {
		t.Error("expected IsUsable false for server 2")
	}
}

func TestClient_CreatePublicApplication(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/applications/public" {
			t.Errorf("expected path /api/v1/applications/public, got %s", r.URL.Path)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}
		var input CreatePublicAppInput
		if err := json.Unmarshal(body, &input); err != nil {
			t.Fatalf("failed to unmarshal request body: %v", err)
		}
		if input.ProjectUUID != "proj-1" {
			t.Errorf("expected project_uuid proj-1, got %s", input.ProjectUUID)
		}
		if input.ServerUUID != "srv-1" {
			t.Errorf("expected server_uuid srv-1, got %s", input.ServerUUID)
		}
		if input.GitRepository != "https://github.com/example/repo" {
			t.Errorf("expected git_repository, got %s", input.GitRepository)
		}
		if input.GitBranch != "main" {
			t.Errorf("expected git_branch main, got %s", input.GitBranch)
		}
		if input.BuildPack != "dockerfile" {
			t.Errorf("expected build_pack dockerfile, got %s", input.BuildPack)
		}
		if input.PortsExposes != "3000" {
			t.Errorf("expected ports_exposes 3000, got %s", input.PortsExposes)
		}

		// Verify Domains is sent as "domains" key (spec requirement)
		var raw map[string]interface{}
		if err := json.Unmarshal(body, &raw); err != nil {
			t.Fatalf("failed to unmarshal raw body: %v", err)
		}
		if _, ok := raw["domains"]; !ok {
			t.Error("expected 'domains' key in JSON body (Domains field)")
		}
		if _, ok := raw["fqdn"]; ok {
			t.Error("unexpected 'fqdn' key in JSON body (should be 'domains')")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(Application{
			UUID:          "app-new-uuid",
			Name:          "my-app",
			GitRepository: input.GitRepository,
			GitBranch:     input.GitBranch,
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	app, err := c.CreatePublicApplication(context.Background(), CreatePublicAppInput{
		ProjectUUID:     "proj-1",
		ServerUUID:      "srv-1",
		EnvironmentName: "production",
		GitRepository:   "https://github.com/example/repo",
		GitBranch:       "main",
		BuildPack:       "dockerfile",
		PortsExposes:    "3000",
		Domains:         "https://app.example.com",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if app.UUID != "app-new-uuid" {
		t.Errorf("expected UUID app-new-uuid, got %s", app.UUID)
	}
}

func TestClient_CreateDatabase_Postgresql(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/databases/postgresql" {
			t.Errorf("expected path /api/v1/databases/postgresql, got %s", r.URL.Path)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}
		var input CreatePostgresqlInput
		if err := json.Unmarshal(body, &input); err != nil {
			t.Fatalf("failed to unmarshal request body: %v", err)
		}
		if input.ServerUUID != "srv-1" {
			t.Errorf("expected server_uuid srv-1, got %s", input.ServerUUID)
		}
		if input.ProjectUUID != "proj-1" {
			t.Errorf("expected project_uuid proj-1, got %s", input.ProjectUUID)
		}
		if input.EnvironmentName != "production" {
			t.Errorf("expected environment_name production, got %s", input.EnvironmentName)
		}
		if input.PostgresUser != "pgadmin" {
			t.Errorf("expected postgres_user pgadmin, got %s", input.PostgresUser)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(Database{UUID: "db-pg-new"})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	db, err := c.CreateDatabase(context.Background(), "postgresql", CreatePostgresqlInput{
		CreateDatabaseBaseInput: CreateDatabaseBaseInput{
			ServerUUID:      "srv-1",
			ProjectUUID:     "proj-1",
			EnvironmentName: "production",
		},
		PostgresUser: "pgadmin",
		PostgresDB:   "testdb",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if db.UUID != "db-pg-new" {
		t.Errorf("expected UUID db-pg-new, got %s", db.UUID)
	}
}

func TestClient_ContextCancellation(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Should never reach here if context is cancelled.
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	_, err := c.ListProjects(ctx)
	if err == nil {
		t.Fatal("expected error with cancelled context, got nil")
	}
}

// --- Servers ---

func TestClient_GetServer(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/servers/srv-uuid-1", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Server{
			UUID:           "srv-uuid-1",
			Name:           "Main Server",
			Description:    "Primary",
			IP:             "192.168.1.1",
			Port:           22,
			User:           "root",
			PrivateKeyUUID: "pk-abc",
			IsBuildServer:  true,
			IsReachable:    true,
			IsUsable:       true,
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	s, err := c.GetServer(context.Background(), "srv-uuid-1")
	require.NoError(t, err)
	assert.Equal(t, "srv-uuid-1", s.UUID)
	assert.Equal(t, "Main Server", s.Name)
	assert.Equal(t, "Primary", s.Description)
	assert.Equal(t, "192.168.1.1", s.IP)
	assert.Equal(t, 22, s.Port)
	assert.Equal(t, "root", s.User)
	assert.Equal(t, "pk-abc", s.PrivateKeyUUID)
	assert.True(t, s.IsBuildServer)
	assert.True(t, s.IsReachable)
	assert.True(t, s.IsUsable)
}

func TestClient_GetServer_WithSettings(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/servers/test-uuid", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"uuid": "test-uuid",
			"name": "Settings Server",
			"ip": "10.0.0.1",
			"port": 22,
			"is_build_server": false,
			"is_reachable": true,
			"is_usable": true,
			"settings": {
				"concurrent_builds": 4,
				"dynamic_timeout": 7200,
				"deployment_queue_limit": 10,
				"server_disk_usage_notification_threshold": 90,
				"server_disk_usage_check_frequency": "0 * * * *"
			}
		}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	s, err := c.GetServer(context.Background(), "test-uuid")
	require.NoError(t, err)
	assert.Equal(t, "test-uuid", s.UUID)
	assert.Equal(t, "Settings Server", s.Name)
	require.NotNil(t, s.Settings)
	assert.Equal(t, 4, s.Settings.ConcurrentBuilds)
	assert.Equal(t, 7200, s.Settings.DynamicTimeout)
	assert.Equal(t, 10, s.Settings.DeploymentQueueLimit)
	assert.Equal(t, 90, s.Settings.ServerDiskUsageNotificationThreshold)
	assert.Equal(t, "0 * * * *", s.Settings.ServerDiskUsageCheckFrequency)
}

func TestClient_GetServer_NotFound(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	_, err := c.GetServer(context.Background(), "test-uuid")
	require.Error(t, err)
	assert.True(t, IsNotFound(err))
}

func TestClient_GetServer_ServerError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"message": "internal error"}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	c.HTTPClient.Timeout = 10 * time.Second
	_, err := c.GetServer(context.Background(), "test-uuid")
	require.Error(t, err)
	assert.False(t, IsNotFound(err))
	// The retryable client retries on 500 and eventually gives up.
	got := err.Error()
	assert.True(t, strings.Contains(got, "giving up") || strings.Contains(got, "500") || strings.Contains(got, "internal error"),
		"expected error to mention retry exhaustion, status 500, or the API message, got: %s", got)
}

func TestClient_CreateServer(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/servers", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var input CreateServerInput
		require.NoError(t, json.Unmarshal(body, &input))
		assert.Equal(t, "New Server", input.Name)
		assert.Equal(t, "10.0.0.5", input.IP)
		assert.Equal(t, 2222, input.Port)
		assert.Equal(t, "deploy", input.User)
		assert.Equal(t, "pk-99", input.PrivateKeyUUID)
		require.NotNil(t, input.IsBuildServer)
		assert.True(t, *input.IsBuildServer)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(Server{UUID: "srv-new"})
	}))
	defer srv.Close()

	buildServer := true
	c := New(srv.URL, "test-token")
	s, err := c.CreateServer(context.Background(), CreateServerInput{
		Name:           "New Server",
		IP:             "10.0.0.5",
		Port:           2222,
		User:           "deploy",
		PrivateKeyUUID: "pk-99",
		IsBuildServer:  &buildServer,
	})
	require.NoError(t, err)
	assert.Equal(t, "srv-new", s.UUID)
}

func TestClient_UpdateServer(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPatch, r.Method)
		assert.Equal(t, "/api/v1/servers/srv-upd", r.URL.Path)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var input UpdateServerInput
		require.NoError(t, json.Unmarshal(body, &input))
		require.NotNil(t, input.Name)
		assert.Equal(t, "Updated", *input.Name)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Server{UUID: "srv-upd", Name: "Updated"})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	name := "Updated"
	s, err := c.UpdateServer(context.Background(), "srv-upd", UpdateServerInput{Name: &name})
	require.NoError(t, err)
	assert.Equal(t, "srv-upd", s.UUID)
	assert.Equal(t, "Updated", s.Name)
}

func TestClient_DeleteServer(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "/api/v1/servers/srv-del", r.URL.Path)
		assert.Equal(t, "force=true", r.URL.RawQuery)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	err := c.DeleteServer(context.Background(), "srv-del")
	require.NoError(t, err)
}

// --- Private Keys ---

func TestClient_ListPrivateKeys(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/security/keys", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]PrivateKey{
			{UUID: "pk-1", Name: "key-one", PrivateKey: "ssh-rsa AAA", PublicKey: "ssh-rsa PUB1", Fingerprint: "SHA256:fp1", IsGitRelated: true},
			{UUID: "pk-2", Name: "key-two", PrivateKey: "ssh-ed25519 BBB", PublicKey: "ssh-ed25519 PUB2", Fingerprint: "SHA256:fp2", IsGitRelated: false},
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	keys, err := c.ListPrivateKeys(context.Background())
	require.NoError(t, err)
	require.Len(t, keys, 2)
	assert.Equal(t, "pk-1", keys[0].UUID)
	assert.Equal(t, "key-one", keys[0].Name)
	assert.Equal(t, "ssh-rsa PUB1", keys[0].PublicKey)
	assert.Equal(t, "SHA256:fp1", keys[0].Fingerprint)
	assert.True(t, keys[0].IsGitRelated)
	assert.Equal(t, "pk-2", keys[1].UUID)
	assert.Equal(t, "ssh-ed25519 PUB2", keys[1].PublicKey)
	assert.Equal(t, "SHA256:fp2", keys[1].Fingerprint)
	assert.False(t, keys[1].IsGitRelated)
}

func TestClient_GetPrivateKey(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/security/keys/pk-uuid-1", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(PrivateKey{
			UUID:         "pk-uuid-1",
			Name:         "my-key",
			Description:  "Test key",
			PrivateKey:   "ssh-rsa AAAA",
			PublicKey:    "ssh-rsa AAAA-public",
			Fingerprint:  "SHA256:client-test-fingerprint",
			IsGitRelated: true,
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	k, err := c.GetPrivateKey(context.Background(), "pk-uuid-1")
	require.NoError(t, err)
	assert.Equal(t, "pk-uuid-1", k.UUID)
	assert.Equal(t, "my-key", k.Name)
	assert.Equal(t, "Test key", k.Description)
	assert.Equal(t, "ssh-rsa AAAA", k.PrivateKey)
	assert.Equal(t, "ssh-rsa AAAA-public", k.PublicKey)
	assert.Equal(t, "SHA256:client-test-fingerprint", k.Fingerprint)
	assert.True(t, k.IsGitRelated)
}

func TestClient_CreatePrivateKey(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/security/keys", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var input CreatePrivateKeyInput
		require.NoError(t, json.Unmarshal(body, &input))
		assert.Equal(t, "deploy-key", input.Name)
		assert.Equal(t, "ssh-ed25519 NEWKEY", input.PrivateKey)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(PrivateKey{UUID: "pk-new"})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	k, err := c.CreatePrivateKey(context.Background(), CreatePrivateKeyInput{
		Name:       "deploy-key",
		PrivateKey: "ssh-ed25519 NEWKEY",
	})
	require.NoError(t, err)
	assert.Equal(t, "pk-new", k.UUID)
}

func TestClient_DeletePrivateKey(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "/api/v1/security/keys/pk-del", r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	err := c.DeletePrivateKey(context.Background(), "pk-del")
	require.NoError(t, err)
}

// --- Databases ---

func TestClient_GetDatabase(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/databases/db-uuid-1", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		port := int64(5432)
		json.NewEncoder(w).Encode(Database{
			UUID:         "db-uuid-1",
			Name:         "my-pg",
			Type:         "postgresql",
			IsPublic:     true,
			PublicPort:   &port,
			PostgresUser: "admin",
			PostgresDB:   "appdb",
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	db, err := c.GetDatabase(context.Background(), "db-uuid-1")
	require.NoError(t, err)
	assert.Equal(t, "db-uuid-1", db.UUID)
	assert.Equal(t, "my-pg", db.Name)
	assert.Equal(t, "postgresql", db.Type)
	assert.True(t, db.IsPublic)
	require.NotNil(t, db.PublicPort)
	assert.Equal(t, int64(5432), *db.PublicPort)
	assert.Equal(t, "admin", db.PostgresUser)
	assert.Equal(t, "appdb", db.PostgresDB)
}

func TestClient_CreateDatabase_Mysql(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/databases/mysql", r.URL.Path)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var input CreateMysqlInput
		require.NoError(t, json.Unmarshal(body, &input))
		assert.Equal(t, "srv-1", input.ServerUUID)
		assert.Equal(t, "proj-1", input.ProjectUUID)
		assert.Equal(t, "production", input.EnvironmentName)
		assert.Equal(t, "myuser", input.MysqlUser)
		assert.Equal(t, "mydb", input.MysqlDatabase)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(Database{UUID: "db-mysql-new"})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	db, err := c.CreateDatabase(context.Background(), "mysql", CreateMysqlInput{
		CreateDatabaseBaseInput: CreateDatabaseBaseInput{
			ServerUUID:      "srv-1",
			ProjectUUID:     "proj-1",
			EnvironmentName: "production",
		},
		MysqlUser:     "myuser",
		MysqlDatabase: "mydb",
	})
	require.NoError(t, err)
	assert.Equal(t, "db-mysql-new", db.UUID)
}

func TestClient_UpdateDatabase(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPatch, r.Method)
		assert.Equal(t, "/api/v1/databases/db-upd", r.URL.Path)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var input UpdateDatabaseInput
		require.NoError(t, json.Unmarshal(body, &input))
		require.NotNil(t, input.Name)
		assert.Equal(t, "renamed-db", *input.Name)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Database{UUID: "db-upd", Name: "renamed-db"})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	name := "renamed-db"
	db, err := c.UpdateDatabase(context.Background(), "db-upd", UpdateDatabaseInput{Name: &name})
	require.NoError(t, err)
	assert.Equal(t, "db-upd", db.UUID)
	assert.Equal(t, "renamed-db", db.Name)
}

func TestClient_UpdateDatabase_StripsDisallowedKeys(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var raw map[string]interface{}
		require.NoError(t, json.Unmarshal(body, &raw))
		for _, k := range DatabaseUpdateDisallowedJSONKeys {
			_, ok := raw[k]
			assert.False(t, ok, "PATCH body must not contain %s", k)
		}
		require.Contains(t, raw, "limits_memory")
		assert.Equal(t, "512M", raw["limits_memory"])
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Database{UUID: "db-san", Name: "sanitized"})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	mem := "512M"
	en := true
	_, err := c.UpdateDatabase(context.Background(), "db-san", UpdateDatabaseInput{
		LimitsMemory:      &mem,
		IsLogDrainEnabled: &en,
		EnableSSL:         &en,
	})
	require.NoError(t, err)
}

func TestClient_DeleteDatabase(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "/api/v1/databases/db-del", r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	err := c.DeleteDatabase(context.Background(), "db-del")
	require.NoError(t, err)
}

func TestClient_StartDatabase(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/databases/db-start/start", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	err := c.StartDatabase(context.Background(), "db-start")
	require.NoError(t, err)
}

func TestClient_StopDatabase(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/databases/db-stop/stop", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	err := c.StopDatabase(context.Background(), "db-stop")
	require.NoError(t, err)
}

// --- Environment Variables ---

func TestClient_CreateEnvVar_Application(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/applications/app-1/envs", r.URL.Path)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var ev EnvironmentVariable
		require.NoError(t, json.Unmarshal(body, &ev))
		assert.Equal(t, "DATABASE_URL", ev.Key)
		assert.Equal(t, "postgres://localhost/db", ev.Value)
		assert.True(t, ev.IsBuild)

		var raw map[string]any
		require.NoError(t, json.Unmarshal(body, &raw))
		_, hasBuildtime := raw["is_buildtime"]
		assert.True(t, hasBuildtime, "expected JSON key 'is_buildtime'")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(CreateEnvVarResponse{UUID: "env-new"})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	createIsBuild := true
	resp, err := c.CreateEnvVar(context.Background(), "applications", "app-1", EnvironmentVariable{
		Key:     "DATABASE_URL",
		Value:   "postgres://localhost/db",
		IsBuild: true,
	}, &EnvVarWriteOpts{IsBuild: &createIsBuild})
	require.NoError(t, err)
	assert.Equal(t, "env-new", resp.UUID)
}

func TestClient_CreateEnvVar_Application_OmitsBuildtimeWhenUnset(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/applications/app-1/envs", r.URL.Path)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var bodyMap map[string]any
		require.NoError(t, json.Unmarshal(body, &bodyMap))
		assert.Equal(t, "DATABASE_URL", bodyMap["key"])
		assert.Equal(t, "postgres://localhost/db", bodyMap["value"])
		assert.NotContains(t, bodyMap, "is_buildtime")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(CreateEnvVarResponse{UUID: "env-new"})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	resp, err := c.CreateEnvVar(context.Background(), "applications", "app-1", EnvironmentVariable{
		Key:     "DATABASE_URL",
		Value:   "postgres://localhost/db",
		IsBuild: true,
	}, nil)
	require.NoError(t, err)
	assert.Equal(t, "env-new", resp.UUID)
}

func TestClient_ListEnvVars_Application(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/applications/app-1/envs", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]EnvironmentVariable{
			{UUID: "ev-1", Key: "PORT", Value: "3000", IsPreview: false, IsBuild: false},
			{UUID: "ev-2", Key: "SECRET", Value: "abc", IsPreview: true, IsBuild: true},
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	vars, err := c.ListEnvVars(context.Background(), "applications", "app-1")
	require.NoError(t, err)
	require.Len(t, vars, 2)
	assert.Equal(t, "PORT", vars[0].Key)
	assert.Equal(t, "3000", vars[0].Value)
	assert.False(t, vars[0].IsPreview)
	assert.Equal(t, "SECRET", vars[1].Key)
	assert.True(t, vars[1].IsPreview)
	assert.True(t, vars[1].IsBuild)
}

func TestClient_PreferNonPreviewEnvVarsByKey(t *testing.T) {
	t.Parallel()
	vars := PreferNonPreviewEnvVarsByKey([]EnvironmentVariable{
		{UUID: "preview-1", Key: "APP_KEY", Value: "preview", IsPreview: true},
		{UUID: "runtime-1", Key: "APP_KEY", Value: "runtime", IsPreview: false},
		{UUID: "preview-2", Key: "APP_KEY", Value: "preview-late", IsPreview: true},
		{UUID: "preview-3", Key: "LOG_LEVEL", Value: "debug", IsPreview: true},
	})

	require.Len(t, vars, 2)
	assert.Equal(t, "runtime", vars["APP_KEY"].Value)
	assert.False(t, vars["APP_KEY"].IsPreview)
	assert.Equal(t, "debug", vars["LOG_LEVEL"].Value)
	assert.True(t, vars["LOG_LEVEL"].IsPreview)
}

func TestClient_PreserveEnvVarValue(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "prior-secret", PreserveEnvVarValue("", "prior-secret"))
	assert.Equal(t, "fresh-value", PreserveEnvVarValue("fresh-value", "prior-secret"))
	assert.Equal(t, "", PreserveEnvVarValue("", ""))
}

func TestClient_FindEnvVarByUUID(t *testing.T) {
	t.Parallel()
	vars := []EnvironmentVariable{
		{UUID: "first", Key: "A", Value: "1"},
		{UUID: "second", Key: "B", Value: "2"},
	}

	found, ok := FindEnvVarByUUID(vars, "second")
	require.True(t, ok)
	assert.Equal(t, "B", found.Key)
	assert.Equal(t, "2", found.Value)

	_, ok = FindEnvVarByUUID(vars, "missing")
	assert.False(t, ok)
}

func TestClient_DeleteEnvVar_Application(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "/api/v1/applications/app-1/envs/env-del", r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	err := c.DeleteEnvVar(context.Background(), "applications", "app-1", "env-del")
	require.NoError(t, err)
}

// --- Services ---

func TestClient_GetService(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/services/svc-uuid-1", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Service{
			UUID:            "svc-uuid-1",
			Name:            "my-service",
			Description:     "A test service",
			Type:            "wordpress",
			ServerUUID:      "srv-1",
			ProjectUUID:     "proj-1",
			EnvironmentName: "production",
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	svc, err := c.GetService(context.Background(), "svc-uuid-1")
	require.NoError(t, err)
	assert.Equal(t, "svc-uuid-1", svc.UUID)
	assert.Equal(t, "my-service", svc.Name)
	assert.Equal(t, "A test service", svc.Description)
	assert.Equal(t, "wordpress", svc.Type)
	assert.Equal(t, "srv-1", svc.ServerUUID)
	assert.Equal(t, "proj-1", svc.ProjectUUID)
	assert.Equal(t, "production", svc.EnvironmentName)
}

func TestClient_CreateService(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/services", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var input CreateServiceInput
		require.NoError(t, json.Unmarshal(body, &input))
		assert.Equal(t, "wordpress", input.Type)
		assert.Equal(t, "blog", input.Name)
		assert.Equal(t, "srv-1", input.ServerUUID)
		assert.Equal(t, "proj-1", input.ProjectUUID)
		assert.Equal(t, "production", input.EnvironmentName)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(Service{UUID: "svc-new"})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	svc, err := c.CreateService(context.Background(), CreateServiceInput{
		Type:            "wordpress",
		Name:            "blog",
		ServerUUID:      "srv-1",
		ProjectUUID:     "proj-1",
		EnvironmentName: "production",
	})
	require.NoError(t, err)
	assert.Equal(t, "svc-new", svc.UUID)
}

func TestClient_DeleteService(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "/api/v1/services/svc-del", r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	err := c.DeleteService(context.Background(), "svc-del")
	require.NoError(t, err)
}

func TestClient_UpdateService(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPatch, r.Method)
		assert.Equal(t, "/api/v1/services/svc-upd", r.URL.Path)

		var input UpdateServiceInput
		require.NoError(t, json.NewDecoder(r.Body).Decode(&input))
		require.NotNil(t, input.Name)
		assert.Equal(t, "Updated Name", *input.Name)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Service{UUID: "svc-upd", Name: "Updated Name", Type: "plausible"})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	updName := "Updated Name"
	svc, err := c.UpdateService(context.Background(), "svc-upd", UpdateServiceInput{Name: &updName})
	require.NoError(t, err)
	assert.Equal(t, "svc-upd", svc.UUID)
	assert.Equal(t, "Updated Name", svc.Name)
}

// --- Teams ---

func TestClient_GetTeam(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/teams/42", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Team{
			ID:          42,
			Name:        "backend-team",
			Description: "Backend engineers",
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	team, err := c.GetTeam(context.Background(), 42)
	require.NoError(t, err)
	assert.Equal(t, 42, team.ID)
	assert.Equal(t, "backend-team", team.Name)
	assert.Equal(t, "Backend engineers", team.Description)
}

func TestClient_ListTeamMembers(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/teams/7/members", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]TeamMember{
			{ID: 1, Name: "Alice", Email: "alice@example.com"},
			{ID: 2, Name: "Bob", Email: "bob@example.com"},
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	members, err := c.ListTeamMembers(context.Background(), 7)
	require.NoError(t, err)
	require.Len(t, members, 2)
	assert.Equal(t, "Alice", members[0].Name)
	assert.Equal(t, "alice@example.com", members[0].Email)
	assert.Equal(t, "Bob", members[1].Name)
	assert.Equal(t, "bob@example.com", members[1].Email)
}

// --- Deployments ---

func TestClient_ListDeployments(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/deployments", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]Deployment{
			{UUID: "dep-1", ID: 100, Status: "finished", ServerUUID: "srv-1"},
			{UUID: "dep-2", ID: 101, Status: "in_progress", ServerUUID: "srv-2"},
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	deps, err := c.ListDeployments(context.Background())
	require.NoError(t, err)
	require.Len(t, deps, 2)
	assert.Equal(t, "dep-1", deps[0].UUID)
	assert.Equal(t, 100, deps[0].ID)
	assert.Equal(t, "finished", deps[0].Status)
	assert.Equal(t, "dep-2", deps[1].UUID)
	assert.Equal(t, "in_progress", deps[1].Status)
}

func TestClient_ListDeployments_ObjectFallback(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/deployments", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		// Simulate Coolify sortBy('id') bug: non-sequential keys produce a JSON object.
		// Include key 10 to verify the fallback sorts numerically, not lexicographically.
		w.Write([]byte(`{"10":{"deployment_uuid":"dep-3","id":102,"status":"failed"},"2":{"deployment_uuid":"dep-2","id":101,"status":"queued"},"0":{"deployment_uuid":"dep-1","id":100,"status":"finished"}}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	deps, err := c.ListDeployments(context.Background())
	require.NoError(t, err)
	assert.Equal(t, []Deployment{
		{UUID: "dep-1", ID: 100, Status: "finished"},
		{UUID: "dep-2", ID: 101, Status: "queued"},
		{UUID: "dep-3", ID: 102, Status: "failed"},
	}, deps)
}

func TestClient_ListDeployments_InvalidJSON(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`not valid json`))
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	_, err := c.ListDeployments(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "listing deployments")
}

func TestClient_DeployByTag(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/deploy", r.URL.Path)
		assert.Equal(t, "v1.2.3", r.URL.Query().Get("tag"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var input DeployByTagInput
		require.NoError(t, json.Unmarshal(body, &input))
		assert.True(t, input.ForceRebuild)

		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	err := c.DeployByTag(context.Background(), "v1.2.3", DeployByTagInput{ForceRebuild: true})
	require.NoError(t, err)
}

// --- Applications (remaining) ---

func TestClient_ListApplications(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/applications", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]Application{
			{UUID: "app-1", Name: "App One", GitRepository: "https://github.com/org/one", GitBranch: "main"},
			{UUID: "app-2", Name: "App Two", GitRepository: "https://github.com/org/two", GitBranch: "develop"},
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	apps, err := c.ListApplications(context.Background())
	require.NoError(t, err)
	require.Len(t, apps, 2)
	assert.Equal(t, "app-1", apps[0].UUID)
	assert.Equal(t, "App One", apps[0].Name)
	assert.Equal(t, "https://github.com/org/one", apps[0].GitRepository)
	assert.Equal(t, "app-2", apps[1].UUID)
	assert.Equal(t, "App Two", apps[1].Name)
	assert.Equal(t, "develop", apps[1].GitBranch)
}

func TestClient_RestartApplication(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/applications/app-restart-1/restart", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(RestartApplicationResponse{
			DeploymentUUID: "dep-uuid-99",
			Message:        "Restarting",
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	resp, err := c.RestartApplication(context.Background(), "app-restart-1")
	require.NoError(t, err)
	assert.Equal(t, "dep-uuid-99", resp.DeploymentUUID)
	assert.Equal(t, "Restarting", resp.Message)
}

func TestClient_RestartApplication_SkipOmitsUUID(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/applications/app-restart-skip/restart", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Deployment already queued for this commit.",
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	resp, err := c.RestartApplication(context.Background(), "app-restart-skip")
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Empty(t, resp.DeploymentUUID)
	assert.Equal(t, "Deployment already queued for this commit.", resp.Message)
}

func TestClient_GetApplication(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/applications/app-get-1", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Application{
			UUID:               "app-get-1",
			Name:               "My App",
			Description:        "A test app",
			Domains:            "https://app.example.com",
			GitRepository:      "https://github.com/org/repo",
			GitBranch:          "main",
			BuildPack:          "nixpacks",
			DockerfileLocation: "/Dockerfile",
			InstallCommand:     "npm install",
			BuildCommand:       "npm run build",
			StartCommand:       "npm start",
			PortsExposes:       "3000",
			ServerUUID:         "srv-1",
			ProjectUUID:        "proj-1",
			Status:             "running",
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	app, err := c.GetApplication(context.Background(), "app-get-1")
	require.NoError(t, err)
	assert.Equal(t, "app-get-1", app.UUID)
	assert.Equal(t, "My App", app.Name)
	assert.Equal(t, "A test app", app.Description)
	assert.Equal(t, "https://app.example.com", app.Domains)
	assert.Equal(t, "https://github.com/org/repo", app.GitRepository)
	assert.Equal(t, "main", app.GitBranch)
	assert.Equal(t, "nixpacks", app.BuildPack)
	assert.Equal(t, "/Dockerfile", app.DockerfileLocation)
	assert.Equal(t, "npm install", app.InstallCommand)
	assert.Equal(t, "npm run build", app.BuildCommand)
	assert.Equal(t, "npm start", app.StartCommand)
	assert.Equal(t, "3000", app.PortsExposes)
	assert.Equal(t, "srv-1", app.ServerUUID)
	assert.Equal(t, "proj-1", app.ProjectUUID)
	assert.Equal(t, "running", app.Status)
}

func TestClient_UpdateApplication(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPatch, r.Method)
		assert.Equal(t, "/api/v1/applications/app-upd-1", r.URL.Path)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var input UpdateApplicationInput
		require.NoError(t, json.Unmarshal(body, &input))
		require.NotNil(t, input.Name)
		assert.Equal(t, "Updated App", *input.Name)
		require.NotNil(t, input.Description)
		assert.Equal(t, "New description", *input.Description)
		assert.Nil(t, input.GitRepository)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Application{UUID: "app-upd-1", Name: "Updated App", Description: "New description"})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	name := "Updated App"
	desc := "New description"
	app, err := c.UpdateApplication(context.Background(), "app-upd-1", UpdateApplicationInput{
		Name:        &name,
		Description: &desc,
	})
	require.NoError(t, err)
	assert.Equal(t, "app-upd-1", app.UUID)
	assert.Equal(t, "Updated App", app.Name)
	assert.Equal(t, "New description", app.Description)
}

// TestUpdateApplicationInput_ClearDomainsJSON documents that *string domains
// with value "" is encoded (omitempty only skips nil), so PATCH can clear FQDN.
func TestUpdateApplicationInput_ClearDomainsJSON(t *testing.T) {
	t.Parallel()
	empty := ""
	name := "app"

	// Unset domains: key omitted
	b, err := json.Marshal(UpdateApplicationInput{Name: &name})
	require.NoError(t, err)
	assert.JSONEq(t, `{"name":"app"}`, string(b))

	// Clear FQDN: domains must be present as ""
	b, err = json.Marshal(UpdateApplicationInput{Name: &name, Domains: &empty})
	require.NoError(t, err)
	assert.JSONEq(t, `{"name":"app","domains":""}`, string(b))
}

func TestClient_UpdateApplication_ClearDomains(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var raw map[string]any
		require.NoError(t, json.Unmarshal(body, &raw))
		v, ok := raw["domains"]
		require.True(t, ok, "domains key must be present to clear FQDN")
		assert.Equal(t, "", v)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Application{UUID: "app-clear-dom", Name: "cleared", Domains: ""})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	empty := ""
	app, err := c.UpdateApplication(context.Background(), "app-clear-dom", UpdateApplicationInput{
		Domains: &empty,
	})
	require.NoError(t, err)
	assert.Equal(t, "app-clear-dom", app.UUID)
	assert.Equal(t, "", app.Domains)
}

func TestClient_DeleteApplication(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "/api/v1/applications/app-del-1", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	err := c.DeleteApplication(context.Background(), "app-del-1")
	require.NoError(t, err)
}

// --- Databases (remaining) ---

func TestClient_ListDatabases(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/databases", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]Database{
			{UUID: "db-1", Name: "postgres-main", Type: "postgresql"},
			{UUID: "db-2", Name: "redis-cache", Type: "redis"},
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	dbs, err := c.ListDatabases(context.Background())
	require.NoError(t, err)
	require.Len(t, dbs, 2)
	assert.Equal(t, "db-1", dbs[0].UUID)
	assert.Equal(t, "postgres-main", dbs[0].Name)
	assert.Equal(t, "postgresql", dbs[0].Type)
	assert.Equal(t, "db-2", dbs[1].UUID)
	assert.Equal(t, "redis-cache", dbs[1].Name)
	assert.Equal(t, "redis", dbs[1].Type)
}

func TestClient_CreateDatabase_Mariadb(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/databases/mariadb", r.URL.Path)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var input CreateMariadbInput
		require.NoError(t, json.Unmarshal(body, &input))
		assert.Equal(t, "srv-1", input.ServerUUID)
		assert.Equal(t, "proj-1", input.ProjectUUID)
		assert.Equal(t, "production", input.EnvironmentName)
		assert.Equal(t, "mdbuser", input.MariadbUser)
		assert.Equal(t, "mdbname", input.MariadbDatabase)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(Database{UUID: "db-mariadb-new"})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	db, err := c.CreateDatabase(context.Background(), "mariadb", CreateMariadbInput{
		CreateDatabaseBaseInput: CreateDatabaseBaseInput{
			ServerUUID:      "srv-1",
			ProjectUUID:     "proj-1",
			EnvironmentName: "production",
		},
		MariadbUser:     "mdbuser",
		MariadbDatabase: "mdbname",
	})
	require.NoError(t, err)
	assert.Equal(t, "db-mariadb-new", db.UUID)
}

func TestClient_CreateDatabase_Redis(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/databases/redis", r.URL.Path)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var input CreateRedisInput
		require.NoError(t, json.Unmarshal(body, &input))
		assert.Equal(t, "srv-1", input.ServerUUID)
		assert.Equal(t, "proj-1", input.ProjectUUID)
		assert.Equal(t, "production", input.EnvironmentName)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(Database{UUID: "db-redis-new"})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	db, err := c.CreateDatabase(context.Background(), "redis", CreateRedisInput{
		CreateDatabaseBaseInput: CreateDatabaseBaseInput{
			ServerUUID:      "srv-1",
			ProjectUUID:     "proj-1",
			EnvironmentName: "production",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "db-redis-new", db.UUID)
}

func TestClient_CreateDatabase_Mongodb(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/databases/mongodb", r.URL.Path)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var input CreateMongodbInput
		require.NoError(t, json.Unmarshal(body, &input))
		assert.Equal(t, "srv-1", input.ServerUUID)
		assert.Equal(t, "proj-1", input.ProjectUUID)
		assert.Equal(t, "production", input.EnvironmentName)
		assert.Equal(t, "mongoroot", input.MongoInitdbRootUsername)
		assert.Equal(t, "appdb", input.MongoInitdbDatabase)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(Database{UUID: "db-mongo-new"})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	db, err := c.CreateDatabase(context.Background(), "mongodb", CreateMongodbInput{
		CreateDatabaseBaseInput: CreateDatabaseBaseInput{
			ServerUUID:      "srv-1",
			ProjectUUID:     "proj-1",
			EnvironmentName: "production",
		},
		MongoInitdbRootUsername: "mongoroot",
		MongoInitdbDatabase:     "appdb",
	})
	require.NoError(t, err)
	assert.Equal(t, "db-mongo-new", db.UUID)
}

func TestClient_GetDeployment(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/deployments/dep-get-1", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Deployment{
			UUID:       "dep-get-1",
			ID:         200,
			Status:     "finished",
			ServerUUID: "srv-1",
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	dep, err := c.GetDeployment(context.Background(), "dep-get-1")
	require.NoError(t, err)
	assert.Equal(t, "dep-get-1", dep.UUID)
	assert.Equal(t, 200, dep.ID)
	assert.Equal(t, "finished", dep.Status)
	assert.Equal(t, "srv-1", dep.ServerUUID)
}

func TestClient_UpdateEnvVar_Application(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPatch, r.Method)
		assert.Equal(t, "/api/v1/applications/app-env-1/envs", r.URL.Path)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var bodyMap map[string]any
		require.NoError(t, json.Unmarshal(body, &bodyMap))
		assert.Equal(t, "DATABASE_URL", bodyMap["key"])
		assert.Equal(t, "postgres://new-host/db", bodyMap["value"])
		assert.Equal(t, false, bodyMap["is_buildtime"])

		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	isBuild := false
	err := c.UpdateEnvVar(context.Background(), "applications", "app-env-1", EnvironmentVariable{
		Key:     "DATABASE_URL",
		Value:   "postgres://new-host/db",
		IsBuild: false,
	}, &EnvVarWriteOpts{IsBuild: &isBuild})
	require.NoError(t, err)
}

func TestClient_CreateDockerImageApplication(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/applications/dockerimage", r.URL.Path)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var input CreateDockerImageAppInput
		require.NoError(t, json.Unmarshal(body, &input))
		assert.Equal(t, "proj-1", input.ProjectUUID)
		assert.Equal(t, "srv-1", input.ServerUUID)
		assert.Equal(t, "production", input.EnvironmentName)
		assert.Equal(t, "nginx:latest", input.DockerImage)
		assert.Equal(t, "80", input.PortsExposes)
		assert.Equal(t, "my-nginx", input.Name)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(Application{UUID: "docker-app-new", Name: "my-nginx"})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	app, err := c.CreateDockerImageApplication(context.Background(), CreateDockerImageAppInput{
		ProjectUUID:     "proj-1",
		ServerUUID:      "srv-1",
		EnvironmentName: "production",
		DockerImage:     "nginx:latest",
		PortsExposes:    "80",
		Name:            "my-nginx",
	})
	require.NoError(t, err)
	assert.Equal(t, "docker-app-new", app.UUID)
	assert.Equal(t, "my-nginx", app.Name)
}

// --- Service Environment Variables ---

func TestClient_CreateEnvVar_Service(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/services/svc-1/envs", r.URL.Path)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var bodyMap map[string]any
		require.NoError(t, json.Unmarshal(body, &bodyMap))
		assert.Equal(t, "REDIS_URL", bodyMap["key"])
		assert.Equal(t, "redis://localhost:6379", bodyMap["value"])
		assert.NotContains(t, bodyMap, "is_buildtime")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(CreateEnvVarResponse{UUID: "svc-env-new"})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	resp, err := c.CreateEnvVar(context.Background(), "services", "svc-1", EnvironmentVariable{
		Key:     "REDIS_URL",
		Value:   "redis://localhost:6379",
		IsBuild: true,
	}, nil)
	require.NoError(t, err)
	assert.Equal(t, "svc-env-new", resp.UUID)
}

func TestClient_ListEnvVars_Service(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/services/svc-1/envs", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]EnvironmentVariable{
			{UUID: "sev-1", Key: "PORT", Value: "5000", IsPreview: false, IsBuild: false},
			{UUID: "sev-2", Key: "SECRET", Value: "xyz", IsPreview: true, IsBuild: true},
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	vars, err := c.ListEnvVars(context.Background(), "services", "svc-1")
	require.NoError(t, err)
	require.Len(t, vars, 2)
	assert.Equal(t, "PORT", vars[0].Key)
	assert.Equal(t, "5000", vars[0].Value)
	assert.False(t, vars[0].IsPreview)
	assert.Equal(t, "SECRET", vars[1].Key)
	assert.True(t, vars[1].IsPreview)
	assert.True(t, vars[1].IsBuild)
}

func TestClient_UpdateEnvVar_Service(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPatch, r.Method)
		assert.Equal(t, "/api/v1/services/svc-env-1/envs", r.URL.Path)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var bodyMap map[string]any
		require.NoError(t, json.Unmarshal(body, &bodyMap))
		assert.Equal(t, "REDIS_URL", bodyMap["key"])
		assert.Equal(t, "redis://new-host:6379", bodyMap["value"])
		assert.NotContains(t, bodyMap, "is_buildtime")

		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	err := c.UpdateEnvVar(context.Background(), "services", "svc-env-1", EnvironmentVariable{
		Key:   "REDIS_URL",
		Value: "redis://new-host:6379",
	}, nil)
	require.NoError(t, err)
}

func TestClient_DeleteEnvVar_Service(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "/api/v1/services/svc-1/envs/sev-del", r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	err := c.DeleteEnvVar(context.Background(), "services", "svc-1", "sev-del")
	require.NoError(t, err)
}

// --- Private Keys (remaining) ---

func TestClient_UpdatePrivateKey(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPatch, r.Method)
		assert.Equal(t, "/api/v1/security/keys/pk-upd", r.URL.Path)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var input UpdatePrivateKeyInput
		require.NoError(t, json.Unmarshal(body, &input))
		require.NotNil(t, input.Name)
		assert.Equal(t, "renamed-key", *input.Name)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(PrivateKey{UUID: "pk-upd", Name: "renamed-key"})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	name := "renamed-key"
	k, err := c.UpdatePrivateKey(context.Background(), "pk-upd", UpdatePrivateKeyInput{Name: &name})
	require.NoError(t, err)
	assert.Equal(t, "pk-upd", k.UUID)
	assert.Equal(t, "renamed-key", k.Name)
}

// --- Projects (remaining) ---

func TestClient_UpdateProject(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPatch, r.Method)
		assert.Equal(t, "/api/v1/projects/proj-upd", r.URL.Path)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var input UpdateProjectInput
		require.NoError(t, json.Unmarshal(body, &input))
		require.NotNil(t, input.Name)
		assert.Equal(t, "Renamed Project", *input.Name)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Project{UUID: "proj-upd", Name: "Renamed Project"})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	name := "Renamed Project"
	proj, err := c.UpdateProject(context.Background(), "proj-upd", UpdateProjectInput{Name: &name})
	require.NoError(t, err)
	assert.Equal(t, "proj-upd", proj.UUID)
	assert.Equal(t, "Renamed Project", proj.Name)
}

// --- Services (remaining) ---

func TestClient_ListServices(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/services", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]Service{
			{UUID: "svc-1", Name: "wordpress", Type: "wordpress", ServerUUID: "srv-1", ProjectUUID: "proj-1", EnvironmentName: "production"},
			{UUID: "svc-2", Name: "plausible", Type: "plausible", ServerUUID: "srv-2", ProjectUUID: "proj-2", EnvironmentName: "staging"},
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	svcs, err := c.ListServices(context.Background())
	require.NoError(t, err)
	require.Len(t, svcs, 2)
	assert.Equal(t, "svc-1", svcs[0].UUID)
	assert.Equal(t, "wordpress", svcs[0].Name)
	assert.Equal(t, "wordpress", svcs[0].Type)
	assert.Equal(t, "srv-1", svcs[0].ServerUUID)
	assert.Equal(t, "svc-2", svcs[1].UUID)
	assert.Equal(t, "plausible", svcs[1].Name)
	assert.Equal(t, "staging", svcs[1].EnvironmentName)
}

func TestClient_StartService(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/services/svc-start/start", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	err := c.StartService(context.Background(), "svc-start")
	require.NoError(t, err)
}

func TestClient_StopService(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/services/svc-stop/stop", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	err := c.StopService(context.Background(), "svc-stop")
	require.NoError(t, err)
}

func TestClient_RestartService(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/services/svc-restart/restart", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	err := c.RestartService(context.Background(), "svc-restart")
	require.NoError(t, err)
}

// --- Private Git Application ---

func TestClient_CreatePrivateGitApplication(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/applications/private-deploy-key", r.URL.Path)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var input CreatePrivateGitAppInput
		require.NoError(t, json.Unmarshal(body, &input))
		assert.Equal(t, "proj-1", input.ProjectUUID)
		assert.Equal(t, "srv-1", input.ServerUUID)
		assert.Equal(t, "production", input.EnvironmentName)
		assert.Equal(t, "git@github.com:org/repo.git", input.GitRepository)
		assert.Equal(t, "main", input.GitBranch)
		assert.Equal(t, "dockerfile", input.BuildPack)
		assert.Equal(t, "8080", input.PortsExposes)
		assert.Equal(t, "pk-deploy", input.PrivateKeyUUID)
		assert.Equal(t, "my-api", input.Name)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(Application{UUID: "pgit-app-new", Name: "my-api"})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	app, err := c.CreatePrivateGitApplication(context.Background(), CreatePrivateGitAppInput{
		ProjectUUID:     "proj-1",
		ServerUUID:      "srv-1",
		EnvironmentName: "production",
		GitRepository:   "git@github.com:org/repo.git",
		GitBranch:       "main",
		BuildPack:       "dockerfile",
		PortsExposes:    "8080",
		PrivateKeyUUID:  "pk-deploy",
		Name:            "my-api",
	})
	require.NoError(t, err)
	assert.Equal(t, "pgit-app-new", app.UUID)
	assert.Equal(t, "my-api", app.Name)
}

// --- Database Backups ---

func TestClient_ListDatabaseBackups(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/databases/db-uuid-1/backups", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		retain := int64(7)
		json.NewEncoder(w).Encode([]DatabaseBackup{
			{ID: 1, UUID: "bk-1", DatabaseUUID: "db-uuid-1", Frequency: "0 * * * *", Enabled: true, RetainAmountLocally: &retain},
			{ID: 2, UUID: "bk-2", DatabaseUUID: "db-uuid-1", Frequency: "0 0 * * *", Enabled: false},
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	backups, err := c.ListDatabaseBackups(context.Background(), "db-uuid-1")
	require.NoError(t, err)
	require.Len(t, backups, 2)
	assert.Equal(t, 1, backups[0].ID)
	assert.Equal(t, "bk-1", backups[0].UUID)
	assert.Equal(t, "db-uuid-1", backups[0].DatabaseUUID)
	assert.Equal(t, "0 * * * *", backups[0].Frequency)
	assert.True(t, backups[0].Enabled)
	require.NotNil(t, backups[0].RetainAmountLocally)
	assert.Equal(t, int64(7), *backups[0].RetainAmountLocally)
	assert.Equal(t, 2, backups[1].ID)
	assert.Equal(t, "bk-2", backups[1].UUID)
	assert.False(t, backups[1].Enabled)
}

func TestClient_CreateDatabaseBackup(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/databases/db-uuid-1/backups", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var input CreateDatabaseBackupInput
		require.NoError(t, json.Unmarshal(body, &input))
		assert.Equal(t, "0 * * * *", input.Frequency)
		assert.True(t, input.Enabled)
		assert.Equal(t, "s3-storage-1", input.S3StorageID)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(DatabaseBackup{ID: 99, UUID: "bk-new", DatabaseUUID: "db-uuid-1", Frequency: "0 * * * *", Enabled: true})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	b, err := c.CreateDatabaseBackup(context.Background(), "db-uuid-1", CreateDatabaseBackupInput{
		Frequency:   "0 * * * *",
		Enabled:     true,
		S3StorageID: "s3-storage-1",
	})
	require.NoError(t, err)
	assert.Equal(t, 99, b.ID)
	assert.Equal(t, "bk-new", b.UUID)
	assert.Equal(t, "db-uuid-1", b.DatabaseUUID)
	assert.Equal(t, "0 * * * *", b.Frequency)
	assert.True(t, b.Enabled)
}

func TestClient_UpdateDatabaseBackup(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPatch, r.Method)
		assert.Equal(t, "/api/v1/databases/db-uuid-1/backups/backup-uuid-10", r.URL.Path)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var input UpdateDatabaseBackupInput
		require.NoError(t, json.Unmarshal(body, &input))
		require.NotNil(t, input.Frequency)
		assert.Equal(t, "0 0 * * *", *input.Frequency)
		require.NotNil(t, input.Enabled)
		assert.False(t, *input.Enabled)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(DatabaseBackup{ID: 10, UUID: "bk-10", DatabaseUUID: "db-uuid-1", Frequency: "0 0 * * *", Enabled: false})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	freq := "0 0 * * *"
	enabled := false
	b, err := c.UpdateDatabaseBackup(context.Background(), "db-uuid-1", "backup-uuid-10", UpdateDatabaseBackupInput{
		Frequency: &freq,
		Enabled:   &enabled,
	})
	require.NoError(t, err)
	assert.Equal(t, 10, b.ID)
	assert.Equal(t, "bk-10", b.UUID)
	assert.Equal(t, "0 0 * * *", b.Frequency)
	assert.False(t, b.Enabled)
}

func TestClient_DeleteDatabaseBackup(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "/api/v1/databases/db-uuid-1/backups/backup-uuid-5", r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	err := c.DeleteDatabaseBackup(context.Background(), "db-uuid-1", "backup-uuid-5")
	require.NoError(t, err)
}

func TestClient_ValidateServer(t *testing.T) {
	t.Parallel()
	expected := ServerValidation{Valid: true, Message: "Server is reachable"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/servers/srv-1/validate", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expected)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	got, err := c.ValidateServer(context.Background(), "srv-1")
	require.NoError(t, err)
	assert.True(t, got.Valid)
	assert.Equal(t, "Server is reachable", got.Message)
}

func TestClient_ListServerResources(t *testing.T) {
	t.Parallel()
	expected := []ServerResource{
		{UUID: "app-1", Name: "my-app", Type: "application"},
		{UUID: "db-1", Name: "my-db", Type: "database"},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/servers/srv-1/resources", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expected)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	got, err := c.ListServerResources(context.Background(), "srv-1")
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "app-1", got[0].UUID)
	assert.Equal(t, "application", got[0].Type)
	assert.Equal(t, "db-1", got[1].UUID)
	assert.Equal(t, "database", got[1].Type)
}

func TestClient_ValidateResourceOnServer(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]ServerResource{
			{UUID: "app-1", Name: "my-app", Type: "application"},
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	require.NoError(t, c.ValidateResourceOnServer(context.Background(), "srv-1", "app-1", "application"))

	err := c.ValidateResourceOnServer(context.Background(), "srv-1", "missing", "application")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is not deployed on server")

	err = (*Client)(nil).ValidateResourceOnServer(context.Background(), "srv-1", "app-1", "application")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "provider client is not configured")
}

func TestClient_ListServerDomains(t *testing.T) {
	t.Parallel()
	expected := []ServerDomain{
		{Domain: "app.example.com", IP: "10.0.0.1"},
		{Domain: "api.example.com", IP: "10.0.0.2"},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/servers/srv-1/domains", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expected)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	got, err := c.ListServerDomains(context.Background(), "srv-1")
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "app.example.com", got[0].Domain)
	assert.Equal(t, "10.0.0.1", got[0].IP)
	assert.Equal(t, "api.example.com", got[1].Domain)
}

func TestClient_CreateDatabase_Keydb(t *testing.T) {
	t.Parallel()
	expected := Database{UUID: "keydb-1", Name: "cache", Type: "keydb"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/databases/keydb", r.URL.Path)

		body, _ := io.ReadAll(r.Body)
		var input CreateKeydbInput
		require.NoError(t, json.Unmarshal(body, &input))
		assert.Equal(t, "proj-1", input.ProjectUUID)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(expected)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	got, err := c.CreateDatabase(context.Background(), "keydb", CreateKeydbInput{
		CreateDatabaseBaseInput: CreateDatabaseBaseInput{
			ServerUUID:      "srv-1",
			ProjectUUID:     "proj-1",
			EnvironmentName: "production",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "keydb-1", got.UUID)
	assert.Equal(t, "keydb", got.Type)
}

func TestClient_CreateDatabase_Dragonfly(t *testing.T) {
	t.Parallel()
	expected := Database{UUID: "df-1", Name: "sessions", Type: "dragonfly"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/databases/dragonfly", r.URL.Path)

		body, _ := io.ReadAll(r.Body)
		var input CreateDragonflyInput
		require.NoError(t, json.Unmarshal(body, &input))
		assert.Equal(t, "proj-1", input.ProjectUUID)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(expected)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	got, err := c.CreateDatabase(context.Background(), "dragonfly", CreateDragonflyInput{
		CreateDatabaseBaseInput: CreateDatabaseBaseInput{
			ServerUUID:      "srv-1",
			ProjectUUID:     "proj-1",
			EnvironmentName: "production",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "df-1", got.UUID)
	assert.Equal(t, "dragonfly", got.Type)
}

func TestClient_CreateDatabase_Clickhouse(t *testing.T) {
	t.Parallel()
	expected := Database{UUID: "ch-1", Name: "analytics", Type: "clickhouse"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/databases/clickhouse", r.URL.Path)

		body, _ := io.ReadAll(r.Body)
		var input CreateClickhouseInput
		require.NoError(t, json.Unmarshal(body, &input))
		assert.Equal(t, "proj-1", input.ProjectUUID)
		assert.Equal(t, "srv-1", input.ServerUUID)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(expected)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	got, err := c.CreateDatabase(context.Background(), "clickhouse", CreateClickhouseInput{
		CreateDatabaseBaseInput: CreateDatabaseBaseInput{
			ServerUUID:      "srv-1",
			ProjectUUID:     "proj-1",
			EnvironmentName: "production",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "ch-1", got.UUID)
	assert.Equal(t, "clickhouse", got.Type)
}

func TestClient_GetVersion(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/version", r.URL.Path)
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`v4.0.0-beta.362`))
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	v, err := c.GetVersion(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "v4.0.0-beta.362", v)
}

func TestClient_StartApplication(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/applications/app-1/start", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	err := c.StartApplication(context.Background(), "app-1")
	require.NoError(t, err)
}

func TestClient_StopApplication(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/applications/app-1/stop", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	err := c.StopApplication(context.Background(), "app-1")
	require.NoError(t, err)
}

// --- extractAPIMessage ---

func TestExtractAPIMessage(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input []byte
		want  string
	}{
		{
			name:  "json with message field",
			input: []byte(`{"message":"server overloaded"}`),
			want:  "server overloaded",
		},
		{
			name:  "json without message field",
			input: []byte(`{"error":"not found","code":404}`),
			want:  "API error response omitted",
		},
		{
			name:  "non-json body",
			input: []byte("plain text error"),
			want:  "API error response omitted",
		},
		{
			name:  "empty body",
			input: []byte(""),
			want:  "API error response omitted",
		},
		{
			name:  "json with empty message omits body",
			input: []byte(`{"message":""}`),
			want:  "API error response omitted",
		},
		{
			name:  "long body is omitted not dumped",
			input: []byte(strings.Repeat("x", 300)),
			want:  "API error response omitted",
		},
		{
			name:  "json with message and errors",
			input: []byte(`{"message":"Validation failed","errors":{"name":["required"]}}`),
			want:  `Validation failed name: ["required"]`,
		},
		{
			name:  "errors password field is redacted",
			input: []byte(`{"message":"Validation failed","errors":{"password":["hunter2"]}}`),
			want:  `Validation failed password: [REDACTED]`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := extractAPIMessage(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExtractAPIMessage_DoesNotDumpRawBody(t *testing.T) {
	t.Parallel()
	html := []byte(`<html><body>token=super-secret-token</body></html>`)
	got := extractAPIMessage(html)
	assert.NotContains(t, got, "super-secret-token")
	assert.NotContains(t, got, "<html>")
	assert.NotContains(t, got, "[raw API response]")

	rawJSON := []byte(`{"error":"not found","token":"leak-me"}`)
	gotJSON := extractAPIMessage(rawJSON)
	assert.NotContains(t, gotJSON, "leak-me")
	assert.NotContains(t, gotJSON, "[raw API response]")
}

// --- doText error paths ---

func TestClient_GetVersion_Non2xx(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("forbidden"))
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	_, err := c.GetVersion(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "403")
	assert.Contains(t, err.Error(), "API error response omitted")
	assert.NotContains(t, err.Error(), "forbidden")
}

func TestClient_GetVersion_JSONQuoted(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`"v4.1.0"`))
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	v, err := c.GetVersion(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "v4.1.0", v)
}

func TestClient_GetHealth(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/health", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	health, err := c.GetHealth(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "OK", health)
}

func TestClient_GetHealth_Non2xx(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"message":"access denied"}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	_, err := c.GetHealth(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "access denied")
}

// --- IsNotFound ---

func TestIsNotFound(t *testing.T) {
	t.Parallel()
	assert.True(t, IsNotFound(&NotFoundError{Message: "gone"}))
	assert.False(t, IsNotFound(io.EOF))
	assert.False(t, IsNotFound(nil))
}

// --- doWithStatus status mismatch ---

func TestClient_CreateProject_WrongStatusCode(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK) // 200, not the expected 201
		json.NewEncoder(w).Encode(Project{UUID: "proj-wrong"})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	_, err := c.CreateProject(context.Background(), CreateProjectInput{Name: "p"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected status 201")
	assert.Contains(t, err.Error(), "got 200")
	// Method and path help operators map failures to the Coolify endpoint.
	assert.Contains(t, err.Error(), "POST")
	assert.Contains(t, err.Error(), "/api/v1/projects")
}

// --- Malformed JSON response ---

func TestClient_GetProject_MalformedJSON(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<html>502 Bad Gateway</html>`))
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	_, err := c.GetProject(context.Background(), "proj-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decoding response")
}

// --- GitHub Apps client-side filtering ---

func TestClient_GetGitHubApp_NotFound(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]GitHubApp{
			{ID: 99, Name: "other-app"},
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	_, err := c.GetGitHubApp(context.Background(), 42)
	require.Error(t, err)
	assert.True(t, IsNotFound(err))
	assert.Contains(t, err.Error(), "42")
}

func TestClient_GetGitHubApp_Found(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]GitHubApp{
			{ID: 10, Name: "first-app"},
			{ID: 42, Name: "target-app"},
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	app, err := c.GetGitHubApp(context.Background(), 42)
	require.NoError(t, err)
	assert.Equal(t, int64(42), app.ID)
	assert.Equal(t, "target-app", app.Name)
}

func TestClient_GetGitHubApp_IgnoresMalformedUnmatchedApps(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]GitHubApp{
			{UUID: "gh-bad", Name: "missing-id"},
			{ID: 42, Name: "target-app"},
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	app, err := c.GetGitHubApp(context.Background(), 42)
	require.NoError(t, err)
	assert.Equal(t, int64(42), app.ID)
	assert.Equal(t, "target-app", app.Name)
}

func TestClient_GetGitHubApp_EmptyList(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]GitHubApp{})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	_, err := c.GetGitHubApp(context.Background(), 1)
	require.Error(t, err)
	assert.True(t, IsNotFound(err))
}

// --- GetGitHubAppByAppID ---

func TestClient_GetGitHubAppByAppID_Found(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]GitHubApp{
			{ID: 10, Name: "other-app", AppID: 111},
			{ID: 20, Name: "target-app", AppID: 222},
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	app, err := c.GetGitHubAppByAppID(context.Background(), 222)
	require.NoError(t, err)
	assert.Equal(t, int64(20), app.ID)
	assert.Equal(t, int64(222), app.AppID)
	assert.Equal(t, "target-app", app.Name)
}

func TestClient_GetGitHubAppByAppID_NotFound(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]GitHubApp{
			{ID: 10, Name: "other-app", AppID: 111},
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	_, err := c.GetGitHubAppByAppID(context.Background(), 999)
	require.Error(t, err)
	assert.True(t, IsNotFound(err))
	assert.Contains(t, err.Error(), "999")
}

func TestClient_GetGitHubAppByAppID_SkipsBuiltInRecord(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// The built-in "Public GitHub" system record has ID=0 and AppID=0.
		json.NewEncoder(w).Encode([]GitHubApp{
			{ID: 0, Name: "Public GitHub", AppID: 0},
			{ID: 5, Name: "real-app", AppID: 333},
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	// Searching for AppID 0 should not match the built-in record (ID=0 is skipped).
	_, err := c.GetGitHubAppByAppID(context.Background(), 0)
	require.Error(t, err)
	assert.True(t, IsNotFound(err))
}

func TestClient_GetGitHubAppByAppID_EmptyList(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]GitHubApp{})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	_, err := c.GetGitHubAppByAppID(context.Background(), 42)
	require.Error(t, err)
	assert.True(t, IsNotFound(err))
}

// --- PollUntilDeleted ---

func TestPollUntilDeleted_ImmediateNotFound(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result := PollUntilDeleted(ctx, func(context.Context) error {
		return &NotFoundError{Message: "gone"}
	})
	assert.True(t, result, "expected true when resource is immediately gone")
}

func TestPollUntilDeleted_CancelledContext(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately
	result := PollUntilDeleted(ctx, func(context.Context) error {
		return nil // resource still exists
	})
	assert.False(t, result, "expected false when context is cancelled")
}

func TestPollUntilDeleted_DeadlineRespected(t *testing.T) {
	t.Parallel()
	// Resource still exists on the immediate probe; the 200ms deadline then
	// expires during the first backoff sleep.
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	start := time.Now()
	result := PollUntilDeleted(ctx, func(context.Context) error {
		return nil // resource still exists
	})
	elapsed := time.Since(start)
	assert.False(t, result, "expected false when deadline expires before poll")
	assert.Less(t, elapsed, 1*time.Second, "should respect short deadline, not wait 2 minutes")
}

func TestPollUntilDeleted_ProbesImmediately(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	start := time.Now()
	var firstProbe time.Duration
	result := PollUntilDeleted(ctx, func(context.Context) error {
		if firstProbe == 0 {
			firstProbe = time.Since(start)
		}
		return &NotFoundError{Message: "gone"}
	})
	assert.True(t, result, "expected true when resource is immediately gone")
	assert.Less(t, firstProbe, 100*time.Millisecond, "first GET should run immediately, not after 500ms sleep")
}

func TestPollUntilDeleted_GetFnReceivesDeadline(t *testing.T) {
	t.Parallel()
	var sawDeadline bool
	result := PollUntilDeleted(context.Background(), func(pollCtx context.Context) error {
		_, sawDeadline = pollCtx.Deadline()
		return &NotFoundError{Message: "gone"}
	})
	assert.True(t, result, "expected true when resource is immediately gone")
	assert.True(t, sawDeadline, "getFn must receive pollCtx with the 2-minute fallback deadline")
}

// --- validateParentType ---

func TestValidateParentType_Valid(t *testing.T) {
	t.Parallel()
	for _, pt := range []string{"applications", "services", "databases"} {
		require.NoError(t, validateParentType(pt), "expected %q to be valid", pt)
	}
}

func TestValidateParentType_Invalid(t *testing.T) {
	t.Parallel()
	for _, pt := range []string{"invalid", "", "APPLICATION", "deployments"} {
		err := validateParentType(pt)
		require.Error(t, err, "expected %q to be invalid", pt)
		assert.Contains(t, err.Error(), pt)
	}
}

// --- Bulk Environment Variables ---

func TestClient_BulkUpdateEnvVars_Application(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPatch, r.Method)
		assert.Equal(t, "/api/v1/applications/app-1/envs/bulk", r.URL.Path)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var input BulkEnvVarInput
		require.NoError(t, json.Unmarshal(body, &input))
		require.Len(t, input.Variables, 2)
		assert.Equal(t, "KEY1", input.Variables[0].Key)
		assert.Equal(t, "val1", input.Variables[0].Value)

		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	err := c.BulkUpdateEnvVars(context.Background(), "applications", "app-1", BulkEnvVarInput{
		Variables: []EnvVarEntry{
			{Key: "KEY1", Value: "val1"},
			{Key: "KEY2", Value: "val2"},
		},
	})
	require.NoError(t, err)
}

func TestClient_BulkUpdateEnvVars_Database(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPatch, r.Method)
		assert.Equal(t, "/api/v1/databases/db-1/envs/bulk", r.URL.Path)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var input BulkEnvVarInput
		require.NoError(t, json.Unmarshal(body, &input))
		require.Len(t, input.Variables, 1)
		assert.Equal(t, "DB_KEY", input.Variables[0].Key)

		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	err := c.BulkUpdateEnvVars(context.Background(), "databases", "db-1", BulkEnvVarInput{
		Variables: []EnvVarEntry{{Key: "DB_KEY", Value: "db_val"}},
	})
	require.NoError(t, err)
}

func TestClient_BulkUpdateEnvVars_Service(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPatch, r.Method)
		assert.Equal(t, "/api/v1/services/svc-1/envs/bulk", r.URL.Path)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var input BulkEnvVarInput
		require.NoError(t, json.Unmarshal(body, &input))
		require.Len(t, input.Variables, 1)
		assert.Equal(t, "SVC_KEY", input.Variables[0].Key)

		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	err := c.BulkUpdateEnvVars(context.Background(), "services", "svc-1", BulkEnvVarInput{
		Variables: []EnvVarEntry{{Key: "SVC_KEY", Value: "svc_val"}},
	})
	require.NoError(t, err)
}

func TestClient_PostNotRetriedOn5xx(t *testing.T) {
	t.Parallel()
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	_, err := c.CreateProject(context.Background(), CreateProjectInput{
		Name: "should-not-retry",
	})
	require.Error(t, err)
	assert.Equal(t, int32(1), atomic.LoadInt32(&attempts),
		"POST should not be retried on 5xx; expected exactly 1 attempt")
}

func strPtr(s string) *string { return &s }

func TestClient_PatchRetriedOn5xx(t *testing.T) {
	t.Parallel()
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token", RetryConfig{
		Attempts: 2,
		MinWait:  time.Millisecond,
		MaxWait:  time.Millisecond,
	})
	_, err := c.UpdateProject(context.Background(), "test-uuid", UpdateProjectInput{
		Name: strPtr("should-retry"),
	})
	require.Error(t, err)
	got := atomic.LoadInt32(&attempts)
	assert.True(t, got > 1,
		"PATCH should be retried on 5xx; expected >1 attempt, got %d", got)
}

func TestClient_DeleteNotRetriedOn5xx(t *testing.T) {
	t.Parallel()
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	err := c.DeleteProject(context.Background(), "test-uuid")
	require.Error(t, err)
	assert.Equal(t, int32(1), atomic.LoadInt32(&attempts),
		"DELETE should not be retried on 5xx; expected exactly 1 attempt")
}

func TestClient_400NotRetried(t *testing.T) {
	t.Parallel()
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"message":"bad request"}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	_, err := c.GetProject(context.Background(), "test-uuid")
	require.Error(t, err)
	assert.Equal(t, int32(1), atomic.LoadInt32(&attempts),
		"GET with 400 should not be retried; expected exactly 1 attempt")
}

func TestClient_422NotRetried(t *testing.T) {
	t.Parallel()
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(`{"message":"validation failed"}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	_, err := c.UpdateProject(context.Background(), "test-uuid", UpdateProjectInput{
		Name: strPtr("bad-update"),
	})
	require.Error(t, err)
	assert.Equal(t, int32(1), atomic.LoadInt32(&attempts),
		"PATCH with 422 should not be retried; expected exactly 1 attempt")
}

func TestDeployment_DeploymentUUID_JSONTag(t *testing.T) {
	t.Parallel()
	d := Deployment{UUID: "test-uuid-123", Status: "finished"}
	data, err := json.Marshal(d)
	require.NoError(t, err)

	var raw map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &raw))
	_, hasDeploymentUUID := raw["deployment_uuid"]
	assert.True(t, hasDeploymentUUID, "expected JSON key 'deployment_uuid'")
	_, hasUUID := raw["uuid"]
	assert.False(t, hasUUID, "unexpected JSON key 'uuid'")

	apiJSON := []byte(`{"deployment_uuid":"round-trip-uuid","status":"queued"}`)
	var parsed Deployment
	require.NoError(t, json.Unmarshal(apiJSON, &parsed))
	assert.Equal(t, "round-trip-uuid", parsed.UUID)
}

func TestClient_CreateDockerfileApplication(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/applications/dockerfile", r.URL.Path)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var input CreateDockerfileAppInput
		require.NoError(t, json.Unmarshal(body, &input))
		assert.Equal(t, "proj-1", input.ProjectUUID)
		assert.Equal(t, "/Dockerfile", input.DockerfileLocation)

		// Guard: API expects "dockerfile" key, NOT "dockerfile_location"
		var raw map[string]interface{}
		require.NoError(t, json.Unmarshal(body, &raw))
		_, hasDockerfile := raw["dockerfile"]
		assert.True(t, hasDockerfile, "expected JSON key 'dockerfile'")
		_, hasDockerfileLocation := raw["dockerfile_location"]
		assert.False(t, hasDockerfileLocation, "unexpected JSON key 'dockerfile_location'")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(Application{UUID: "dockerfile-app-new"})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	app, err := c.CreateDockerfileApplication(context.Background(), CreateDockerfileAppInput{
		ProjectUUID:        "proj-1",
		ServerUUID:         "srv-1",
		EnvironmentName:    "production",
		DockerfileLocation: "/Dockerfile",
		PortsExposes:       "80",
	})
	require.NoError(t, err)
	assert.Equal(t, "dockerfile-app-new", app.UUID)
}

func TestClient_CreateHetznerServer(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/servers/hetzner", r.URL.Path)
		var body map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "my-hetzner", body["name"])
		assert.Equal(t, "tok-uuid-1", body["cloud_provider_token_uuid"])
		assert.Equal(t, "cx11", body["server_type"])
		assert.Equal(t, "fsn1", body["location"])
		assert.Equal(t, "ubuntu-22.04", body["image"])
		assert.Equal(t, "pk-uuid-1", body["private_key_uuid"])
		assert.Equal(t, []interface{}{float64(38), float64(39)}, body["hetzner_firewall_ids"])
		assert.Equal(t, []interface{}{float64(456), float64(457)}, body["hetzner_network_ids"])
		_, hasHetznerToken := body["hetzner_token"]
		assert.False(t, hasHetznerToken, "should not send old hetzner_token field")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": "hz-srv-1"})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	server, err := c.CreateHetznerServer(context.Background(), CreateHetznerServerInput{
		Name:                   "my-hetzner",
		CloudProviderTokenUUID: "tok-uuid-1",
		ServerType:             "cx11",
		Location:               "fsn1",
		Image:                  "ubuntu-22.04",
		PrivateKeyUUID:         "pk-uuid-1",
		HetznerFirewallIDs:     []int64{38, 39},
		HetznerNetworkIDs:      []int64{456, 457},
	})
	require.NoError(t, err)
	assert.Equal(t, "hz-srv-1", server.UUID)
}

func TestGitHubApp_OrganizationName_JSONTag(t *testing.T) {
	t.Parallel()
	app := GitHubApp{OrganizationName: "my-org"}
	data, err := json.Marshal(app)
	require.NoError(t, err)

	var raw map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &raw))
	_, hasOrganization := raw["organization"]
	assert.True(t, hasOrganization, "expected JSON key 'organization'")
	_, hasOrganizationName := raw["organization_name"]
	assert.False(t, hasOrganizationName, "unexpected JSON key 'organization_name'")

	apiJSON := []byte(`{"id":1,"name":"test","organization":"round-trip-org"}`)
	var parsed GitHubApp
	require.NoError(t, json.Unmarshal(apiJSON, &parsed))
	assert.Equal(t, "round-trip-org", parsed.OrganizationName)
}

func TestGitHubAppCreate_PrivateKeyUUID_JSONTag(t *testing.T) {
	t.Parallel()
	input := CreateGitHubAppIntegrationInput{PrivateKeyUUID: "pk-uuid-123"}
	data, err := json.Marshal(input)
	require.NoError(t, err)

	var raw map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &raw))
	_, hasPrivateKeyUUID := raw["private_key_uuid"]
	assert.True(t, hasPrivateKeyUUID, "expected JSON key 'private_key_uuid'")
	_, hasPrivateKey := raw["private_key"]
	assert.False(t, hasPrivateKey, "should not send old 'private_key' field")
}

func TestHetznerCreate_CloudProviderTokenUUID_JSONTag(t *testing.T) {
	t.Parallel()
	input := CreateHetznerServerInput{CloudProviderTokenUUID: "tok-uuid-1"}
	data, err := json.Marshal(input)
	require.NoError(t, err)

	var raw map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &raw))
	_, hasToken := raw["cloud_provider_token_uuid"]
	assert.True(t, hasToken, "expected JSON key 'cloud_provider_token_uuid'")
	_, hasOldToken := raw["hetzner_token"]
	assert.False(t, hasOldToken, "should not send old 'hetzner_token' field")
}

func TestHetznerCreate_NetworkAndFirewallIDs_JSONTags(t *testing.T) {
	t.Parallel()
	input := CreateHetznerServerInput{
		HetznerSSHKeyIDs:   []int64{12345, 67890},
		HetznerFirewallIDs: []int64{38, 39},
		HetznerNetworkIDs:  []int64{456},
	}
	data, err := json.Marshal(input)
	require.NoError(t, err)

	var raw map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &raw))
	assert.Equal(t, []interface{}{float64(12345), float64(67890)}, raw["hetzner_ssh_key_ids"])
	assert.Equal(t, []interface{}{float64(38), float64(39)}, raw["hetzner_firewall_ids"])
	assert.Equal(t, []interface{}{float64(456)}, raw["hetzner_network_ids"])

	empty := CreateHetznerServerInput{}
	emptyData, err := json.Marshal(empty)
	require.NoError(t, err)
	var emptyRaw map[string]interface{}
	require.NoError(t, json.Unmarshal(emptyData, &emptyRaw))
	_, hasSSHKeys := emptyRaw["hetzner_ssh_key_ids"]
	assert.False(t, hasSSHKeys, "empty slice must omit hetzner_ssh_key_ids")
	_, hasFirewalls := emptyRaw["hetzner_firewall_ids"]
	assert.False(t, hasFirewalls, "empty slice must omit hetzner_firewall_ids")
	_, hasNetworks := emptyRaw["hetzner_network_ids"]
	assert.False(t, hasNetworks, "empty slice must omit hetzner_network_ids")
}

func TestHetznerCreate_EnableBackups_JSONTag(t *testing.T) {
	t.Parallel()
	enabled := true
	input := CreateHetznerServerInput{EnableBackups: &enabled}
	data, err := json.Marshal(input)
	require.NoError(t, err)

	var raw map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &raw))
	assert.Equal(t, true, raw["enable_backups"])

	empty := CreateHetznerServerInput{}
	emptyData, err := json.Marshal(empty)
	require.NoError(t, err)
	var emptyRaw map[string]interface{}
	require.NoError(t, json.Unmarshal(emptyData, &emptyRaw))
	_, hasBackups := emptyRaw["enable_backups"]
	assert.False(t, hasBackups, "nil pointer must omit enable_backups")
}

// --- Scheduled Tasks ---

func TestClient_ListScheduledTasks(t *testing.T) {
	t.Parallel()
	for _, parentType := range []string{"applications", "services"} {
		t.Run(parentType, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/api/v1/"+parentType+"/parent-uuid-1/scheduled-tasks", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode([]ScheduledTask{
					{UUID: "task-1", Name: "backup", Command: "pg_dump", Frequency: "0 * * * *", Enabled: true},
					{UUID: "task-2", Name: "cleanup", Command: "rm -rf /tmp/*", Frequency: "0 0 * * *", Enabled: false},
				})
			}))
			defer srv.Close()

			c := New(srv.URL, "test-token")
			tasks, err := c.ListScheduledTasks(context.Background(), parentType, "parent-uuid-1")
			require.NoError(t, err)
			require.Len(t, tasks, 2)
			assert.Equal(t, "task-1", tasks[0].UUID)
			assert.Equal(t, "backup", tasks[0].Name)
			assert.Equal(t, "pg_dump", tasks[0].Command)
			assert.Equal(t, "0 * * * *", tasks[0].Frequency)
			assert.True(t, tasks[0].Enabled)
			assert.Equal(t, "task-2", tasks[1].UUID)
			assert.Equal(t, "cleanup", tasks[1].Name)
			assert.False(t, tasks[1].Enabled)
		})
	}
}

func TestClient_ListScheduledTasks_InvalidParentType(t *testing.T) {
	t.Parallel()
	var called int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&called, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	_, err := c.ListScheduledTasks(context.Background(), "invalid", "uuid-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid")
	assert.Equal(t, int32(0), atomic.LoadInt32(&called), "should not make HTTP request for invalid parent type")
}

func TestClient_CreateScheduledTask(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/applications/app-uuid-1/scheduled-tasks", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var input CreateScheduledTaskInput
		require.NoError(t, json.Unmarshal(body, &input))
		assert.Equal(t, "daily-backup", input.Name)
		assert.Equal(t, "pg_dump mydb", input.Command)
		assert.Equal(t, "0 0 * * *", input.Frequency)
		assert.True(t, input.Enabled)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(createScheduledTaskResponse{UUID: "task-new-uuid"})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	uuid, err := c.CreateScheduledTask(context.Background(), "applications", "app-uuid-1", CreateScheduledTaskInput{
		Name:      "daily-backup",
		Command:   "pg_dump mydb",
		Frequency: "0 0 * * *",
		Enabled:   true,
	})
	require.NoError(t, err)
	assert.Equal(t, "task-new-uuid", uuid)
}

func TestClient_CreateScheduledTask_InvalidParentType(t *testing.T) {
	t.Parallel()
	var called int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&called, 1)
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	_, err := c.CreateScheduledTask(context.Background(), "deployments", "uuid-1", CreateScheduledTaskInput{
		Name: "test", Command: "echo hi", Frequency: "* * * * *",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "deployments")
	assert.Equal(t, int32(0), atomic.LoadInt32(&called))
}

func TestClient_UpdateScheduledTask(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPatch, r.Method)
		assert.Equal(t, "/api/v1/services/svc-uuid-1/scheduled-tasks/task-uuid-1", r.URL.Path)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var input UpdateScheduledTaskInput
		require.NoError(t, json.Unmarshal(body, &input))
		require.NotNil(t, input.Name)
		assert.Equal(t, "renamed-task", *input.Name)
		require.NotNil(t, input.Enabled)
		assert.False(t, *input.Enabled)
		assert.Nil(t, input.Command)
		assert.Nil(t, input.Frequency)

		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	name := "renamed-task"
	enabled := false
	err := c.UpdateScheduledTask(context.Background(), "services", "svc-uuid-1", "task-uuid-1", UpdateScheduledTaskInput{
		Name:    &name,
		Enabled: &enabled,
	})
	require.NoError(t, err)
}

func TestClient_UpdateScheduledTask_InvalidParentType(t *testing.T) {
	t.Parallel()
	var called int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&called, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	err := c.UpdateScheduledTask(context.Background(), "pods", "uuid-1", "task-1", UpdateScheduledTaskInput{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pods")
	assert.Equal(t, int32(0), atomic.LoadInt32(&called))
}

func TestClient_DeleteScheduledTask(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "/api/v1/databases/db-uuid-1/scheduled-tasks/task-del-1", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	err := c.DeleteScheduledTask(context.Background(), "databases", "db-uuid-1", "task-del-1")
	require.NoError(t, err)
}

func TestClient_DeleteScheduledTask_InvalidParentType(t *testing.T) {
	t.Parallel()
	var called int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&called, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	err := c.DeleteScheduledTask(context.Background(), "", "uuid-1", "task-1")
	require.Error(t, err)
	assert.Equal(t, int32(0), atomic.LoadInt32(&called))
}

func TestClient_ListTaskExecutions(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/applications/app-uuid-1/scheduled-tasks/task-uuid-1/executions", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]TaskExecution{
			{UUID: "exec-1", Status: "success", Message: "completed", CreatedAt: "2025-01-01T00:00:00Z"},
			{UUID: "exec-2", Status: "failed", Message: "timeout", CreatedAt: "2025-01-02T00:00:00Z"},
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	execs, err := c.ListTaskExecutions(context.Background(), "applications", "app-uuid-1", "task-uuid-1")
	require.NoError(t, err)
	require.Len(t, execs, 2)
	assert.Equal(t, "exec-1", execs[0].UUID)
	assert.Equal(t, "success", execs[0].Status)
	assert.Equal(t, "completed", execs[0].Message)
	assert.Equal(t, "2025-01-01T00:00:00Z", execs[0].CreatedAt)
	assert.Equal(t, "exec-2", execs[1].UUID)
	assert.Equal(t, "failed", execs[1].Status)
	assert.Equal(t, "timeout", execs[1].Message)
}

func TestClient_ListTaskExecutions_InvalidParentType(t *testing.T) {
	t.Parallel()
	var called int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&called, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	_, err := c.ListTaskExecutions(context.Background(), "containers", "uuid-1", "task-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "containers")
	assert.Equal(t, int32(0), atomic.LoadInt32(&called))
}

// --- Persistent Storages ---

func TestClient_ListStorages(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/applications/app-uuid-1/storages", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(storageListResponse{
			PersistentStorages: []Storage{
				{UUID: "ps-1", Name: "data-vol", MountPath: "/data", HostPath: "/mnt/data"},
				{UUID: "ps-2", Name: "logs-vol", MountPath: "/var/log"},
			},
			FileStorages: []Storage{
				{UUID: "fs-1", Name: "config-file", MountPath: "/etc/app/config.yml"},
			},
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	storages, err := c.ListStorages(context.Background(), "applications", "app-uuid-1")
	require.NoError(t, err)
	require.Len(t, storages, 3, "should merge persistent_storages and file_storages")
	assert.Equal(t, "ps-1", storages[0].UUID)
	assert.Equal(t, "data-vol", storages[0].Name)
	assert.Equal(t, "/data", storages[0].MountPath)
	assert.Equal(t, "/mnt/data", storages[0].HostPath)
	assert.Equal(t, "ps-2", storages[1].UUID)
	assert.Equal(t, "logs-vol", storages[1].Name)
	assert.Equal(t, "fs-1", storages[2].UUID)
	assert.Equal(t, "config-file", storages[2].Name)
	assert.Equal(t, "/etc/app/config.yml", storages[2].MountPath)
}

func TestClient_ListStorages_EmptyArrays(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(storageListResponse{
			PersistentStorages: []Storage{},
			FileStorages:       []Storage{},
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	storages, err := c.ListStorages(context.Background(), "services", "svc-uuid-1")
	require.NoError(t, err)
	assert.Empty(t, storages)
}

func TestClient_ListStorages_InvalidParentType(t *testing.T) {
	t.Parallel()
	var called int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&called, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	_, err := c.ListStorages(context.Background(), "invalid", "uuid-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid")
	assert.Equal(t, int32(0), atomic.LoadInt32(&called))
}

func TestClient_CreateStorage(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/databases/db-uuid-1/storages", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var input CreateStorageInput
		require.NoError(t, json.Unmarshal(body, &input))
		assert.Equal(t, "persistent", input.Type)
		assert.Equal(t, "my-volume", input.Name)
		assert.Equal(t, "/data", input.MountPath)
		assert.Equal(t, "/mnt/host-data", input.HostPath)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(CreateStorageResponse{UUID: "storage-new-uuid"})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	resp, err := c.CreateStorage(context.Background(), "databases", "db-uuid-1", CreateStorageInput{
		Type:      "persistent",
		Name:      "my-volume",
		MountPath: "/data",
		HostPath:  "/mnt/host-data",
	})
	require.NoError(t, err)
	assert.Equal(t, "storage-new-uuid", resp.UUID)
}

func TestClient_CreateStorage_InvalidParentType(t *testing.T) {
	t.Parallel()
	var called int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&called, 1)
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	_, err := c.CreateStorage(context.Background(), "volumes", "uuid-1", CreateStorageInput{
		Name: "test", MountPath: "/data",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "volumes")
	assert.Equal(t, int32(0), atomic.LoadInt32(&called))
}

func TestClient_UpdateStorage(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPatch, r.Method)
		assert.Equal(t, "/api/v1/services/svc-uuid-1/storages", r.URL.Path)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var input UpdateStorageInput
		require.NoError(t, json.Unmarshal(body, &input))
		require.NotNil(t, input.UUID)
		assert.Equal(t, "storage-uuid-1", *input.UUID)
		assert.Equal(t, "persistent", input.Type)
		require.NotNil(t, input.Name)
		assert.Equal(t, "renamed-vol", *input.Name)
		require.NotNil(t, input.MountPath)
		assert.Equal(t, "/new-path", *input.MountPath)

		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	storageUUID := "storage-uuid-1"
	name := "renamed-vol"
	mountPath := "/new-path"
	err := c.UpdateStorage(context.Background(), "services", "svc-uuid-1", UpdateStorageInput{
		UUID:      &storageUUID,
		Type:      "persistent",
		Name:      &name,
		MountPath: &mountPath,
	})
	require.NoError(t, err)
}

func TestClient_UpdateStorage_InvalidParentType(t *testing.T) {
	t.Parallel()
	var called int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&called, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	err := c.UpdateStorage(context.Background(), "pods", "uuid-1", UpdateStorageInput{Type: "persistent"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pods")
	assert.Equal(t, int32(0), atomic.LoadInt32(&called))
}

func TestClient_DeleteStorage(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "/api/v1/applications/app-uuid-1/storages/storage-del-1", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	err := c.DeleteStorage(context.Background(), "applications", "app-uuid-1", "storage-del-1")
	require.NoError(t, err)
}

func TestClient_DeleteStorage_InvalidParentType(t *testing.T) {
	t.Parallel()
	var called int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&called, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	err := c.DeleteStorage(context.Background(), "APPLICATIONS", "uuid-1", "s-1")
	require.Error(t, err)
	assert.Equal(t, int32(0), atomic.LoadInt32(&called))
}

// --- Environments ---

func TestClient_ListEnvironments(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/projects/proj-uuid-1/environments", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]Environment{
			{ID: 1, Name: "production", ProjectUUID: "proj-uuid-1", Description: "Prod env", CreatedAt: "2025-01-01T00:00:00Z", UpdatedAt: "2025-06-01T00:00:00Z"},
			{ID: 2, Name: "staging", ProjectUUID: "proj-uuid-1"},
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	envs, err := c.ListEnvironments(context.Background(), "proj-uuid-1")
	require.NoError(t, err)
	require.Len(t, envs, 2)
	assert.Equal(t, int64(1), envs[0].ID)
	assert.Equal(t, "production", envs[0].Name)
	assert.Equal(t, "proj-uuid-1", envs[0].ProjectUUID)
	assert.Equal(t, "Prod env", envs[0].Description)
	assert.Equal(t, "2025-01-01T00:00:00Z", envs[0].CreatedAt)
	assert.Equal(t, "2025-06-01T00:00:00Z", envs[0].UpdatedAt)
	assert.Equal(t, int64(2), envs[1].ID)
	assert.Equal(t, "staging", envs[1].Name)
}

func TestClient_GetEnvironment(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/projects/proj-uuid-1/production", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Environment{
			ID:          1,
			Name:        "production",
			ProjectUUID: "proj-uuid-1",
			Description: "Production environment",
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	env, err := c.GetEnvironment(context.Background(), "proj-uuid-1", "production")
	require.NoError(t, err)
	assert.Equal(t, int64(1), env.ID)
	assert.Equal(t, "production", env.Name)
	assert.Equal(t, "proj-uuid-1", env.ProjectUUID)
	assert.Equal(t, "Production environment", env.Description)
}

func TestClient_CreateEnvironment(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/projects/proj-uuid-1/environments", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var input CreateEnvironmentInput
		require.NoError(t, json.Unmarshal(body, &input))
		assert.Equal(t, "staging", input.Name)

		// Verify no description field is sent (API rejects it).
		var raw map[string]interface{}
		require.NoError(t, json.Unmarshal(body, &raw))
		_, hasDesc := raw["description"]
		assert.False(t, hasDesc, "CreateEnvironmentInput should not send 'description'")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(Environment{ID: 3, Name: "staging", ProjectUUID: "proj-uuid-1"})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	env, err := c.CreateEnvironment(context.Background(), "proj-uuid-1", CreateEnvironmentInput{Name: "staging"})
	require.NoError(t, err)
	assert.Equal(t, int64(3), env.ID)
	assert.Equal(t, "staging", env.Name)
	assert.Equal(t, "proj-uuid-1", env.ProjectUUID)
}

func TestClient_DeleteEnvironment(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "/api/v1/projects/proj-uuid-1/environments/staging", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	err := c.DeleteEnvironment(context.Background(), "proj-uuid-1", "staging")
	require.NoError(t, err)
}

// --- Cloud Tokens ---

func TestClient_ListCloudTokens(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/cloud-tokens", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]CloudToken{
			{UUID: "ct-1", Name: "hetzner-token", Provider: "hetzner", Token: "tok-1"},
			{UUID: "ct-2", Name: "aws-token", Provider: "aws", Token: "tok-2"},
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	tokens, err := c.ListCloudTokens(context.Background())
	require.NoError(t, err)
	require.Len(t, tokens, 2)
	assert.Equal(t, "ct-1", tokens[0].UUID)
	assert.Equal(t, "hetzner-token", tokens[0].Name)
	assert.Equal(t, "hetzner", tokens[0].Provider)
	assert.Equal(t, "tok-1", tokens[0].Token)
	assert.Equal(t, "ct-2", tokens[1].UUID)
	assert.Equal(t, "aws-token", tokens[1].Name)
	assert.Equal(t, "aws", tokens[1].Provider)
}

func TestClient_GetCloudToken(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/cloud-tokens/ct-uuid-1", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(CloudToken{
			UUID:     "ct-uuid-1",
			Name:     "my-cloud-token",
			Provider: "hetzner",
			Token:    "secret-token-value",
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	ct, err := c.GetCloudToken(context.Background(), "ct-uuid-1")
	require.NoError(t, err)
	assert.Equal(t, "ct-uuid-1", ct.UUID)
	assert.Equal(t, "my-cloud-token", ct.Name)
	assert.Equal(t, "hetzner", ct.Provider)
	assert.Equal(t, "secret-token-value", ct.Token)
}

func TestClient_GetCloudToken_NotFound(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	_, err := c.GetCloudToken(context.Background(), "nonexistent-uuid")
	require.Error(t, err)
	assert.True(t, IsNotFound(err))
}

func TestClient_GetCloudToken_URLEscape(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/cloud-tokens/uuid%2Fwith%2Fslashes", r.URL.EscapedPath())
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(CloudToken{UUID: "uuid/with/slashes", Name: "escaped"})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	ct, err := c.GetCloudToken(context.Background(), "uuid/with/slashes")
	require.NoError(t, err)
	assert.Equal(t, "escaped", ct.Name)
}

func TestClient_CreateCloudToken(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/cloud-tokens", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var input CreateCloudTokenInput
		require.NoError(t, json.Unmarshal(body, &input))
		assert.Equal(t, "new-token", input.Name)
		assert.Equal(t, "hetzner", input.Provider)
		assert.Equal(t, "hc-api-key-123", input.Token)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(CloudToken{UUID: "ct-new", Name: "new-token", Provider: "hetzner"})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	ct, err := c.CreateCloudToken(context.Background(), CreateCloudTokenInput{
		Name:     "new-token",
		Provider: "hetzner",
		Token:    "hc-api-key-123",
	})
	require.NoError(t, err)
	assert.Equal(t, "ct-new", ct.UUID)
	assert.Equal(t, "new-token", ct.Name)
	assert.Equal(t, "hetzner", ct.Provider)
}

func TestClient_CreateCloudToken_WrongStatusCode(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK) // 200, not the expected 201
		json.NewEncoder(w).Encode(CloudToken{UUID: "ct-wrong"})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	_, err := c.CreateCloudToken(context.Background(), CreateCloudTokenInput{
		Name: "t", Provider: "hetzner", Token: "tok",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected status 201")
	assert.Contains(t, err.Error(), "got 200")
}

func TestClient_UpdateCloudToken(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPatch, r.Method)
		assert.Equal(t, "/api/v1/cloud-tokens/ct-upd", r.URL.Path)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var input UpdateCloudTokenInput
		require.NoError(t, json.Unmarshal(body, &input))
		require.NotNil(t, input.Name)
		assert.Equal(t, "renamed-token", *input.Name)
		require.NotNil(t, input.Token)
		assert.Equal(t, "new-secret", *input.Token)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(CloudToken{UUID: "ct-upd", Name: "renamed-token", Provider: "hetzner"})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	name := "renamed-token"
	token := "new-secret"
	ct, err := c.UpdateCloudToken(context.Background(), "ct-upd", UpdateCloudTokenInput{
		Name:  &name,
		Token: &token,
	})
	require.NoError(t, err)
	assert.Equal(t, "ct-upd", ct.UUID)
	assert.Equal(t, "renamed-token", ct.Name)
}

func TestClient_UpdateCloudToken_PartialUpdate(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		// Verify omitempty: only "name" should be present, not "token"
		var raw map[string]interface{}
		require.NoError(t, json.Unmarshal(body, &raw))
		_, hasName := raw["name"]
		assert.True(t, hasName, "expected 'name' in request body")
		_, hasToken := raw["token"]
		assert.False(t, hasToken, "expected 'token' to be omitted when nil")

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(CloudToken{UUID: "ct-partial", Name: "only-name"})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	name := "only-name"
	ct, err := c.UpdateCloudToken(context.Background(), "ct-partial", UpdateCloudTokenInput{
		Name: &name,
	})
	require.NoError(t, err)
	assert.Equal(t, "ct-partial", ct.UUID)
	assert.Equal(t, "only-name", ct.Name)
}

func TestClient_DeleteCloudToken(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "/api/v1/cloud-tokens/ct-del", r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	err := c.DeleteCloudToken(context.Background(), "ct-del")
	require.NoError(t, err)
}

func TestClient_DeleteCloudToken_NotFound(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	err := c.DeleteCloudToken(context.Background(), "nonexistent")
	require.Error(t, err)
	assert.True(t, IsNotFound(err))
}

func TestClient_ValidateCloudToken(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/cloud-tokens/ct-val/validate", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	err := c.ValidateCloudToken(context.Background(), "ct-val")
	require.NoError(t, err)
}

func TestClient_ValidateCloudToken_NotFound(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	err := c.ValidateCloudToken(context.Background(), "nonexistent")
	require.Error(t, err)
	assert.True(t, IsNotFound(err))
}

// --- S3 Storages ---

func TestClient_ListS3Storages(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/s3-storages", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]S3Storage{
			{UUID: "s3-1", Name: "minio", Endpoint: "http://minio:9000", Bucket: "b1", Region: "us-east-1"},
			{UUID: "s3-2", Name: "aws", Endpoint: "https://s3.amazonaws.com", Bucket: "b2", Region: "eu-west-1"},
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	list, err := c.ListS3Storages(context.Background())
	require.NoError(t, err)
	require.Len(t, list, 2)
	assert.Equal(t, "s3-1", list[0].UUID)
	assert.Equal(t, "minio", list[0].Name)
	assert.Equal(t, "s3-2", list[1].UUID)
}

func TestClient_GetS3Storage(t *testing.T) {
	t.Parallel()
	usable := true
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/s3-storages/s3-uuid-1", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(S3Storage{
			UUID: "s3-uuid-1", Name: "my-s3", Description: "desc",
			Endpoint: "https://s3.us-east-1.amazonaws.com", Bucket: "my-bucket",
			Region: "us-east-1", Key: "ak", Secret: "sk", IsUsable: &usable,
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	s, err := c.GetS3Storage(context.Background(), "s3-uuid-1")
	require.NoError(t, err)
	assert.Equal(t, "s3-uuid-1", s.UUID)
	assert.Equal(t, "my-s3", s.Name)
	assert.Equal(t, "my-bucket", s.Bucket)
	assert.Equal(t, "ak", s.Key)
	require.NotNil(t, s.IsUsable)
	assert.True(t, *s.IsUsable)
}

func TestClient_GetS3Storage_NotFound(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	_, err := c.GetS3Storage(context.Background(), "nonexistent")
	require.Error(t, err)
	assert.True(t, IsNotFound(err))
}

func TestClient_GetS3Storage_URLEscape(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/s3-storages/uuid%2Fwith%2Fslashes", r.URL.EscapedPath())
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(S3Storage{UUID: "uuid/with/slashes", Name: "escaped"})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	s, err := c.GetS3Storage(context.Background(), "uuid/with/slashes")
	require.NoError(t, err)
	assert.Equal(t, "escaped", s.Name)
}

func TestClient_CreateS3Storage(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/s3-storages", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var input CreateS3StorageInput
		require.NoError(t, json.Unmarshal(body, &input))
		assert.Equal(t, "new-s3", input.Name)
		assert.Equal(t, "https://s3.amazonaws.com", input.Endpoint)
		assert.Equal(t, "bucket", input.Bucket)
		assert.Equal(t, "us-east-1", input.Region)
		assert.Equal(t, "ak", input.Key)
		assert.Equal(t, "sk", input.Secret)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(S3Storage{UUID: "s3-new", Name: "new-s3"})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	s, err := c.CreateS3Storage(context.Background(), CreateS3StorageInput{
		Name: "new-s3", Endpoint: "https://s3.amazonaws.com", Bucket: "bucket",
		Region: "us-east-1", Key: "ak", Secret: "sk",
	})
	require.NoError(t, err)
	assert.Equal(t, "s3-new", s.UUID)
	assert.Equal(t, "new-s3", s.Name)
}

func TestClient_CreateS3Storage_WrongStatusCode(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(S3Storage{UUID: "s3-wrong"})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	_, err := c.CreateS3Storage(context.Background(), CreateS3StorageInput{
		Name: "t", Endpoint: "https://s3.amazonaws.com", Bucket: "b", Region: "us-east-1", Key: "k", Secret: "s",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected status 201")
	assert.Contains(t, err.Error(), "got 200")
}

func TestClient_UpdateS3Storage(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPatch, r.Method)
		assert.Equal(t, "/api/v1/s3-storages/s3-upd", r.URL.Path)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var input UpdateS3StorageInput
		require.NoError(t, json.Unmarshal(body, &input))
		require.NotNil(t, input.Name)
		assert.Equal(t, "renamed", *input.Name)
		require.NotNil(t, input.Bucket)
		assert.Equal(t, "new-bucket", *input.Bucket)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(S3Storage{UUID: "s3-upd", Name: "renamed"})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	name := "renamed"
	bucket := "new-bucket"
	s, err := c.UpdateS3Storage(context.Background(), "s3-upd", UpdateS3StorageInput{
		Name: &name, Bucket: &bucket,
	})
	require.NoError(t, err)
	assert.Equal(t, "s3-upd", s.UUID)
	assert.Equal(t, "renamed", s.Name)
}

func TestClient_UpdateS3Storage_PartialUpdate(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var raw map[string]interface{}
		require.NoError(t, json.Unmarshal(body, &raw))
		_, hasName := raw["name"]
		assert.True(t, hasName, "expected 'name' in request body")
		_, hasSecret := raw["secret"]
		assert.False(t, hasSecret, "expected 'secret' to be omitted when nil")

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(S3Storage{UUID: "s3-partial", Name: "only-name"})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	name := "only-name"
	s, err := c.UpdateS3Storage(context.Background(), "s3-partial", UpdateS3StorageInput{
		Name: &name,
	})
	require.NoError(t, err)
	assert.Equal(t, "s3-partial", s.UUID)
}

func TestClient_DeleteS3Storage(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "/api/v1/s3-storages/s3-del", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	err := c.DeleteS3Storage(context.Background(), "s3-del")
	require.NoError(t, err)
}

func TestClient_DeleteS3Storage_NotFound(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	err := c.DeleteS3Storage(context.Background(), "nonexistent")
	require.Error(t, err)
	assert.True(t, IsNotFound(err))
}

func TestClient_ValidateS3Storage(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/s3-storages/s3-val/validate", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	err := c.ValidateS3Storage(context.Background(), "s3-val")
	require.NoError(t, err)
}

func TestClient_ValidateS3Storage_NotFound(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	err := c.ValidateS3Storage(context.Background(), "nonexistent")
	require.Error(t, err)
	assert.True(t, IsNotFound(err))
}

// --- GitHub Apps (CRUD + Repositories/Branches) ---

func TestClient_ListGitHubApps(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/github-apps", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]GitHubApp{
			{ID: 1, UUID: "gh-1", Name: "App One", OrganizationName: "my-org", AppID: 100, InstallationID: 200, ClientID: "cid-1"},
			{ID: 2, UUID: "gh-2", Name: "App Two", AppID: 101, InstallationID: 201, ClientID: "cid-2"},
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	apps, err := c.ListGitHubApps(context.Background())
	require.NoError(t, err)
	require.Len(t, apps, 2)
	assert.Equal(t, int64(1), apps[0].ID)
	assert.Equal(t, "gh-1", apps[0].UUID)
	assert.Equal(t, "App One", apps[0].Name)
	assert.Equal(t, "my-org", apps[0].OrganizationName)
	assert.Equal(t, int64(100), apps[0].AppID)
	assert.Equal(t, int64(200), apps[0].InstallationID)
	assert.Equal(t, "cid-1", apps[0].ClientID)
	assert.Equal(t, int64(2), apps[1].ID)
	assert.Equal(t, "gh-2", apps[1].UUID)
	assert.Equal(t, "App Two", apps[1].Name)
}

func TestClient_ListGitHubApps_MalformedSuccessResponse(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]GitHubApp{{ID: 5, UUID: "gh-1"}})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	_, err := c.ListGitHubApps(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid app at index 0")
	assert.Contains(t, err.Error(), "missing name")
}

func TestClient_ListGitHubApps_SkipsSystemRecord(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]GitHubApp{
			{ID: 0, Name: "Public GitHub"},
			{ID: 1, UUID: "gh-1", Name: "Real App", AppID: 100, InstallationID: 200, ClientID: "cid-1"},
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	apps, err := c.ListGitHubApps(context.Background())
	require.NoError(t, err)
	require.Len(t, apps, 1)
	assert.Equal(t, int64(1), apps[0].ID)
	assert.Equal(t, "Real App", apps[0].Name)
}

func TestClient_CreateGitHubApp(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/github-apps", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var input CreateGitHubAppIntegrationInput
		require.NoError(t, json.Unmarshal(body, &input))
		assert.Equal(t, "My GitHub App", input.Name)
		assert.Equal(t, "my-org", input.OrganizationName)
		assert.Equal(t, "https://api.github.com", input.APIURL)
		assert.Equal(t, "https://github.com", input.HTMLURL)
		assert.Equal(t, int64(12345), input.AppID)
		assert.Equal(t, int64(67890), input.InstallationID)
		assert.Equal(t, "Iv1.abc123", input.ClientID)
		assert.Equal(t, "client-secret-val", input.ClientSecret)
		assert.Equal(t, "wh-secret-val", input.WebhookSecret)
		assert.Equal(t, "pk-uuid-1", input.PrivateKeyUUID)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(GitHubApp{ID: 42, UUID: "gh-new", Name: "My GitHub App"})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	app, err := c.CreateGitHubApp(context.Background(), CreateGitHubAppIntegrationInput{
		Name:             "My GitHub App",
		OrganizationName: "my-org",
		APIURL:           "https://api.github.com",
		HTMLURL:          "https://github.com",
		AppID:            12345,
		InstallationID:   67890,
		ClientID:         "Iv1.abc123",
		ClientSecret:     "client-secret-val",
		WebhookSecret:    "wh-secret-val",
		PrivateKeyUUID:   "pk-uuid-1",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(42), app.ID)
	assert.Equal(t, "gh-new", app.UUID)
	assert.Equal(t, "My GitHub App", app.Name)
}

func TestClient_CreateGitHubApp_WrongStatusCode(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK) // 200, not the expected 201
		json.NewEncoder(w).Encode(GitHubApp{ID: 1})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	_, err := c.CreateGitHubApp(context.Background(), CreateGitHubAppIntegrationInput{
		Name: "t", AppID: 1, InstallationID: 1, ClientID: "c", ClientSecret: "s", PrivateKeyUUID: "pk",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected status 201")
	assert.Contains(t, err.Error(), "got 200")
}

func TestClient_CreateGitHubApp_MalformedSuccessResponse(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(GitHubApp{UUID: "gh-new"})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	_, err := c.CreateGitHubApp(context.Background(), CreateGitHubAppIntegrationInput{
		Name: "t", AppID: 1, InstallationID: 1, ClientID: "c", ClientSecret: "s", PrivateKeyUUID: "pk",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing id")
}

func TestClient_UpdateGitHubApp(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPatch, r.Method)
		assert.Equal(t, "/api/v1/github-apps/42", r.URL.Path)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var input UpdateGitHubAppIntegrationInput
		require.NoError(t, json.Unmarshal(body, &input))
		require.NotNil(t, input.Name)
		assert.Equal(t, "Updated App", *input.Name)
		require.NotNil(t, input.OrganizationName)
		assert.Equal(t, "new-org", *input.OrganizationName)
		assert.Nil(t, input.AppID)
		assert.Nil(t, input.ClientSecret)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"message": "GitHub app updated successfully", "data": GitHubApp{ID: 42, UUID: "gh-upd", Name: "Updated App", OrganizationName: "new-org"}})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	name := "Updated App"
	org := "new-org"
	app, err := c.UpdateGitHubApp(context.Background(), 42, UpdateGitHubAppIntegrationInput{
		Name:             &name,
		OrganizationName: &org,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(42), app.ID)
	assert.Equal(t, "gh-upd", app.UUID)
	assert.Equal(t, "Updated App", app.Name)
	assert.Equal(t, "new-org", app.OrganizationName)
}

func TestClient_UpdateGitHubApp_PartialUpdate(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		// Verify omitempty: only "name" should be present
		var raw map[string]interface{}
		require.NoError(t, json.Unmarshal(body, &raw))
		_, hasName := raw["name"]
		assert.True(t, hasName, "expected 'name' in request body")
		_, hasAppID := raw["app_id"]
		assert.False(t, hasAppID, "expected 'app_id' to be omitted when nil")
		_, hasClientSecret := raw["client_secret"]
		assert.False(t, hasClientSecret, "expected 'client_secret' to be omitted when nil")

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"message": "GitHub app updated successfully", "data": GitHubApp{ID: 10, Name: "partial"}})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	name := "partial"
	app, err := c.UpdateGitHubApp(context.Background(), 10, UpdateGitHubAppIntegrationInput{Name: &name})
	require.NoError(t, err)
	assert.Equal(t, "partial", app.Name)
}

func TestClient_UpdateGitHubApp_MalformedSuccessResponse(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"message": "GitHub app updated successfully", "data": map[string]any{"name": "missing-id"}})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	name := "Updated App"
	_, err := c.UpdateGitHubApp(context.Background(), 42, UpdateGitHubAppIntegrationInput{Name: &name})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing id")
}

func TestClient_DeleteGitHubApp(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "/api/v1/github-apps/99", r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	err := c.DeleteGitHubApp(context.Background(), 99)
	require.NoError(t, err)
}

func TestClient_DeleteGitHubApp_NotFound(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	err := c.DeleteGitHubApp(context.Background(), 999)
	require.Error(t, err)
	assert.True(t, IsNotFound(err))
}

func TestClient_ListGitHubAppRepositories(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/github-apps/42/repositories", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"repositories": []GitHubRepository{
			{Name: "repo-one", FullName: "org/repo-one", Private: false},
			{Name: "repo-two", FullName: "org/repo-two", Private: true},
		}})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	repos, err := c.ListGitHubAppRepositories(context.Background(), 42)
	require.NoError(t, err)
	require.Len(t, repos, 2)
	assert.Equal(t, "repo-one", repos[0].Name)
	assert.Equal(t, "org/repo-one", repos[0].FullName)
	assert.False(t, repos[0].Private)
	assert.Equal(t, "repo-two", repos[1].Name)
	assert.Equal(t, "org/repo-two", repos[1].FullName)
	assert.True(t, repos[1].Private)
}

func TestClient_ListGitHubAppBranches(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/github-apps/42/repositories/my-org/my-repo/branches", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"branches": []GitHubBranch{
			{Name: "main"},
			{Name: "develop"},
			{Name: "feature/new-thing"},
		}})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	branches, err := c.ListGitHubAppBranches(context.Background(), 42, "my-org", "my-repo")
	require.NoError(t, err)
	require.Len(t, branches, 3)
	assert.Equal(t, "main", branches[0].Name)
	assert.Equal(t, "develop", branches[1].Name)
	assert.Equal(t, "feature/new-thing", branches[2].Name)
}

func TestClient_ListGitHubAppBranches_URLEscape(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		// owner "my org" and repo "my repo" should be URL-escaped
		assert.Equal(t, "/api/v1/github-apps/10/repositories/my%20org/my%20repo/branches", r.URL.EscapedPath())
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"branches": []GitHubBranch{{Name: "main"}}})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	branches, err := c.ListGitHubAppBranches(context.Background(), 10, "my org", "my repo")
	require.NoError(t, err)
	require.Len(t, branches, 1)
	assert.Equal(t, "main", branches[0].Name)
}

func TestClient_ListGitHubAppRepositories_NotFound(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	_, err := c.ListGitHubAppRepositories(context.Background(), 999)
	require.Error(t, err)
	assert.True(t, IsNotFound(err))
}

// --- Database Environment Variables ---

func TestClient_CreateEnvVar_Database(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/databases/db-1/envs", r.URL.Path)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var bodyMap map[string]any
		require.NoError(t, json.Unmarshal(body, &bodyMap))
		assert.Equal(t, "DB_HOST", bodyMap["key"])
		assert.Equal(t, "localhost", bodyMap["value"])
		assert.NotContains(t, bodyMap, "is_buildtime")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(CreateEnvVarResponse{UUID: "db-env-new"})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	resp, err := c.CreateEnvVar(context.Background(), "databases", "db-1", EnvironmentVariable{
		Key:     "DB_HOST",
		Value:   "localhost",
		IsBuild: true,
	}, nil)
	require.NoError(t, err)
	assert.Equal(t, "db-env-new", resp.UUID)
}

func TestClient_ListEnvVars_Database(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/databases/db-1/envs", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]EnvironmentVariable{
			{UUID: "dev-1", Key: "DB_PORT", Value: "5432", IsPreview: false, IsBuild: false},
			{UUID: "dev-2", Key: "DB_PASS", Value: "secret", IsPreview: true, IsBuild: true},
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	vars, err := c.ListEnvVars(context.Background(), "databases", "db-1")
	require.NoError(t, err)
	require.Len(t, vars, 2)
	assert.Equal(t, "DB_PORT", vars[0].Key)
	assert.Equal(t, "5432", vars[0].Value)
	assert.False(t, vars[0].IsPreview)
	assert.Equal(t, "DB_PASS", vars[1].Key)
	assert.True(t, vars[1].IsPreview)
	assert.True(t, vars[1].IsBuild)
}

func TestClient_UpdateEnvVar_Database(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPatch, r.Method)
		assert.Equal(t, "/api/v1/databases/db-env-1/envs", r.URL.Path)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var bodyMap map[string]any
		require.NoError(t, json.Unmarshal(body, &bodyMap))
		assert.Equal(t, "DB_HOST", bodyMap["key"])
		assert.Equal(t, "new-host", bodyMap["value"])
		assert.NotContains(t, bodyMap, "is_buildtime")

		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	err := c.UpdateEnvVar(context.Background(), "databases", "db-env-1", EnvironmentVariable{
		Key:   "DB_HOST",
		Value: "new-host",
	}, nil)
	require.NoError(t, err)
}

func TestClient_CreateEnvVar_Application_ExtendedFlags(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var bodyMap map[string]any
		require.NoError(t, json.Unmarshal(body, &bodyMap))
		assert.Equal(t, "APP_SECRET", bodyMap["key"])
		assert.Equal(t, false, bodyMap["is_runtime"])
		assert.Equal(t, true, bodyMap["is_buildtime"])
		assert.Equal(t, true, bodyMap["is_literal"])
		assert.Equal(t, true, bodyMap["is_multiline"])
		assert.Equal(t, "build only", bodyMap["comment"])
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(CreateEnvVarResponse{UUID: "env-ext"})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	isBuild, isRuntime, isLiteral, isMulti := true, false, true, true
	comment := "build only"
	resp, err := c.CreateEnvVar(context.Background(), "applications", "app-1", EnvironmentVariable{
		Key:   "APP_SECRET",
		Value: "secret",
	}, &EnvVarWriteOpts{
		IsBuild:     &isBuild,
		IsRuntime:   &isRuntime,
		IsLiteral:   &isLiteral,
		IsMultiline: &isMulti,
		Comment:     &comment,
	})
	require.NoError(t, err)
	assert.Equal(t, "env-ext", resp.UUID)
}

func TestClient_CreateEnvVar_Service_LiteralComment(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var bodyMap map[string]any
		require.NoError(t, json.Unmarshal(body, &bodyMap))
		assert.Equal(t, true, bodyMap["is_literal"])
		assert.Equal(t, "note", bodyMap["comment"])
		assert.NotContains(t, bodyMap, "is_buildtime")
		assert.NotContains(t, bodyMap, "is_runtime")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(CreateEnvVarResponse{UUID: "svc-ext"})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	isLiteral := true
	comment := "note"
	resp, err := c.CreateEnvVar(context.Background(), "services", "svc-1", EnvironmentVariable{
		Key:   "FLAG",
		Value: "1",
	}, &EnvVarWriteOpts{IsLiteral: &isLiteral, Comment: &comment})
	require.NoError(t, err)
	assert.Equal(t, "svc-ext", resp.UUID)
}

func TestClient_DeleteEnvVar_Database(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "/api/v1/databases/db-1/envs/dev-del", r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	err := c.DeleteEnvVar(context.Background(), "databases", "db-1", "dev-del")
	require.NoError(t, err)
}

// --- Teams (remaining) ---

func TestClient_ListTeams(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/teams", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]Team{
			{ID: 1, Name: "engineering", Description: "Engineering team"},
			{ID: 2, Name: "design", Description: "Design team"},
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	teams, err := c.ListTeams(context.Background())
	require.NoError(t, err)
	require.Len(t, teams, 2)
	assert.Equal(t, 1, teams[0].ID)
	assert.Equal(t, "engineering", teams[0].Name)
	assert.Equal(t, "Engineering team", teams[0].Description)
	assert.Equal(t, 2, teams[1].ID)
	assert.Equal(t, "design", teams[1].Name)
	assert.Equal(t, "Design team", teams[1].Description)
}

func TestClient_GetCurrentTeam(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/teams/current", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Team{
			ID:          10,
			Name:        "my-team",
			Description: "Current team",
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	team, err := c.GetCurrentTeam(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 10, team.ID)
	assert.Equal(t, "my-team", team.Name)
	assert.Equal(t, "Current team", team.Description)
}

func TestClient_GetCurrentTeamMembers(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/teams/current/members", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]TeamMember{
			{ID: 1, Name: "Charlie", Email: "charlie@example.com"},
			{ID: 2, Name: "Dana", Email: "dana@example.com"},
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	members, err := c.GetCurrentTeamMembers(context.Background())
	require.NoError(t, err)
	require.Len(t, members, 2)
	assert.Equal(t, 1, members[0].ID)
	assert.Equal(t, "Charlie", members[0].Name)
	assert.Equal(t, "charlie@example.com", members[0].Email)
	assert.Equal(t, 2, members[1].ID)
	assert.Equal(t, "Dana", members[1].Name)
	assert.Equal(t, "dana@example.com", members[1].Email)
}

// --- Deployments (remaining) ---

func TestClient_ListApplicationDeployments(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/deployments/applications/app-dep-1", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]Deployment{
			{UUID: "dep-a1", ID: 300, Status: "finished", ServerUUID: "srv-1"},
			{UUID: "dep-a2", ID: 301, Status: "queued", ServerUUID: "srv-1"},
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	deps, err := c.ListApplicationDeployments(context.Background(), "app-dep-1")
	require.NoError(t, err)
	require.Len(t, deps, 2)
	assert.Equal(t, "dep-a1", deps[0].UUID)
	assert.Equal(t, 300, deps[0].ID)
	assert.Equal(t, "finished", deps[0].Status)
	assert.Equal(t, "dep-a2", deps[1].UUID)
	assert.Equal(t, 301, deps[1].ID)
	assert.Equal(t, "queued", deps[1].Status)
}

func TestClient_CancelDeployment(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/deployments/dep-cancel-1/cancel", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	err := c.CancelDeployment(context.Background(), "dep-cancel-1")
	require.NoError(t, err)
}

func TestClient_ListApplicationDeployments_WrappedObject(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/deployments/applications/app-wrap-1", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"count": 2,
			"deployments": []Deployment{
				{UUID: "dep-wrap-1", Status: "finished"},
				{UUID: "dep-wrap-2", Status: "in_progress"},
			},
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	deps, err := c.ListApplicationDeployments(context.Background(), "app-wrap-1")
	require.NoError(t, err)
	require.Len(t, deps, 2)
	assert.Equal(t, "dep-wrap-1", deps[0].UUID)
	assert.Equal(t, "finished", deps[0].Status)
	assert.Equal(t, "dep-wrap-2", deps[1].UUID)
	assert.Equal(t, "in_progress", deps[1].Status)
}

func TestClient_DeployApplication(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/deploy", r.URL.Path)
		assert.Equal(t, "app-deploy-1", r.URL.Query().Get("uuid"))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"deployments": []map[string]string{{
				"message":         "Application nginx deployment queued.",
				"resource_uuid":   "app-deploy-1",
				"deployment_uuid": "dep-from-deploy",
			}},
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	got, err := c.DeployApplication(context.Background(), "app-deploy-1")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "dep-from-deploy", got.DeploymentUUID)
	assert.Equal(t, "Application nginx deployment queued.", got.Message)
}

func TestClient_DeployApplication_SkipOmitsUUID(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/deploy", r.URL.Path)
		assert.Equal(t, "app-skip-1", r.URL.Query().Get("uuid"))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"deployments": []map[string]string{{
				"message":       "Deployment already queued for this commit.",
				"resource_uuid": "app-skip-1",
			}},
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	got, err := c.DeployApplication(context.Background(), "app-skip-1")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Empty(t, got.DeploymentUUID)
	assert.Equal(t, "Deployment already queued for this commit.", got.Message)
}

func TestClient_LatestApplicationDeploymentUUID(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/deployments/applications/app-latest-1", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"count": 2,
			"deployments": []Deployment{
				{UUID: "dep-finished", Status: "finished"},
				{UUID: "dep-queued", Status: "queued"},
			},
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	got, err := c.LatestApplicationDeploymentUUID(context.Background(), "app-latest-1")
	require.NoError(t, err)
	assert.Equal(t, "dep-queued", got)
}

func TestPickDeploymentUUID(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		deps []Deployment
		want string
	}{
		{name: "empty", want: ""},
		{
			name: "prefer queued over earlier finished",
			deps: []Deployment{
				{UUID: "fin", Status: "finished"},
				{UUID: "q", Status: "queued"},
			},
			want: "q",
		},
		{
			name: "prefer in_progress over finished",
			deps: []Deployment{
				{UUID: "fin", Status: "finished"},
				{UUID: "ip", Status: "in_progress"},
			},
			want: "ip",
		},
		{
			name: "status is case insensitive",
			deps: []Deployment{{UUID: "q", Status: "QUEUED"}},
			want: "q",
		},
		{
			name: "fallback to first non-empty uuid",
			deps: []Deployment{
				{UUID: "", Status: "finished"},
				{UUID: "first", Status: "finished"},
				{UUID: "second", Status: "finished"},
			},
			want: "first",
		},
		{
			name: "skip in-flight without uuid",
			deps: []Deployment{
				{UUID: "", Status: "queued"},
				{UUID: "fin", Status: "finished"},
			},
			want: "fin",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, pickDeploymentUUID(tt.deps))
		})
	}
}

func TestClient_Deploy(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/deploy", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	err := c.Deploy(context.Background())
	require.NoError(t, err)
}

// --- Databases (remaining) ---

func TestClient_RestartDatabase(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/databases/db-restart/restart", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	err := c.RestartDatabase(context.Background(), "db-restart")
	require.NoError(t, err)
}

func TestClient_ListBackupExecutions(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/databases/db-uuid-1/backups/bk-uuid-1/executions", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(struct {
			Executions []BackupExecution `json:"executions"`
		}{
			Executions: []BackupExecution{
				{UUID: "exec-1", Status: "success", CreatedAt: "2025-01-01T00:00:00Z", Size: 1024},
				{UUID: "exec-2", Status: "failed", CreatedAt: "2025-01-02T00:00:00Z", Size: 0},
			},
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	execs, err := c.ListBackupExecutions(context.Background(), "db-uuid-1", "bk-uuid-1")
	require.NoError(t, err)
	require.Len(t, execs, 2)
	assert.Equal(t, "exec-1", execs[0].UUID)
	assert.Equal(t, "success", execs[0].Status)
	assert.Equal(t, "2025-01-01T00:00:00Z", execs[0].CreatedAt)
	assert.Equal(t, int64(1024), execs[0].Size)
	assert.Equal(t, "exec-2", execs[1].UUID)
	assert.Equal(t, "failed", execs[1].Status)
	assert.Equal(t, "2025-01-02T00:00:00Z", execs[1].CreatedAt)
}

func TestClient_DeleteBackupExecution(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "/api/v1/databases/db-uuid-1/backups/bk-uuid-1/executions/exec-del-1", r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	err := c.DeleteBackupExecution(context.Background(), "db-uuid-1", "bk-uuid-1", "exec-del-1")
	require.NoError(t, err)
}

// --- Applications (remaining) ---

func TestClient_CreateGitHubAppApplication(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/applications/private-github-app", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var input CreateGitHubAppInput
		require.NoError(t, json.Unmarshal(body, &input))
		assert.Equal(t, "proj-1", input.ProjectUUID)
		assert.Equal(t, "srv-1", input.ServerUUID)
		assert.Equal(t, "production", input.EnvironmentName)
		assert.Equal(t, "gh-app-uuid", input.GitHubAppUUID)
		assert.Equal(t, "https://github.com/org/repo", input.GitRepository)
		assert.Equal(t, "main", input.GitBranch)
		assert.Equal(t, "dockerfile", input.BuildPack)
		assert.Equal(t, "8080", input.PortsExposes)
		assert.Equal(t, "my-ghapp", input.Name)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(Application{UUID: "ghapp-new", Name: "my-ghapp"})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	app, err := c.CreateGitHubAppApplication(context.Background(), CreateGitHubAppInput{
		ProjectUUID:     "proj-1",
		ServerUUID:      "srv-1",
		EnvironmentName: "production",
		GitHubAppUUID:   "gh-app-uuid",
		GitRepository:   "https://github.com/org/repo",
		GitBranch:       "main",
		BuildPack:       "dockerfile",
		PortsExposes:    "8080",
		Name:            "my-ghapp",
	})
	require.NoError(t, err)
	assert.Equal(t, "ghapp-new", app.UUID)
	assert.Equal(t, "my-ghapp", app.Name)
}

func TestClient_GetApplicationLogs(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/applications/app-logs-1/logs", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]ApplicationLog{
			{Line: "Server started on port 3000", Timestamp: "2025-06-01T12:00:00Z"},
			{Line: "Connected to database", Timestamp: "2025-06-01T12:00:01Z"},
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	logs, err := c.GetApplicationLogs(context.Background(), "app-logs-1")
	require.NoError(t, err)
	require.Len(t, logs, 2)
	assert.Equal(t, "Server started on port 3000", logs[0].Line)
	assert.Equal(t, "2025-06-01T12:00:00Z", logs[0].Timestamp)
	assert.Equal(t, "Connected to database", logs[1].Line)
	assert.Equal(t, "2025-06-01T12:00:01Z", logs[1].Timestamp)
}

func TestClient_DeletePreviewDeployment(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "/api/v1/applications/app-prev-1/previews/42", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	err := c.DeletePreviewDeployment(context.Background(), "app-prev-1", 42)
	require.NoError(t, err)
}

// --- Client Enable/Disable API ---

func TestClient_EnableAPI(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/enable", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	err := c.EnableAPI(context.Background())
	require.NoError(t, err)
}

func TestClient_DisableAPI(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/disable", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	err := c.DisableAPI(context.Background())
	require.NoError(t, err)
}

func TestClient_EnableMCP(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/mcp/enable", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	err := c.EnableMCP(context.Background())
	require.NoError(t, err)
}

func TestClient_DisableMCP(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/mcp/disable", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	err := c.DisableMCP(context.Background())
	require.NoError(t, err)
}

// --- Resources ---

func TestClient_ListResources(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/resources", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]Resource{
			{UUID: "res-1", Name: "my-app", Type: "application", Status: "running"},
			{UUID: "res-2", Name: "my-db", Type: "database", Status: "stopped"},
			{UUID: "res-3", Name: "my-svc", Type: "service", Status: "running"},
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	resources, err := c.ListResources(context.Background())
	require.NoError(t, err)
	require.Len(t, resources, 3)
	assert.Equal(t, "res-1", resources[0].UUID)
	assert.Equal(t, "my-app", resources[0].Name)
	assert.Equal(t, "application", resources[0].Type)
	assert.Equal(t, "running", resources[0].Status)
	assert.Equal(t, "res-2", resources[1].UUID)
	assert.Equal(t, "my-db", resources[1].Name)
	assert.Equal(t, "database", resources[1].Type)
	assert.Equal(t, "stopped", resources[1].Status)
	assert.Equal(t, "res-3", resources[2].UUID)
	assert.Equal(t, "my-svc", resources[2].Name)
	assert.Equal(t, "service", resources[2].Type)
	assert.Equal(t, "running", resources[2].Status)
}

// ---------------------------------------------------------------------------
// RetryDelete tests
// ---------------------------------------------------------------------------

func TestRetryDelete_ImmediateSuccess(t *testing.T) {
	err := RetryDelete(context.Background(), 3, time.Millisecond,
		func() error { return nil },
		func(error) bool { return true },
	)
	assert.NoError(t, err)
}

func TestRetryDelete_NotFoundIsSuccess(t *testing.T) {
	err := RetryDelete(context.Background(), 3, time.Millisecond,
		func() error { return &NotFoundError{Message: "gone"} },
		func(error) bool { return true },
	)
	assert.NoError(t, err)
}

func TestRetryDelete_NonRetryableError(t *testing.T) {
	err := RetryDelete(context.Background(), 3, time.Millisecond,
		func() error { return assert.AnError },
		func(error) bool { return false },
	)
	assert.ErrorIs(t, err, assert.AnError)
}

func TestRetryDelete_RetriesUntilSuccess(t *testing.T) {
	var calls atomic.Int32
	err := RetryDelete(context.Background(), 5, time.Millisecond,
		func() error {
			if calls.Add(1) < 3 {
				return assert.AnError
			}
			return nil
		},
		func(error) bool { return true },
	)
	assert.NoError(t, err)
	assert.Equal(t, int32(3), calls.Load())
}

func TestRetryDelete_ExhaustsAttempts(t *testing.T) {
	var calls atomic.Int32
	err := RetryDelete(context.Background(), 2, time.Millisecond,
		func() error { calls.Add(1); return assert.AnError },
		func(error) bool { return true },
	)
	assert.ErrorIs(t, err, assert.AnError)
	// 2 retries + 1 final attempt = 3 calls
	assert.Equal(t, int32(3), calls.Load())
}

func TestRetryDelete_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := RetryDelete(ctx, 10, time.Second,
		func() error { return assert.AnError },
		func(error) bool { return true },
	)
	assert.ErrorIs(t, err, context.Canceled)
}

// --- List Cache ---

func TestCachedList_HitAvoidsDuplicateRequest(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]Project{
			{UUID: "p1", Name: "Alpha"},
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")

	// First call hits the server.
	var result1 []Project
	require.NoError(t, c.doCachedList(context.Background(), "/api/v1/projects", &result1))
	require.Len(t, result1, 1)
	assert.Equal(t, "Alpha", result1[0].Name)

	// Second call returns cached data without another HTTP call.
	var result2 []Project
	require.NoError(t, c.doCachedList(context.Background(), "/api/v1/projects", &result2))
	require.Len(t, result2, 1)
	assert.Equal(t, "Alpha", result2[0].Name)

	assert.Equal(t, int32(1), calls.Load(), "expected exactly 1 HTTP call, second should be cached")
}

func TestCachedList_InvalidationForcesRefresh(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		if n == 1 {
			json.NewEncoder(w).Encode([]Project{{UUID: "p1", Name: "Before"}})
		} else {
			json.NewEncoder(w).Encode([]Project{{UUID: "p1", Name: "After"}})
		}
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	path := "/api/v1/projects"

	var r1 []Project
	require.NoError(t, c.doCachedList(context.Background(), path, &r1))
	assert.Equal(t, "Before", r1[0].Name)

	c.listCache.invalidate(path)

	var r2 []Project
	require.NoError(t, c.doCachedList(context.Background(), path, &r2))
	assert.Equal(t, "After", r2[0].Name)
	assert.Equal(t, int32(2), calls.Load())
}

func TestCachedList_404ReturnsNotFoundError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	var result []Project
	err := c.doCachedList(context.Background(), "/api/v1/missing", &result)
	require.Error(t, err)
	assert.True(t, IsNotFound(err))
}

func TestCachedList_DifferentPathsIndependent(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]Project{{UUID: "p1", Name: r.URL.Path}})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")

	var r1, r2 []Project
	require.NoError(t, c.doCachedList(context.Background(), "/path/a", &r1))
	require.NoError(t, c.doCachedList(context.Background(), "/path/b", &r2))
	assert.Equal(t, "/path/a", r1[0].Name)
	assert.Equal(t, "/path/b", r2[0].Name)
	assert.Equal(t, int32(2), calls.Load())

	// Re-fetching /path/a should use cache.
	var r3 []Project
	require.NoError(t, c.doCachedList(context.Background(), "/path/a", &r3))
	assert.Equal(t, int32(2), calls.Load())
}

func TestCachedList_EvictsOnUnmarshalError(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]Project{{UUID: "p1", Name: "fresh"}})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")

	// Populate cache with a valid response for []Project.
	var projects []Project
	require.NoError(t, c.doCachedList(context.Background(), "/api/v1/projects", &projects))
	assert.Equal(t, int32(1), calls.Load())

	// Now unmarshal into an incompatible type. The cached JSON is a
	// []Project array, which cannot decode into a single string.
	// doCachedList should evict the stale entry and retry from the server.
	var single string
	err := c.doCachedList(context.Background(), "/api/v1/projects", &single)
	// The fresh fetch also returns []Project which won't decode into string,
	// so we expect an error, but the cache entry should be evicted.
	assert.Error(t, err)
	// Two server calls: original populate + one retry after eviction.
	assert.Equal(t, int32(2), calls.Load())

	// A subsequent call with the correct type should also hit the server
	// (the retry above stored new data that also failed unmarshal into
	// string, but a fresh call with []Project should work).
	var projects2 []Project
	require.NoError(t, c.doCachedList(context.Background(), "/api/v1/projects", &projects2))
	assert.Equal(t, "fresh", projects2[0].Name)
}

func TestCachedList_ConcurrentAccess(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]Project{{UUID: "p1", Name: "concurrent"}})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")

	// Warm the cache first so concurrent reads hit it.
	var warmup []Project
	require.NoError(t, c.doCachedList(context.Background(), "/api/v1/projects", &warmup))
	assert.Equal(t, int32(1), calls.Load())

	// Now launch concurrent reads; all should hit the cache.
	const goroutines = 10
	errs := make(chan error, goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			var projects []Project
			errs <- c.doCachedList(context.Background(), "/api/v1/projects", &projects)
		}()
	}
	for i := 0; i < goroutines; i++ {
		require.NoError(t, <-errs)
	}
	// Only the warmup call should have hit the server.
	assert.Equal(t, int32(1), calls.Load(),
		"all concurrent reads should use cache; expected 1 server call total")
}

func TestClient_ListGitLabApps_CachesWithinTTL(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/gitlab-apps", r.URL.Path)
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]GitLabApp{
			{ID: 1, UUID: "gl-1", Name: "GitLab App"},
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	first, err := c.ListGitLabApps(context.Background())
	require.NoError(t, err)
	require.Len(t, first, 1)
	assert.Equal(t, "GitLab App", first[0].Name)

	second, err := c.ListGitLabApps(context.Background())
	require.NoError(t, err)
	require.Len(t, second, 1)
	assert.Equal(t, "GitLab App", second[0].Name)

	assert.Equal(t, int32(1), calls.Load(), "expected exactly 1 HTTP call within 5s TTL")
}

func TestClient_ListGitLabApps_InvalidateOnWrite(t *testing.T) {
	t.Parallel()
	var listCalls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/gitlab-apps":
			n := listCalls.Add(1)
			json.NewEncoder(w).Encode([]GitLabApp{
				{ID: 1, UUID: "gl-1", Name: fmt.Sprintf("App-%d", n)},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/gitlab-apps":
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(GitLabApp{ID: 2, UUID: "gl-2", Name: "Created"})
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/gitlab-apps/1":
			json.NewEncoder(w).Encode(GitLabApp{ID: 1, UUID: "gl-1", Name: "Updated"})
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/gitlab-apps/1":
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	_, err := c.ListGitLabApps(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int32(1), listCalls.Load())

	_, err = c.CreateGitLabApp(context.Background(), CreateGitLabAppInput{Name: "Created", HTMLURL: "https://gitlab.example"})
	require.NoError(t, err)
	created, err := c.ListGitLabApps(context.Background())
	require.NoError(t, err)
	require.Len(t, created, 1)
	assert.Equal(t, "App-2", created[0].Name, "create should invalidate the list cache")
	assert.Equal(t, int32(2), listCalls.Load())

	_, err = c.UpdateGitLabApp(context.Background(), 1, UpdateGitLabAppInput{})
	require.NoError(t, err)
	updated, err := c.ListGitLabApps(context.Background())
	require.NoError(t, err)
	require.Len(t, updated, 1)
	assert.Equal(t, "App-3", updated[0].Name, "update should invalidate the list cache")
	assert.Equal(t, int32(3), listCalls.Load())

	require.NoError(t, c.DeleteGitLabApp(context.Background(), 1))
	deleted, err := c.ListGitLabApps(context.Background())
	require.NoError(t, err)
	require.Len(t, deleted, 1)
	assert.Equal(t, "App-4", deleted[0].Name, "delete should invalidate the list cache")
	assert.Equal(t, int32(4), listCalls.Load())
}

func TestClient_ListSharedEnvs_CachesWithinTTL(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/team/envs", r.URL.Path)
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]SharedEnvironmentVariable{
			{ID: 1, UUID: "env-1", Key: "FOO", Value: "bar"},
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	first, err := c.ListSharedEnvs(context.Background(), "team", "", "", "")
	require.NoError(t, err)
	require.Len(t, first, 1)
	assert.Equal(t, "FOO", first[0].Key)

	second, err := c.ListSharedEnvs(context.Background(), "team", "", "", "")
	require.NoError(t, err)
	require.Len(t, second, 1)
	assert.Equal(t, "FOO", second[0].Key)

	assert.Equal(t, int32(1), calls.Load(), "expected exactly 1 HTTP call within 5s TTL")
}

func TestClient_ListSharedEnvs_InvalidateOnWrite(t *testing.T) {
	t.Parallel()
	var listCalls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/team/envs":
			n := listCalls.Add(1)
			json.NewEncoder(w).Encode([]SharedEnvironmentVariable{
				{ID: 1, UUID: "env-1", Key: fmt.Sprintf("KEY-%d", n)},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/team/envs":
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(SharedEnvironmentVariable{ID: 2, UUID: "env-2", Key: "CREATED"})
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/team/envs/1":
			json.NewEncoder(w).Encode(SharedEnvironmentVariable{ID: 1, UUID: "env-1", Key: "UPDATED"})
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/team/envs/1":
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	_, err := c.ListSharedEnvs(context.Background(), "team", "", "", "")
	require.NoError(t, err)
	assert.Equal(t, int32(1), listCalls.Load())

	_, err = c.CreateSharedEnv(context.Background(), "team", "", "", "", SharedEnvInput{Key: "CREATED", Value: "v"})
	require.NoError(t, err)
	created, err := c.ListSharedEnvs(context.Background(), "team", "", "", "")
	require.NoError(t, err)
	require.Len(t, created, 1)
	assert.Equal(t, "KEY-2", created[0].Key, "create should invalidate the list cache")
	assert.Equal(t, int32(2), listCalls.Load())

	_, err = c.UpdateSharedEnv(context.Background(), "team", "", "", "", "1", SharedEnvInput{Value: "v2"})
	require.NoError(t, err)
	updated, err := c.ListSharedEnvs(context.Background(), "team", "", "", "")
	require.NoError(t, err)
	require.Len(t, updated, 1)
	assert.Equal(t, "KEY-3", updated[0].Key, "update should invalidate the list cache")
	assert.Equal(t, int32(3), listCalls.Load())

	require.NoError(t, c.DeleteSharedEnv(context.Background(), "team", "", "", "", "1"))
	deleted, err := c.ListSharedEnvs(context.Background(), "team", "", "", "")
	require.NoError(t, err)
	require.Len(t, deleted, 1)
	assert.Equal(t, "KEY-4", deleted[0].Key, "delete should invalidate the list cache")
	assert.Equal(t, int32(4), listCalls.Load())
}

// --- TLS / CA Cert ---

func TestClient_CustomCACert(t *testing.T) {
	t.Parallel()
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("4.0.0"))
	}))
	defer srv.Close()

	caCert := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: srv.TLS.Certificates[0].Certificate[0],
	})

	c := New(srv.URL, "test-token", RetryConfig{
		CACert: string(caCert),
	})
	version, err := c.GetVersion(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "4.0.0", version)
}

func TestClient_InsecureSkipsVerification(t *testing.T) {
	t.Parallel()
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("4.0.0"))
	}))
	defer srv.Close()

	// Without insecure, TLS should fail (self-signed cert)
	cNoInsecure := New(srv.URL, "test-token")
	_, err := cNoInsecure.GetVersion(context.Background())
	require.Error(t, err)

	// With insecure, TLS should succeed
	cInsecure := New(srv.URL, "test-token", RetryConfig{
		Insecure: true,
	})
	version, err := cInsecure.GetVersion(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "4.0.0", version)
}

func TestClient_RetryWorksWithCustomTLS(t *testing.T) {
	t.Parallel()
	var attempts int32
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&attempts, 1)
		if n <= 2 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Write([]byte("4.0.0"))
	}))
	defer srv.Close()

	caCert := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: srv.TLS.Certificates[0].Certificate[0],
	})

	c := New(srv.URL, "test-token", RetryConfig{
		CACert: string(caCert),
	})
	version, err := c.GetVersion(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "4.0.0", version)
	assert.Equal(t, int32(3), atomic.LoadInt32(&attempts), "expected 3 attempts (2 retries + 1 success)")
}

func TestRedactJSON_Object(t *testing.T) {
	t.Parallel()
	input := `{"name":"db","postgres_password":"secret123","image":"pg:16"}`
	got := redactJSON([]byte(input))
	assert.Contains(t, got, `"name":"db"`)
	assert.Contains(t, got, `[REDACTED]`)
	assert.NotContains(t, got, "secret123")
}

func TestRedactJSON_Array(t *testing.T) {
	t.Parallel()
	input := `[{"uuid":"a","redis_password":"pass1"},{"uuid":"b","redis_password":"pass2"}]`
	got := redactJSON([]byte(input))
	assert.NotContains(t, got, "pass1")
	assert.NotContains(t, got, "pass2")
	assert.Contains(t, got, `[REDACTED]`)
	assert.Contains(t, got, `"uuid":"a"`)
}

func TestRedactJSON_Nested(t *testing.T) {
	t.Parallel()
	input := `{"settings":{"password":"nested-secret"},"name":"ok"}`
	got := redactJSON([]byte(input))
	assert.NotContains(t, got, "nested-secret")
	assert.Contains(t, got, `[REDACTED]`)
	assert.Contains(t, got, `"name":"ok"`)
}

func TestRedactJSON_APIKeysAndLicenses(t *testing.T) {
	t.Parallel()
	input := `{
		"name":"drain",
		"logdrain_axiom_api_key":"axiom-secret",
		"logdrain_newrelic_license_key":"nr-license",
		"resend_api_key":"re_123",
		"pushover_user_key":"po-user"
	}`
	got := redactJSON([]byte(input))
	assert.Contains(t, got, `"name":"drain"`)
	assert.NotContains(t, got, "axiom-secret")
	assert.NotContains(t, got, "nr-license")
	assert.NotContains(t, got, "re_123")
	assert.NotContains(t, got, "po-user")
	assert.Contains(t, got, `[REDACTED]`)
}

func TestRedactJSON_InvalidJSON(t *testing.T) {
	t.Parallel()
	got := redactJSON([]byte("not json"))
	assert.Equal(t, "[non-JSON body omitted]", got)
}

func TestRedactJSON_ScriptAndWebhookURL(t *testing.T) {
	t.Parallel()
	input := `{"name":"hook","script":"#cloud-config\npassword: hunter2","webhook_url":"https://hooks.example/abc"}`
	got := redactJSON([]byte(input))
	assert.Contains(t, got, `"name":"hook"`)
	assert.Contains(t, got, `[REDACTED]`)
	assert.NotContains(t, got, "hunter2")
	assert.NotContains(t, got, "#cloud-config")
	assert.NotContains(t, got, "hooks.example")
}

func TestRedactJSON_S3KeyWithSecret(t *testing.T) {
	t.Parallel()
	input := `{"name":"backups","key":"AKIAIOSFODNN7EXAMPLE","secret":"wJalrXUtnFEMI/K7MDENG"}`
	got := redactJSON([]byte(input))
	assert.Contains(t, got, `"name":"backups"`)
	assert.NotContains(t, got, "AKIAIOSFODNN7EXAMPLE")
	assert.NotContains(t, got, "wJalrXUtnFEMI/K7MDENG")
	assert.Contains(t, got, `[REDACTED]`)
}

func TestRedactJSON_EnvKeyValueDoesNotRedactKeyName(t *testing.T) {
	t.Parallel()
	input := `{"key":"DB_PASS","value":"s3cret-env"}`
	got := redactJSON([]byte(input))
	assert.Contains(t, got, `"key":"DB_PASS"`)
	assert.NotContains(t, got, "s3cret-env")
	assert.Contains(t, got, `[REDACTED]`)
}

func TestClient_CreateDatabase_EmptyUUID(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(Database{UUID: "", Name: "broken"})
	}))
	defer srv.Close()
	c := New(srv.URL, "test-token")
	_, err := c.CreateDatabase(context.Background(), "postgresql", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty UUID")
}

func TestClient_CreateService_EmptyUUID(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(Service{UUID: "", Name: "broken"})
	}))
	defer srv.Close()
	c := New(srv.URL, "test-token")
	_, err := c.CreateService(context.Background(), CreateServiceInput{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty UUID")
}

func TestClient_CreatePublicApplication_EmptyUUID(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(Application{UUID: "", Name: "broken"})
	}))
	defer srv.Close()
	c := New(srv.URL, "test-token")
	_, err := c.CreatePublicApplication(context.Background(), CreatePublicAppInput{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty UUID")
}

func TestClient_CreatePrivateGitApplication_EmptyUUID(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(Application{UUID: "", Name: "broken"})
	}))
	defer srv.Close()
	c := New(srv.URL, "test-token")
	_, err := c.CreatePrivateGitApplication(context.Background(), CreatePrivateGitAppInput{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty UUID")
}

func TestClient_CreateDockerImageApplication_EmptyUUID(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(Application{UUID: "", Name: "broken"})
	}))
	defer srv.Close()
	c := New(srv.URL, "test-token")
	_, err := c.CreateDockerImageApplication(context.Background(), CreateDockerImageAppInput{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty UUID")
}

func TestClient_CreateDockerfileApplication_EmptyUUID(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(Application{UUID: "", Name: "broken"})
	}))
	defer srv.Close()
	c := New(srv.URL, "test-token")
	_, err := c.CreateDockerfileApplication(context.Background(), CreateDockerfileAppInput{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty UUID")
}

func TestClient_CreateGitHubAppApplication_EmptyUUID(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(Application{UUID: "", Name: "broken"})
	}))
	defer srv.Close()
	c := New(srv.URL, "test-token")
	_, err := c.CreateGitHubAppApplication(context.Background(), CreateGitHubAppInput{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty UUID")
}

// --- Hetzner List Endpoints ---

func TestClient_ListHetznerImages(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/hetzner/images", r.URL.Path)
		assert.Equal(t, "tok-uuid-1", r.URL.Query().Get("cloud_provider_token_uuid"))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]HetznerImage{
			{ID: 1, Name: "ubuntu-22.04", Description: "Ubuntu 22.04"},
			{ID: 2, Name: "debian-12", Description: "Debian 12"},
		})
	}))
	defer srv.Close()
	c := New(srv.URL, "test-token")
	images, err := c.ListHetznerImages(context.Background(), "tok-uuid-1")
	require.NoError(t, err)
	require.Len(t, images, 2)
	assert.Equal(t, int64(1), images[0].ID)
	assert.Equal(t, "ubuntu-22.04", images[0].Name)
}

func TestClient_ListHetznerImages_NotFound(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"message":"Hetzner cloud provider token not found."}`, http.StatusNotFound)
	}))
	defer srv.Close()
	c := New(srv.URL, "test-token")
	_, err := c.ListHetznerImages(context.Background(), "missing-tok")
	require.Error(t, err)
	assert.True(t, IsNotFound(err))
}

func TestClient_ListHetznerLocations(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/hetzner/locations", r.URL.Path)
		assert.Equal(t, "tok-uuid-1", r.URL.Query().Get("cloud_provider_token_uuid"))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]HetznerLocation{
			{ID: 1, Name: "fsn1", Description: "Falkenstein DC Park 1", City: "Falkenstein", Country: "DE"},
		})
	}))
	defer srv.Close()
	c := New(srv.URL, "test-token")
	locations, err := c.ListHetznerLocations(context.Background(), "tok-uuid-1")
	require.NoError(t, err)
	require.Len(t, locations, 1)
	assert.Equal(t, "fsn1", locations[0].Name)
	assert.Equal(t, "DE", locations[0].Country)
}

func TestClient_ListHetznerLocations_NotFound(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"message":"Hetzner cloud provider token not found."}`, http.StatusNotFound)
	}))
	defer srv.Close()
	c := New(srv.URL, "test-token")
	_, err := c.ListHetznerLocations(context.Background(), "missing-tok")
	require.Error(t, err)
	assert.True(t, IsNotFound(err))
}

func TestClient_ListHetznerServerTypes(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/hetzner/server-types", r.URL.Path)
		assert.Equal(t, "tok-uuid-1", r.URL.Query().Get("cloud_provider_token_uuid"))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]HetznerServerType{
			{ID: 1, Name: "cx22", Description: "CX22", Cores: 2, Memory: 4, Disk: 40},
		})
	}))
	defer srv.Close()
	c := New(srv.URL, "test-token")
	types, err := c.ListHetznerServerTypes(context.Background(), "tok-uuid-1")
	require.NoError(t, err)
	require.Len(t, types, 1)
	assert.Equal(t, "cx22", types[0].Name)
	assert.Equal(t, int64(2), types[0].Cores)
	assert.Equal(t, int64(4), types[0].Memory)
}

func TestClient_ListHetznerServerTypes_NotFound(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"message":"Hetzner cloud provider token not found."}`, http.StatusNotFound)
	}))
	defer srv.Close()
	c := New(srv.URL, "test-token")
	_, err := c.ListHetznerServerTypes(context.Background(), "missing-tok")
	require.Error(t, err)
	assert.True(t, IsNotFound(err))
}

func TestClient_ListHetznerSSHKeys(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/hetzner/ssh-keys", r.URL.Path)
		assert.Equal(t, "tok-uuid-1", r.URL.Query().Get("cloud_provider_token_uuid"))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]HetznerSSHKey{
			{ID: 1, Name: "deploy-key", Fingerprint: "aa:bb:cc:dd"},
		})
	}))
	defer srv.Close()
	c := New(srv.URL, "test-token")
	keys, err := c.ListHetznerSSHKeys(context.Background(), "tok-uuid-1")
	require.NoError(t, err)
	require.Len(t, keys, 1)
	assert.Equal(t, "deploy-key", keys[0].Name)
	assert.Equal(t, "aa:bb:cc:dd", keys[0].Fingerprint)
}

func TestClient_ListHetznerSSHKeys_NotFound(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"message":"Hetzner cloud provider token not found."}`, http.StatusNotFound)
	}))
	defer srv.Close()
	c := New(srv.URL, "test-token")
	_, err := c.ListHetznerSSHKeys(context.Background(), "missing-tok")
	require.Error(t, err)
	assert.True(t, IsNotFound(err))
}

func TestClient_ListHetznerFirewalls(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/hetzner/firewalls", r.URL.Path)
		assert.Equal(t, "tok-uuid-1", r.URL.Query().Get("cloud_provider_token_uuid"))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]HetznerFirewall{
			{ID: 38, Name: "web"},
			{ID: 39, Name: "db"},
		})
	}))
	defer srv.Close()
	c := New(srv.URL, "test-token")
	firewalls, err := c.ListHetznerFirewalls(context.Background(), "tok-uuid-1")
	require.NoError(t, err)
	require.Len(t, firewalls, 2)
	assert.Equal(t, int64(38), firewalls[0].ID)
	assert.Equal(t, "web", firewalls[0].Name)
}

func TestClient_ListHetznerFirewalls_NotFound(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"message":"Hetzner cloud provider token not found."}`, http.StatusNotFound)
	}))
	defer srv.Close()
	c := New(srv.URL, "test-token")
	_, err := c.ListHetznerFirewalls(context.Background(), "missing-tok")
	require.Error(t, err)
	assert.True(t, IsNotFound(err))
}

func TestClient_ListHetznerNetworks(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/hetzner/networks", r.URL.Path)
		assert.Equal(t, "tok-uuid-1", r.URL.Query().Get("cloud_provider_token_uuid"))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]HetznerNetwork{
			{ID: 456, Name: "acme-private", IPRange: "10.0.0.0/16"},
		})
	}))
	defer srv.Close()
	c := New(srv.URL, "test-token")
	networks, err := c.ListHetznerNetworks(context.Background(), "tok-uuid-1")
	require.NoError(t, err)
	require.Len(t, networks, 1)
	assert.Equal(t, int64(456), networks[0].ID)
	assert.Equal(t, "acme-private", networks[0].Name)
	assert.Equal(t, "10.0.0.0/16", networks[0].IPRange)
}

func TestClient_ListHetznerNetworks_NotFound(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"message":"Hetzner cloud provider token not found."}`, http.StatusNotFound)
	}))
	defer srv.Close()
	c := New(srv.URL, "test-token")
	_, err := c.ListHetznerNetworks(context.Background(), "missing-tok")
	require.Error(t, err)
	assert.True(t, IsNotFound(err))
}

// --- Cloudflare Access Headers ---

func TestClient_CFAccessHeaders_Set(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "test-cf-id", r.Header.Get("CF-Access-Client-Id"))
		assert.Equal(t, "test-cf-secret", r.Header.Get("CF-Access-Client-Secret"))
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode("4.0.0")
	}))
	defer srv.Close()
	c := New(srv.URL, "test-token", RetryConfig{
		CFAccessClientID:  "test-cf-id",
		CFAccessClientSec: "test-cf-secret",
	})
	_, err := c.GetVersion(context.Background())
	require.NoError(t, err)
}

func TestClient_CFAccessHeaders_NotSet(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Empty(t, r.Header.Get("CF-Access-Client-Id"))
		assert.Empty(t, r.Header.Get("CF-Access-Client-Secret"))
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode("4.0.0")
	}))
	defer srv.Close()
	c := New(srv.URL, "test-token")
	_, err := c.GetVersion(context.Background())
	require.NoError(t, err)
}

// --- decodeGitHubApp tests ---

func TestDecodeGitHubApp_Envelope(t *testing.T) {
	t.Parallel()
	raw := json.RawMessage(`{"message":"ok","data":{"id":7,"uuid":"abc","name":"My App"}}`)
	app, err := decodeGitHubApp(raw)
	require.NoError(t, err)
	assert.Equal(t, int64(7), app.ID)
	assert.Equal(t, "abc", app.UUID)
	assert.Equal(t, "My App", app.Name)
}

func TestDecodeGitHubApp_Direct(t *testing.T) {
	t.Parallel()
	raw := json.RawMessage(`{"id":9,"uuid":"def","name":"Direct App","organization":"my-org"}`)
	app, err := decodeGitHubApp(raw)
	require.NoError(t, err)
	assert.Equal(t, int64(9), app.ID)
	assert.Equal(t, "def", app.UUID)
	assert.Equal(t, "Direct App", app.Name)
	assert.Equal(t, "my-org", app.OrganizationName)
}

func TestDecodeGitHubApp_EnvelopeNullData(t *testing.T) {
	t.Parallel()
	// Envelope with null data falls through to direct decode which also
	// produces an invalid app (missing id/name), resulting in an error.
	raw := json.RawMessage(`{"message":"ok","data":null}`)
	_, err := decodeGitHubApp(raw)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing id")
}

func TestDecodeGitHubApp_InvalidJSON(t *testing.T) {
	t.Parallel()
	raw := json.RawMessage(`not json`)
	_, err := decodeGitHubApp(raw)
	require.Error(t, err)
}

func TestDecodeGitHubApp_DirectMissingName(t *testing.T) {
	t.Parallel()
	raw := json.RawMessage(`{"id":1}`)
	_, err := decodeGitHubApp(raw)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing name")
}

// --- decodeGitHubRepositories tests ---

func TestDecodeGitHubRepositories_DirectArray(t *testing.T) {
	t.Parallel()
	raw := json.RawMessage(`[{"name":"repo1","full_name":"org/repo1","private":false},{"name":"repo2","full_name":"org/repo2","private":true}]`)
	repos, err := decodeGitHubRepositories(raw)
	require.NoError(t, err)
	require.Len(t, repos, 2)
	assert.Equal(t, "repo1", repos[0].Name)
	assert.True(t, repos[1].Private)
}

func TestDecodeGitHubRepositories_Envelope(t *testing.T) {
	t.Parallel()
	raw := json.RawMessage(`{"repositories":[{"name":"wrapped","full_name":"org/wrapped","private":false}]}`)
	repos, err := decodeGitHubRepositories(raw)
	require.NoError(t, err)
	require.Len(t, repos, 1)
	assert.Equal(t, "wrapped", repos[0].Name)
}

func TestDecodeGitHubRepositories_InvalidJSON(t *testing.T) {
	t.Parallel()
	raw := json.RawMessage(`not json`)
	_, err := decodeGitHubRepositories(raw)
	require.Error(t, err)
}

// --- decodeGitHubBranches tests ---

func TestDecodeGitHubBranches_DirectArray(t *testing.T) {
	t.Parallel()
	raw := json.RawMessage(`[{"name":"main"},{"name":"develop"}]`)
	branches, err := decodeGitHubBranches(raw)
	require.NoError(t, err)
	require.Len(t, branches, 2)
	assert.Equal(t, "main", branches[0].Name)
	assert.Equal(t, "develop", branches[1].Name)
}

func TestDecodeGitHubBranches_Envelope(t *testing.T) {
	t.Parallel()
	raw := json.RawMessage(`{"branches":[{"name":"feature-1"}]}`)
	branches, err := decodeGitHubBranches(raw)
	require.NoError(t, err)
	require.Len(t, branches, 1)
	assert.Equal(t, "feature-1", branches[0].Name)
}

func TestDecodeGitHubBranches_InvalidJSON(t *testing.T) {
	t.Parallel()
	raw := json.RawMessage(`not json`)
	_, err := decodeGitHubBranches(raw)
	require.Error(t, err)
}

// --- API/MCP toggle error paths ---

func TestClient_EnableAPI_APIError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/enable", r.URL.Path)
		http.Error(w, `{"message":"forbidden"}`, http.StatusForbidden)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	err := c.EnableAPI(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "enabling API")
}

func TestClient_DisableAPI_APIError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/disable", r.URL.Path)
		http.Error(w, `{"message":"forbidden"}`, http.StatusForbidden)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	err := c.DisableAPI(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "disabling API")
}

func TestClient_EnableMCP_APIError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/mcp/enable", r.URL.Path)
		http.Error(w, `{"message":"not root team"}`, http.StatusForbidden)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	err := c.EnableMCP(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "enabling MCP server")
}

func TestClient_DisableMCP_APIError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/mcp/disable", r.URL.Path)
		http.Error(w, `{"message":"not root team"}`, http.StatusForbidden)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	err := c.DisableMCP(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "disabling MCP server")
}

func TestClient_CreateDigitalOceanServer(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/servers/digitalocean", r.URL.Path)
		var body map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "do-node", body["name"])
		assert.Equal(t, "tok-1", body["cloud_provider_token_uuid"])
		assert.Equal(t, "nyc1", body["region"])
		assert.Equal(t, "s-1vcpu-1gb", body["size"])
		assert.Equal(t, "ubuntu-24-04-x64", body["image"])
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(Server{UUID: "do-uuid-1", Name: "do-node", IP: "203.0.113.10"})
	}))
	defer srv.Close()
	c := New(srv.URL, "test-token")
	server, err := c.CreateDigitalOceanServer(context.Background(), CreateDigitalOceanServerInput{
		Name: "do-node", CloudProviderTokenUUID: "tok-1", Region: "nyc1", Size: "s-1vcpu-1gb",
		Image: "ubuntu-24-04-x64", PrivateKeyUUID: "pk-1",
	})
	require.NoError(t, err)
	assert.Equal(t, "do-uuid-1", server.UUID)
}

func TestClient_CreateVultrServer(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/servers/vultr", r.URL.Path)
		var body map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "vultr-node", body["name"])
		assert.EqualValues(t, 1743, body["os_id"])
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(Server{UUID: "vu-uuid-1", Name: "vultr-node"})
	}))
	defer srv.Close()
	c := New(srv.URL, "test-token")
	server, err := c.CreateVultrServer(context.Background(), CreateVultrServerInput{
		Name: "vultr-node", CloudProviderTokenUUID: "tok-1", Region: "ewr", Plan: "vc2-1c-1gb",
		OsID: 1743, PrivateKeyUUID: "pk-1",
	})
	require.NoError(t, err)
	assert.Equal(t, "vu-uuid-1", server.UUID)
}

func TestClient_CreateHetznerServer_EmptyUUID(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(Server{Name: "hz-node"})
	}))
	defer srv.Close()
	c := New(srv.URL, "test-token")
	_, err := c.CreateHetznerServer(context.Background(), CreateHetznerServerInput{
		Name: "hz-node", CloudProviderTokenUUID: "tok-1", Location: "fsn1", ServerType: "cx22",
		Image: "ubuntu-24.04", PrivateKeyUUID: "pk-1",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty UUID")
}

func TestClient_CreateDigitalOceanServer_EmptyUUID(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(Server{Name: "do-node"})
	}))
	defer srv.Close()
	c := New(srv.URL, "test-token")
	_, err := c.CreateDigitalOceanServer(context.Background(), CreateDigitalOceanServerInput{
		Name: "do-node", CloudProviderTokenUUID: "tok-1", Region: "nyc1", Size: "s-1vcpu-1gb",
		Image: "ubuntu-24-04-x64", PrivateKeyUUID: "pk-1",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty UUID")
}

func TestClient_CreateVultrServer_EmptyUUID(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(Server{Name: "vultr-node"})
	}))
	defer srv.Close()
	c := New(srv.URL, "test-token")
	_, err := c.CreateVultrServer(context.Background(), CreateVultrServerInput{
		Name: "vultr-node", CloudProviderTokenUUID: "tok-1", Region: "ewr", Plan: "vc2-1c-1gb",
		OsID: 1743, PrivateKeyUUID: "pk-1",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty UUID")
}

func TestClient_ListDigitalOceanRegions(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/digitalocean/regions", r.URL.Path)
		assert.Equal(t, "tok-1", r.URL.Query().Get("cloud_provider_token_uuid"))
		json.NewEncoder(w).Encode([]DigitalOceanRegion{{Slug: "nyc1", Name: "New York 1", Available: true}})
	}))
	defer srv.Close()
	c := New(srv.URL, "test-token")
	regions, err := c.ListDigitalOceanRegions(context.Background(), "tok-1")
	require.NoError(t, err)
	require.Len(t, regions, 1)
	assert.Equal(t, "nyc1", regions[0].Slug)
}

func TestClient_ListDigitalOceanRegions_NotFound(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"message":"DigitalOcean cloud provider token not found."}`, http.StatusNotFound)
	}))
	defer srv.Close()
	c := New(srv.URL, "test-token")
	_, err := c.ListDigitalOceanRegions(context.Background(), "missing-tok")
	require.Error(t, err)
	assert.True(t, IsNotFound(err))
}

func TestClient_ListDigitalOceanSizes(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/digitalocean/sizes", r.URL.Path)
		assert.Equal(t, "tok-uuid-1", r.URL.Query().Get("cloud_provider_token_uuid"))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]DigitalOceanSize{
			{Slug: "s-1vcpu-1gb", Memory: 1024, VCPUs: 1, Disk: 25, Available: true},
		})
	}))
	defer srv.Close()
	c := New(srv.URL, "test-token")
	sizes, err := c.ListDigitalOceanSizes(context.Background(), "tok-uuid-1")
	require.NoError(t, err)
	require.Len(t, sizes, 1)
	assert.Equal(t, "s-1vcpu-1gb", sizes[0].Slug)
	assert.Equal(t, int64(1), sizes[0].VCPUs)
}

func TestClient_ListDigitalOceanSizes_NotFound(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"message":"DigitalOcean cloud provider token not found."}`, http.StatusNotFound)
	}))
	defer srv.Close()
	c := New(srv.URL, "test-token")
	_, err := c.ListDigitalOceanSizes(context.Background(), "missing-tok")
	require.Error(t, err)
	assert.True(t, IsNotFound(err))
}

func TestClient_ListDigitalOceanImages(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/digitalocean/images", r.URL.Path)
		assert.Equal(t, "tok-uuid-1", r.URL.Query().Get("cloud_provider_token_uuid"))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]DigitalOceanImage{
			{ID: 12, Name: "24.04 x64", Distribution: "Ubuntu", Slug: "ubuntu-24-04-x64", Public: true, Type: "snapshot"},
		})
	}))
	defer srv.Close()
	c := New(srv.URL, "test-token")
	images, err := c.ListDigitalOceanImages(context.Background(), "tok-uuid-1")
	require.NoError(t, err)
	require.Len(t, images, 1)
	assert.Equal(t, int64(12), images[0].ID)
	assert.Equal(t, "ubuntu-24-04-x64", images[0].Slug)
}

func TestClient_ListDigitalOceanImages_NotFound(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"message":"DigitalOcean cloud provider token not found."}`, http.StatusNotFound)
	}))
	defer srv.Close()
	c := New(srv.URL, "test-token")
	_, err := c.ListDigitalOceanImages(context.Background(), "missing-tok")
	require.Error(t, err)
	assert.True(t, IsNotFound(err))
}

func TestClient_ListDigitalOceanSSHKeys(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/digitalocean/ssh-keys", r.URL.Path)
		assert.Equal(t, "tok-uuid-1", r.URL.Query().Get("cloud_provider_token_uuid"))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]DigitalOceanSSHKey{
			{ID: 7, Name: "deploy", Fingerprint: "aa:bb:cc"},
		})
	}))
	defer srv.Close()
	c := New(srv.URL, "test-token")
	keys, err := c.ListDigitalOceanSSHKeys(context.Background(), "tok-uuid-1")
	require.NoError(t, err)
	require.Len(t, keys, 1)
	assert.Equal(t, "deploy", keys[0].Name)
	assert.Equal(t, "aa:bb:cc", keys[0].Fingerprint)
}

func TestClient_ListDigitalOceanSSHKeys_NotFound(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"message":"DigitalOcean cloud provider token not found."}`, http.StatusNotFound)
	}))
	defer srv.Close()
	c := New(srv.URL, "test-token")
	_, err := c.ListDigitalOceanSSHKeys(context.Background(), "missing-tok")
	require.Error(t, err)
	assert.True(t, IsNotFound(err))
}

func TestClient_ListVultrRegions(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/vultr/regions", r.URL.Path)
		assert.Equal(t, "tok-uuid-1", r.URL.Query().Get("cloud_provider_token_uuid"))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]VultrRegion{
			{ID: "ewr", City: "Newark", Country: "US", Continent: "North America"},
		})
	}))
	defer srv.Close()
	c := New(srv.URL, "test-token")
	regions, err := c.ListVultrRegions(context.Background(), "tok-uuid-1")
	require.NoError(t, err)
	require.Len(t, regions, 1)
	assert.Equal(t, "ewr", regions[0].ID)
	assert.Equal(t, "Newark", regions[0].City)
}

func TestClient_ListVultrRegions_NotFound(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"message":"Vultr cloud provider token not found."}`, http.StatusNotFound)
	}))
	defer srv.Close()
	c := New(srv.URL, "test-token")
	_, err := c.ListVultrRegions(context.Background(), "missing-tok")
	require.Error(t, err)
	assert.True(t, IsNotFound(err))
}

func TestClient_ListVultrPlans(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/vultr/plans", r.URL.Path)
		assert.Equal(t, "tok-uuid-1", r.URL.Query().Get("cloud_provider_token_uuid"))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]VultrPlan{
			{ID: "vc2-1c-1gb", VCPUCount: 1, RAM: 1024, Disk: 25, Type: "vc2"},
		})
	}))
	defer srv.Close()
	c := New(srv.URL, "test-token")
	plans, err := c.ListVultrPlans(context.Background(), "tok-uuid-1")
	require.NoError(t, err)
	require.Len(t, plans, 1)
	assert.Equal(t, "vc2-1c-1gb", plans[0].ID)
	assert.Equal(t, int64(1), plans[0].VCPUCount)
}

func TestClient_ListVultrPlans_NotFound(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"message":"Vultr cloud provider token not found."}`, http.StatusNotFound)
	}))
	defer srv.Close()
	c := New(srv.URL, "test-token")
	_, err := c.ListVultrPlans(context.Background(), "missing-tok")
	require.Error(t, err)
	assert.True(t, IsNotFound(err))
}

func TestClient_ListVultrOS(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/vultr/os", r.URL.Path)
		assert.Equal(t, "tok-uuid-1", r.URL.Query().Get("cloud_provider_token_uuid"))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]VultrOS{
			{ID: 1743, Name: "Ubuntu 24.04 LTS x64", Arch: "x64", Family: "ubuntu"},
		})
	}))
	defer srv.Close()
	c := New(srv.URL, "test-token")
	oses, err := c.ListVultrOS(context.Background(), "tok-uuid-1")
	require.NoError(t, err)
	require.Len(t, oses, 1)
	assert.Equal(t, int64(1743), oses[0].ID)
	assert.Equal(t, "ubuntu", oses[0].Family)
}

func TestClient_ListVultrOS_NotFound(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"message":"Vultr cloud provider token not found."}`, http.StatusNotFound)
	}))
	defer srv.Close()
	c := New(srv.URL, "test-token")
	_, err := c.ListVultrOS(context.Background(), "missing-tok")
	require.Error(t, err)
	assert.True(t, IsNotFound(err))
}

func TestClient_ListVultrSSHKeys(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/vultr/ssh-keys", r.URL.Path)
		assert.Equal(t, "tok-uuid-1", r.URL.Query().Get("cloud_provider_token_uuid"))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]VultrSSHKey{
			{ID: "abc", Name: "deploy", DateCreated: "2026-01-01"},
		})
	}))
	defer srv.Close()
	c := New(srv.URL, "test-token")
	keys, err := c.ListVultrSSHKeys(context.Background(), "tok-uuid-1")
	require.NoError(t, err)
	require.Len(t, keys, 1)
	assert.Equal(t, "abc", keys[0].ID)
	assert.Equal(t, "deploy", keys[0].Name)
}

func TestClient_ListVultrSSHKeys_NotFound(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"message":"Vultr cloud provider token not found."}`, http.StatusNotFound)
	}))
	defer srv.Close()
	c := New(srv.URL, "test-token")
	_, err := c.ListVultrSSHKeys(context.Background(), "missing-tok")
	require.Error(t, err)
	assert.True(t, IsNotFound(err))
}

func TestClient_DestinationCRUD(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/destinations", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]Destination{{UUID: "d1", Name: "net1", Network: "coolify", Type: "standalone", ServerUUID: "s1"}})
	})
	mux.HandleFunc("GET /api/v1/destinations/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "d1", r.PathValue("uuid"))
		json.NewEncoder(w).Encode(Destination{UUID: "d1", Name: "net1", Network: "coolify", Type: "standalone", ServerUUID: "s1"})
	})
	mux.HandleFunc("POST /api/v1/servers/{server_uuid}/destinations", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "s1", r.PathValue("server_uuid"))
		var input CreateDestinationInput
		require.NoError(t, json.NewDecoder(r.Body).Decode(&input))
		assert.Equal(t, "coolify", input.Network)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(Destination{UUID: "d-new", Name: input.Name, Network: input.Network, Type: "standalone", ServerUUID: "s1"})
	})
	mux.HandleFunc("DELETE /api/v1/destinations/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "d1", r.PathValue("uuid"))
		w.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	c := New(srv.URL, "test-token")

	list, err := c.ListDestinations(context.Background())
	require.NoError(t, err)
	require.Len(t, list, 1)

	got, err := c.GetDestination(context.Background(), "d1")
	require.NoError(t, err)
	assert.Equal(t, "coolify", got.Network)

	created, err := c.CreateDestination(context.Background(), "s1", CreateDestinationInput{Name: "n", Network: "coolify", Type: "standalone"})
	require.NoError(t, err)
	assert.Equal(t, "d-new", created.UUID)

	require.NoError(t, c.DeleteDestination(context.Background(), "d1"))
}

func TestClient_CreateDestination_EmptyUUID(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/servers/{server_uuid}/destinations", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(Destination{Name: "n", Network: "coolify", Type: "standalone", ServerUUID: "s1"})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	c := New(srv.URL, "test-token")

	_, err := c.CreateDestination(context.Background(), "s1", CreateDestinationInput{Network: "coolify"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty UUID")
}

func TestClient_CloudTokenDescriptionIsReadOnlyJSON(t *testing.T) {
	t.Parallel()
	// description is returned on GET when present, but never sent on create/update.
	ct := CloudToken{UUID: "ct-1", Name: "n", Description: "from-api", Provider: "hetzner"}
	data, err := json.Marshal(CreateCloudTokenInput{Name: "n", Provider: "hetzner", Token: "t"})
	require.NoError(t, err)
	var raw map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &raw))
	_, hasDesc := raw["description"]
	assert.False(t, hasDesc, "CreateCloudTokenInput must not send description (API rejects it)")
	assert.Equal(t, "from-api", ct.Description)
}

func TestClient_ResolveDestinationUUID(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/servers/{uuid}/destinations", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]Destination{
			{UUID: "d-other", Network: "other", Type: "standalone"},
			{UUID: "d-coolify", Network: "coolify", Type: "standalone"},
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	c := New(srv.URL, "test-token")

	// explicit wins
	got, err := c.ResolveDestinationUUID(context.Background(), "s1", "explicit")
	require.NoError(t, err)
	assert.Equal(t, "explicit", got)

	// prefers coolify network when multiple
	got, err = c.ResolveDestinationUUID(context.Background(), "s1", "")
	require.NoError(t, err)
	assert.Equal(t, "d-coolify", got)
}

func TestClient_CreateDockerfileApplication_ResolvesDestination(t *testing.T) {
	t.Parallel()
	var attempts int
	var gotBody map[string]any
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/servers/{uuid}/destinations", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]Destination{{UUID: "dest-1", Network: "coolify", Type: "standalone"}})
	})
	mux.HandleFunc("POST /api/v1/applications/dockerfile", func(w http.ResponseWriter, r *http.Request) {
		attempts++
		require.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
		if attempts == 1 {
			http.Error(w, `{"message":"Server has multiple destinations and you do not set destination_uuid."}`, http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(Application{UUID: "app-1"})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	c := New(srv.URL, "test-token")
	_, err := c.CreateDockerfileApplication(context.Background(), CreateDockerfileAppInput{
		ProjectUUID: "p", ServerUUID: "s", EnvironmentName: "production", Name: "n",
	})
	require.NoError(t, err)
	assert.Equal(t, 2, attempts)
	assert.Equal(t, "dest-1", gotBody["destination_uuid"])
}

// TestClient_CreateDatabase_ResolvesDestination covers the retry path in
// CreateDatabase: when Coolify rejects the first POST for missing
// destination_uuid on a multi-destination server, the client resolves a
// destination and retries. Applications resolve proactively; databases use
// this error-driven path (inputWithResolvedDestination).
func TestClient_CreateDatabase_ResolvesDestination(t *testing.T) {
	t.Parallel()
	var attempts int
	var gotBody map[string]any
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/servers/{uuid}/destinations", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]Destination{{UUID: "dest-db-1", Network: "coolify", Type: "standalone"}})
	})
	mux.HandleFunc("POST /api/v1/databases/{dbType}", func(w http.ResponseWriter, r *http.Request) {
		attempts++
		require.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
		if attempts == 1 {
			http.Error(w, `{"message":"Server has multiple destinations and you do not set destination_uuid."}`, http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(Database{UUID: "db-1", Name: "pg"})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	c := New(srv.URL, "test-token")
	db, err := c.CreateDatabase(context.Background(), "postgresql", CreatePostgresqlInput{
		CreateDatabaseBaseInput: CreateDatabaseBaseInput{
			ProjectUUID: "p", ServerUUID: "s", EnvironmentName: "production", Name: "pg",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "db-1", db.UUID)
	assert.Equal(t, 2, attempts)
	assert.Equal(t, "dest-db-1", gotBody["destination_uuid"])
}

func TestClient_CreateDatabase_ResolveDestinationFails(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	// Empty destinations list: resolve returns "" and retry is skipped.
	mux.HandleFunc("GET /api/v1/servers/{uuid}/destinations", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]Destination{})
	})
	mux.HandleFunc("POST /api/v1/databases/{dbType}", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"Server has multiple destinations and you do not set destination_uuid."}`, http.StatusBadRequest)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	c := New(srv.URL, "test-token")
	_, err := c.CreateDatabase(context.Background(), "redis", CreateRedisInput{
		CreateDatabaseBaseInput: CreateDatabaseBaseInput{
			ProjectUUID: "p", ServerUUID: "s", EnvironmentName: "production",
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "destination_uuid")
}

// TestClient_CreateService_ResolvesDestination covers multi-destination
// auto-retry for services (parity with applications/databases).
func TestClient_CreateService_ResolvesDestination(t *testing.T) {
	t.Parallel()
	var attempts int
	var gotBody map[string]any
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/servers/{uuid}/destinations", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]Destination{{UUID: "dest-svc-1", Network: "coolify", Type: "standalone"}})
	})
	mux.HandleFunc("POST /api/v1/services", func(w http.ResponseWriter, r *http.Request) {
		attempts++
		require.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
		if attempts == 1 {
			http.Error(w, `{"message":"Server has multiple destinations and you do not set destination_uuid."}`, http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(Service{UUID: "svc-1", Name: "n"})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	c := New(srv.URL, "test-token")
	svc, err := c.CreateService(context.Background(), CreateServiceInput{
		Type: "plausible", ServerUUID: "s", ProjectUUID: "p", EnvironmentName: "production",
	})
	require.NoError(t, err)
	assert.Equal(t, "svc-1", svc.UUID)
	assert.Equal(t, 2, attempts)
	assert.Equal(t, "dest-svc-1", gotBody["destination_uuid"])
}

func TestClient_CreateService_SendsExplicitDestinationUUID(t *testing.T) {
	t.Parallel()
	const dest = "dddd0001-0001-4000-8000-000000000001"
	var got CreateServiceInput
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&got))
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(Service{UUID: "svc-x"})
	}))
	defer srv.Close()
	c := New(srv.URL, "test-token")
	_, err := c.CreateService(context.Background(), CreateServiceInput{
		Type: "plausible", ServerUUID: "s", ProjectUUID: "p", EnvironmentName: "production",
		DestinationUUID: dest,
	})
	require.NoError(t, err)
	assert.Equal(t, dest, got.DestinationUUID)
}

func TestApplication_PromoteSettings(t *testing.T) {
	t.Parallel()
	preview := true
	secrets := true
	grace := int64(45)
	app := Application{
		Settings: &ApplicationSettings{
			IsPreviewDeploymentsEnabled: &preview,
			UseBuildSecrets:             &secrets,
			StopGracePeriod:             &grace,
		},
	}
	app.PromoteSettings()
	if app.IsPreviewDeploymentsEnabled == nil || !*app.IsPreviewDeploymentsEnabled {
		t.Fatal("expected is_preview_deployments_enabled promoted")
	}
	if app.UseBuildSecrets == nil || !*app.UseBuildSecrets {
		t.Fatal("expected use_build_secrets promoted")
	}
	if app.StopGracePeriod == nil || *app.StopGracePeriod != 45 {
		t.Fatalf("stop_grace_period = %v", app.StopGracePeriod)
	}
	// top-level wins over nested
	topFalse := false
	app2 := Application{
		IsPreviewDeploymentsEnabled: &topFalse,
		Settings:                    &ApplicationSettings{IsPreviewDeploymentsEnabled: &preview},
	}
	app2.PromoteSettings()
	if app2.IsPreviewDeploymentsEnabled == nil || *app2.IsPreviewDeploymentsEnabled {
		t.Fatal("top-level value must not be overwritten by settings")
	}
}

func TestClient_GetApplication_PromotesNestedSettings(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		// Real Coolify GET nests settings; top-level omit these keys.
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"uuid": "app-nested-1",
			"name": "nested-app",
			"settings": {
				"is_preview_deployments_enabled": true,
				"use_build_secrets": true,
				"stop_grace_period": 42,
				"is_auto_deploy_enabled": true,
				"is_build_server_enabled": true,
				"is_force_https_enabled": false
			}
		}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	c := New(srv.URL, "test-token")
	app, err := c.GetApplication(context.Background(), "app-nested-1")
	if err != nil {
		t.Fatal(err)
	}
	if app.IsPreviewDeploymentsEnabled == nil || !*app.IsPreviewDeploymentsEnabled {
		t.Fatalf("IsPreviewDeploymentsEnabled = %v", app.IsPreviewDeploymentsEnabled)
	}
	if app.UseBuildSecrets == nil || !*app.UseBuildSecrets {
		t.Fatalf("UseBuildSecrets = %v", app.UseBuildSecrets)
	}
	if app.StopGracePeriod == nil || *app.StopGracePeriod != 42 {
		t.Fatalf("StopGracePeriod = %v", app.StopGracePeriod)
	}
	if app.UseBuildServer == nil || !*app.UseBuildServer {
		t.Fatalf("UseBuildServer (from is_build_server_enabled) = %v", app.UseBuildServer)
	}
	if app.IsForceHTTPSEnabled == nil || *app.IsForceHTTPSEnabled {
		t.Fatalf("IsForceHTTPSEnabled = %v", app.IsForceHTTPSEnabled)
	}
}
