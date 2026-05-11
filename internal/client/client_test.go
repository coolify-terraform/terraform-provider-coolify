package client

import (
	"context"
	"encoding/json"
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
	if got := err.Error(); !contains(got, "giving up") && !contains(got, "500") {
		t.Errorf("expected error message to mention retry exhaustion or status 500, got: %s", got)
	}
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
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if app.UUID != "app-new-uuid" {
		t.Errorf("expected UUID app-new-uuid, got %s", app.UUID)
	}
}

func TestClient_CreatePostgresqlDatabase(t *testing.T) {
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
	db, err := c.CreatePostgresqlDatabase(context.Background(), CreatePostgresqlInput{
		ServerUUID:      "srv-1",
		ProjectUUID:     "proj-1",
		EnvironmentName: "production",
		PostgresUser:    "pgadmin",
		PostgresDB:      "testdb",
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

// contains checks whether s contains substr (avoids importing strings).
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
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
		assert.True(t, input.IsBuildServer)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(Server{UUID: "srv-new"})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	s, err := c.CreateServer(context.Background(), CreateServerInput{
		Name:           "New Server",
		IP:             "10.0.0.5",
		Port:           2222,
		User:           "deploy",
		PrivateKeyUUID: "pk-99",
		IsBuildServer:  true,
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
			{UUID: "pk-1", Name: "key-one", PrivateKey: "ssh-rsa AAA", IsGitRelated: true},
			{UUID: "pk-2", Name: "key-two", PrivateKey: "ssh-ed25519 BBB", IsGitRelated: false},
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	keys, err := c.ListPrivateKeys(context.Background())
	require.NoError(t, err)
	require.Len(t, keys, 2)
	assert.Equal(t, "pk-1", keys[0].UUID)
	assert.Equal(t, "key-one", keys[0].Name)
	assert.True(t, keys[0].IsGitRelated)
	assert.Equal(t, "pk-2", keys[1].UUID)
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

func TestClient_CreateMysqlDatabase(t *testing.T) {
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
	db, err := c.CreateMysqlDatabase(context.Background(), CreateMysqlInput{
		ServerUUID:      "srv-1",
		ProjectUUID:     "proj-1",
		EnvironmentName: "production",
		MysqlUser:       "myuser",
		MysqlDatabase:   "mydb",
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
		assert.Equal(t, http.MethodGet, r.Method)
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
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/databases/db-stop/stop", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	err := c.StopDatabase(context.Background(), "db-stop")
	require.NoError(t, err)
}

// --- Environment Variables ---

func TestClient_CreateApplicationEnvVar(t *testing.T) {
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

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(CreateEnvVarResponse{UUID: "env-new"})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	resp, err := c.CreateApplicationEnvVar(context.Background(), "app-1", EnvironmentVariable{
		Key:     "DATABASE_URL",
		Value:   "postgres://localhost/db",
		IsBuild: true,
	})
	require.NoError(t, err)
	assert.Equal(t, "env-new", resp.UUID)
}

func TestClient_ListApplicationEnvVars(t *testing.T) {
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
	vars, err := c.ListApplicationEnvVars(context.Background(), "app-1")
	require.NoError(t, err)
	require.Len(t, vars, 2)
	assert.Equal(t, "PORT", vars[0].Key)
	assert.Equal(t, "3000", vars[0].Value)
	assert.False(t, vars[0].IsPreview)
	assert.Equal(t, "SECRET", vars[1].Key)
	assert.True(t, vars[1].IsPreview)
	assert.True(t, vars[1].IsBuild)
}

func TestClient_DeleteApplicationEnvVar(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "/api/v1/applications/app-1/envs/env-del", r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	err := c.DeleteApplicationEnvVar(context.Background(), "app-1", "env-del")
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
		assert.Equal(t, "Updated Name", input.Name)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Service{UUID: "svc-upd", Name: "Updated Name", Type: "plausible"})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	svc, err := c.UpdateService(context.Background(), "svc-upd", UpdateServiceInput{Name: "Updated Name"})
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
			FQDN:               "https://app.example.com",
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
	assert.Equal(t, "https://app.example.com", app.FQDN)
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

func TestClient_CreateMariadbDatabase(t *testing.T) {
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
	db, err := c.CreateMariadbDatabase(context.Background(), CreateMariadbInput{
		ServerUUID:      "srv-1",
		ProjectUUID:     "proj-1",
		EnvironmentName: "production",
		MariadbUser:     "mdbuser",
		MariadbDatabase: "mdbname",
	})
	require.NoError(t, err)
	assert.Equal(t, "db-mariadb-new", db.UUID)
}

func TestClient_CreateRedisDatabase(t *testing.T) {
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
	db, err := c.CreateRedisDatabase(context.Background(), CreateRedisInput{
		ServerUUID:      "srv-1",
		ProjectUUID:     "proj-1",
		EnvironmentName: "production",
	})
	require.NoError(t, err)
	assert.Equal(t, "db-redis-new", db.UUID)
}

func TestClient_CreateMongodbDatabase(t *testing.T) {
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
	db, err := c.CreateMongodbDatabase(context.Background(), CreateMongodbInput{
		ServerUUID:              "srv-1",
		ProjectUUID:             "proj-1",
		EnvironmentName:         "production",
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

func TestClient_UpdateApplicationEnvVar(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPatch, r.Method)
		assert.Equal(t, "/api/v1/applications/app-env-1/envs", r.URL.Path)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var ev EnvironmentVariable
		require.NoError(t, json.Unmarshal(body, &ev))
		assert.Equal(t, "DATABASE_URL", ev.Key)
		assert.Equal(t, "postgres://new-host/db", ev.Value)

		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	err := c.UpdateApplicationEnvVar(context.Background(), "app-env-1", EnvironmentVariable{
		Key:   "DATABASE_URL",
		Value: "postgres://new-host/db",
	})
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

func TestClient_CreateServiceEnvVar(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/services/svc-1/envs", r.URL.Path)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var ev EnvironmentVariable
		require.NoError(t, json.Unmarshal(body, &ev))
		assert.Equal(t, "REDIS_URL", ev.Key)
		assert.Equal(t, "redis://localhost:6379", ev.Value)
		assert.True(t, ev.IsBuild)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(CreateEnvVarResponse{UUID: "svc-env-new"})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	resp, err := c.CreateServiceEnvVar(context.Background(), "svc-1", EnvironmentVariable{
		Key:     "REDIS_URL",
		Value:   "redis://localhost:6379",
		IsBuild: true,
	})
	require.NoError(t, err)
	assert.Equal(t, "svc-env-new", resp.UUID)
}

func TestClient_ListServiceEnvVars(t *testing.T) {
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
	vars, err := c.ListServiceEnvVars(context.Background(), "svc-1")
	require.NoError(t, err)
	require.Len(t, vars, 2)
	assert.Equal(t, "PORT", vars[0].Key)
	assert.Equal(t, "5000", vars[0].Value)
	assert.False(t, vars[0].IsPreview)
	assert.Equal(t, "SECRET", vars[1].Key)
	assert.True(t, vars[1].IsPreview)
	assert.True(t, vars[1].IsBuild)
}

func TestClient_UpdateServiceEnvVar(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPatch, r.Method)
		assert.Equal(t, "/api/v1/services/svc-env-1/envs", r.URL.Path)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var ev EnvironmentVariable
		require.NoError(t, json.Unmarshal(body, &ev))
		assert.Equal(t, "REDIS_URL", ev.Key)
		assert.Equal(t, "redis://new-host:6379", ev.Value)

		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	err := c.UpdateServiceEnvVar(context.Background(), "svc-env-1", EnvironmentVariable{
		Key:   "REDIS_URL",
		Value: "redis://new-host:6379",
	})
	require.NoError(t, err)
}

func TestClient_DeleteServiceEnvVar(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "/api/v1/services/svc-1/envs/sev-del", r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	err := c.DeleteServiceEnvVar(context.Background(), "svc-1", "sev-del")
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
		assert.Equal(t, http.MethodGet, r.Method)
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
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/services/svc-stop/stop", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	err := c.StopService(context.Background(), "svc-stop")
	require.NoError(t, err)
}

// --- Private Git Application ---

func TestClient_CreatePrivateGitApplication(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/applications/private-github-app", r.URL.Path)

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
			{ID: 1, UUID: "bk-1", DatabaseUUID: "db-uuid-1", Frequency: "0 * * * *", Enabled: true, RetainDays: &retain},
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
	require.NotNil(t, backups[0].RetainDays)
	assert.Equal(t, int64(7), *backups[0].RetainDays)
	assert.Equal(t, 2, backups[1].ID)
	assert.Equal(t, "bk-2", backups[1].UUID)
	assert.False(t, backups[1].Enabled)
}

func TestClient_GetDatabaseBackup(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/databases/db-uuid-1/backups/42", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(DatabaseBackup{
			ID:           42,
			UUID:         "bk-42",
			DatabaseUUID: "db-uuid-1",
			Frequency:    "0 0 * * *",
			Enabled:      true,
			DatabaseType: "postgresql",
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	b, err := c.GetDatabaseBackup(context.Background(), "db-uuid-1", 42)
	require.NoError(t, err)
	assert.Equal(t, 42, b.ID)
	assert.Equal(t, "bk-42", b.UUID)
	assert.Equal(t, "db-uuid-1", b.DatabaseUUID)
	assert.Equal(t, "0 0 * * *", b.Frequency)
	assert.True(t, b.Enabled)
	assert.Equal(t, "postgresql", b.DatabaseType)
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
		assert.Equal(t, "/api/v1/databases/db-uuid-1/backups/10", r.URL.Path)

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
	b, err := c.UpdateDatabaseBackup(context.Background(), "db-uuid-1", 10, UpdateDatabaseBackupInput{
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
		assert.Equal(t, "/api/v1/databases/db-uuid-1/backups/5", r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	err := c.DeleteDatabaseBackup(context.Background(), "db-uuid-1", 5)
	require.NoError(t, err)
}

func TestClient_ValidateServer(t *testing.T) {
	t.Parallel()
	expected := ServerValidation{Valid: true, Message: "Server is reachable"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
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

func TestClient_CreateDockerComposeApplication(t *testing.T) {
	t.Parallel()
	expected := Application{UUID: "compose-1", Name: "my-compose-app", BuildPack: "dockercompose"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/applications/dockercompose", r.URL.Path)

		body, _ := io.ReadAll(r.Body)
		var input CreateDockerComposeAppInput
		require.NoError(t, json.Unmarshal(body, &input))
		assert.Equal(t, "proj-1", input.ProjectUUID)
		assert.Equal(t, "srv-1", input.ServerUUID)
		assert.Contains(t, input.DockerComposeRaw, "version:")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(expected)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	got, err := c.CreateDockerComposeApplication(context.Background(), CreateDockerComposeAppInput{
		ProjectUUID:      "proj-1",
		ServerUUID:       "srv-1",
		EnvironmentName:  "production",
		DockerComposeRaw: "version: '3'\nservices:\n  web:\n    image: nginx",
	})
	require.NoError(t, err)
	assert.Equal(t, "compose-1", got.UUID)
	assert.Equal(t, "dockercompose", got.BuildPack)
}

func TestClient_CreateKeydbDatabase(t *testing.T) {
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
	got, err := c.CreateKeydbDatabase(context.Background(), CreateKeydbInput{
		ProjectUUID: "proj-1", ServerUUID: "srv-1", EnvironmentName: "production",
	})
	require.NoError(t, err)
	assert.Equal(t, "keydb-1", got.UUID)
	assert.Equal(t, "keydb", got.Type)
}

func TestClient_CreateDragonflyDatabase(t *testing.T) {
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
	got, err := c.CreateDragonflyDatabase(context.Background(), CreateDragonflyInput{
		ProjectUUID: "proj-1", ServerUUID: "srv-1", EnvironmentName: "production",
	})
	require.NoError(t, err)
	assert.Equal(t, "df-1", got.UUID)
	assert.Equal(t, "dragonfly", got.Type)
}

func TestClient_CreateClickhouseDatabase(t *testing.T) {
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
	got, err := c.CreateClickhouseDatabase(context.Background(), CreateClickhouseInput{
		ProjectUUID:     "proj-1",
		ServerUUID:      "srv-1",
		EnvironmentName: "production",
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
		assert.Equal(t, http.MethodGet, r.Method)
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
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/applications/app-1/stop", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	err := c.StopApplication(context.Background(), "app-1")
	require.NoError(t, err)
}

// --- S3 Storages ---

func TestClient_ListS3Storages(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/storages", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]S3Storage{
			{ID: 1, UUID: "s3-1", Name: "backup-s3", Endpoint: "https://s3.amazonaws.com", Bucket: "backups", Region: "us-east-1", AccessKey: "AK1", SecretKey: "SK1"},
			{ID: 2, UUID: "s3-2", Name: "archive-s3", Endpoint: "https://s3.eu-west-1.amazonaws.com", Bucket: "archives", Region: "eu-west-1", AccessKey: "AK2", SecretKey: "SK2"},
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	storages, err := c.ListS3Storages(context.Background())
	require.NoError(t, err)
	require.Len(t, storages, 2)
	assert.Equal(t, "s3-1", storages[0].UUID)
	assert.Equal(t, "backup-s3", storages[0].Name)
	assert.Equal(t, "https://s3.amazonaws.com", storages[0].Endpoint)
	assert.Equal(t, "backups", storages[0].Bucket)
	assert.Equal(t, "us-east-1", storages[0].Region)
	assert.Equal(t, "s3-2", storages[1].UUID)
	assert.Equal(t, "archive-s3", storages[1].Name)
	assert.Equal(t, "eu-west-1", storages[1].Region)
}

func TestClient_GetS3Storage(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/storages/s3-uuid-1", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(S3Storage{
			ID:          1,
			UUID:        "s3-uuid-1",
			Name:        "my-s3",
			Description: "Primary S3",
			Endpoint:    "https://s3.amazonaws.com",
			Bucket:      "my-bucket",
			Region:      "us-east-1",
			AccessKey:   "AKIAIOSFODNN7EXAMPLE",
			SecretKey:   "wJalrXUtnFEMI/EXAMPLE",
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	s, err := c.GetS3Storage(context.Background(), "s3-uuid-1")
	require.NoError(t, err)
	assert.Equal(t, "s3-uuid-1", s.UUID)
	assert.Equal(t, "my-s3", s.Name)
	assert.Equal(t, "Primary S3", s.Description)
	assert.Equal(t, "https://s3.amazonaws.com", s.Endpoint)
	assert.Equal(t, "my-bucket", s.Bucket)
	assert.Equal(t, "us-east-1", s.Region)
	assert.Equal(t, "AKIAIOSFODNN7EXAMPLE", s.AccessKey)
	assert.Equal(t, "wJalrXUtnFEMI/EXAMPLE", s.SecretKey)
}

func TestClient_CreateS3Storage(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/storages", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var input CreateS3StorageInput
		require.NoError(t, json.Unmarshal(body, &input))
		assert.Equal(t, "new-s3", input.Name)
		assert.Equal(t, "https://s3.amazonaws.com", input.Endpoint)
		assert.Equal(t, "new-bucket", input.Bucket)
		assert.Equal(t, "us-west-2", input.Region)
		assert.Equal(t, "AKID", input.AccessKey)
		assert.Equal(t, "SKEY", input.SecretKey)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(S3Storage{UUID: "s3-new"})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	s, err := c.CreateS3Storage(context.Background(), CreateS3StorageInput{
		Name:      "new-s3",
		Endpoint:  "https://s3.amazonaws.com",
		Bucket:    "new-bucket",
		Region:    "us-west-2",
		AccessKey: "AKID",
		SecretKey: "SKEY",
	})
	require.NoError(t, err)
	assert.Equal(t, "s3-new", s.UUID)
}

func TestClient_UpdateS3Storage(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPatch, r.Method)
		assert.Equal(t, "/api/v1/storages/s3-upd", r.URL.Path)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var input UpdateS3StorageInput
		require.NoError(t, json.Unmarshal(body, &input))
		require.NotNil(t, input.Name)
		assert.Equal(t, "renamed-s3", *input.Name)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(S3Storage{UUID: "s3-upd", Name: "renamed-s3"})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	name := "renamed-s3"
	s, err := c.UpdateS3Storage(context.Background(), "s3-upd", UpdateS3StorageInput{Name: &name})
	require.NoError(t, err)
	assert.Equal(t, "s3-upd", s.UUID)
	assert.Equal(t, "renamed-s3", s.Name)
}

func TestClient_DeleteS3Storage(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "/api/v1/storages/s3-del", r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	err := c.DeleteS3Storage(context.Background(), "s3-del")
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
			want:  `{"error":"not found","code":404}`,
		},
		{
			name:  "non-json body",
			input: []byte("plain text error"),
			want:  "plain text error",
		},
		{
			name:  "empty body",
			input: []byte(""),
			want:  "",
		},
		{
			name:  "json with empty message falls back to raw",
			input: []byte(`{"message":""}`),
			want:  `{"message":""}`,
		},
		{
			name:  "long body is truncated",
			input: []byte(strings.Repeat("x", 300)),
			want:  strings.Repeat("x", 200) + "... (truncated)",
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
	assert.Contains(t, err.Error(), "forbidden")
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
