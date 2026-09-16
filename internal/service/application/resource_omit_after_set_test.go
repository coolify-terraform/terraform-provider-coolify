//go:build !ci_app_a

package application_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestGitApplicationResource_CommandsOmitKeepsState(t *testing.T) {
	t.Parallel()

	app := client.Application{
		UUID:            "commands-omit",
		Name:            "commands-omit",
		GitRepository:   "https://github.com/example/repo",
		GitBranch:       "main",
		BuildPack:       "nixpacks",
		PortsExposes:    "3000",
		ProjectUUID:     "aaaa0001-0001-4000-8000-000000000001",
		ServerUUID:      "bbbb0001-0001-4000-8000-000000000001",
		EnvironmentName: "production",
		InstallCommand:  "npm install",
		BuildCommand:    "npm run build",
		StartCommand:    "npm start",
	}
	var deleted atomic.Bool
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/public", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"uuid": app.UUID})
	})
	mux.HandleFunc("GET /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if deleted.Load() {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(app)
	})
	mux.HandleFunc("PATCH /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		t.Error("omit-after-set must not PATCH install_command, build_command, or start_command")
		http.Error(w, `{"error":"unexpected patch"}`, http.StatusUnprocessableEntity)
	})
	mux.HandleFunc("DELETE /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		deleted.Store(true)
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	base := `
	name           = "commands-omit"
	project_uuid   = "aaaa0001-0001-4000-8000-000000000001"
	server_uuid    = "bbbb0001-0001-4000-8000-000000000001"
	git_repository = "https://github.com/example/repo"
	git_branch     = "main"
	build_pack     = "nixpacks"
	ports_exposes  = "3000"
`
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_application", "test", base+`
	install_command = "npm install"
	build_command   = "npm run build"
	start_command   = "npm start"
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_application.test", "install_command", "npm install"),
					resource.TestCheckResourceAttr("coolify_application.test", "build_command", "npm run build"),
					resource.TestCheckResourceAttr("coolify_application.test", "start_command", "npm start"),
				),
			},
			{
				Config:             acctest.TestResourceConfig(srv.URL, "coolify_application", "test", base),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestGitApplicationResource_BaseDirectoryOmitKeepsState(t *testing.T) {
	t.Parallel()
	testGitApplicationDirectoryOmitKeepsState(t, "base_directory", "/app")
}

func TestGitApplicationResource_PublishDirectoryOmitKeepsState(t *testing.T) {
	t.Parallel()
	testGitApplicationDirectoryOmitKeepsState(t, "publish_directory", "/dist")
}

func TestGitApplicationResource_WatchPathsOmitKeepsState(t *testing.T) {
	t.Parallel()
	testGitApplicationDirectoryOmitKeepsState(t, "watch_paths", "/src")
}

func testGitApplicationDirectoryOmitKeepsState(t *testing.T, attr, value string) {
	t.Helper()

	app := client.Application{
		UUID:            "dir-omit-" + attr,
		Name:            "dir-omit-" + attr,
		GitRepository:   "https://github.com/example/repo",
		GitBranch:       "main",
		BuildPack:       "nixpacks",
		PortsExposes:    "3000",
		ProjectUUID:     "aaaa0001-0001-4000-8000-000000000001",
		ServerUUID:      "bbbb0001-0001-4000-8000-000000000001",
		EnvironmentName: "production",
	}
	switch attr {
	case "base_directory":
		app.BaseDirectory = value
	case "publish_directory":
		app.PublishDirectory = value
	case "watch_paths":
		app.WatchPaths = value
	}

	// Create POSTs omit these fields; Create then PATCHes extended build/deploy
	// values. Allow that first PATCH, then fail if omit-after-set sends another.
	var allowCreatePatch atomic.Bool
	allowCreatePatch.Store(true)
	var deleted atomic.Bool
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/public", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"uuid": app.UUID})
	})
	mux.HandleFunc("GET /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if deleted.Load() {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(app)
	})
	mux.HandleFunc("PATCH /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if !allowCreatePatch.Load() {
			t.Errorf("omit-after-set must not PATCH %s", attr)
			http.Error(w, `{"error":"unexpected patch"}`, http.StatusUnprocessableEntity)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(app)
	})
	mux.HandleFunc("DELETE /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		deleted.Store(true)
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	base := `
	name           = "dir-omit-` + attr + `"
	project_uuid   = "aaaa0001-0001-4000-8000-000000000001"
	server_uuid    = "bbbb0001-0001-4000-8000-000000000001"
	git_repository = "https://github.com/example/repo"
	git_branch     = "main"
	build_pack     = "nixpacks"
	ports_exposes  = "3000"
`
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_application", "test", base+`
	`+attr+` = "`+value+`"
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_application.test", attr, value),
					func(*terraform.State) error {
						allowCreatePatch.Store(false)
						return nil
					},
				),
			},
			{
				Config:             acctest.TestResourceConfig(srv.URL, "coolify_application", "test", base),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}
