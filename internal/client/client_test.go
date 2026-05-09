package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestClient_RetryOn429(t *testing.T) {
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

func TestClient_CreateApplication(t *testing.T) {
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
