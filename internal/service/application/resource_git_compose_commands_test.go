//go:build !ci_app_a

package application_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestGitApplicationResources_DockerComposeCustomCommands(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		resourceType string
		createPath   string
		extraAttrs   string
	}{
		{
			name:         "public git",
			resourceType: "coolify_application",
			createPath:   "/api/v1/applications/public",
			extraAttrs:   "",
		},
		{
			name:         "private git",
			resourceType: "coolify_application_private_git",
			createPath:   "/api/v1/applications/private-deploy-key",
			extraAttrs:   `private_key_uuid = "dddd0001-0001-4000-8000-000000000001"`,
		},
		{
			name:         "github app",
			resourceType: "coolify_application_github_app",
			createPath:   "/api/v1/applications/private-github-app",
			extraAttrs:   `github_app_uuid = "cccc0001-0001-4000-8000-000000000001"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			app := client.Application{
				UUID:                            "compose-commands-test",
				Name:                            "compose-commands-test",
				GitRepository:                   "https://github.com/example/repo",
				GitBranch:                       "main",
				BuildPack:                       "dockercompose",
				PortsExposes:                    "3000",
				ProjectUUID:                     "aaaa0001-0001-4000-8000-000000000001",
				ServerUUID:                      "bbbb0001-0001-4000-8000-000000000001",
				EnvironmentName:                 "production",
				DockerComposeCustomStartCommand: "docker compose up -d",
				DockerComposeCustomBuildCommand: "docker compose build",
			}

			var appMu sync.Mutex
			var deleted atomic.Bool
			mux := http.NewServeMux()

			mux.HandleFunc("POST "+tt.createPath, func(w http.ResponseWriter, r *http.Request) {
				body, ok := decodeRequestBodyMap(t, w, r)
				if !ok {
					return
				}
				gotStart, ok := body["docker_compose_custom_start_command"].(string)
				if !ok || gotStart != "docker compose up -d" {
					t.Errorf("create start command = %v, want %q", body["docker_compose_custom_start_command"], "docker compose up -d")
					http.Error(w, `{"error":"unexpected docker_compose_custom_start_command"}`, http.StatusUnprocessableEntity)
					return
				}
				gotBuild, ok := body["docker_compose_custom_build_command"].(string)
				if !ok || gotBuild != "docker compose build" {
					t.Errorf("create build command = %v, want %q", body["docker_compose_custom_build_command"], "docker compose build")
					http.Error(w, `{"error":"unexpected docker_compose_custom_build_command"}`, http.StatusUnprocessableEntity)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				_ = json.NewEncoder(w).Encode(map[string]string{"uuid": app.UUID})
			})

			mux.HandleFunc("GET /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
				if r.PathValue("uuid") != app.UUID || deleted.Load() {
					http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
					return
				}
				appMu.Lock()
				defer appMu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(app)
			})

			mux.HandleFunc("PATCH /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
				if r.PathValue("uuid") != app.UUID {
					http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
					return
				}
				body, ok := decodeRequestBodyMap(t, w, r)
				if !ok {
					return
				}
				gotStart, ok := body["docker_compose_custom_start_command"].(string)
				if !ok || gotStart != "docker compose up && docker compose logs" {
					t.Errorf("update start command = %v, want chained command", body["docker_compose_custom_start_command"])
					http.Error(w, `{"error":"unexpected docker_compose_custom_start_command"}`, http.StatusUnprocessableEntity)
					return
				}
				appMu.Lock()
				app.DockerComposeCustomStartCommand = gotStart
				appMu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]string{"message": "updated"})
			})

			mux.HandleFunc("DELETE /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
				deleted.Store(true)
				w.WriteHeader(http.StatusNoContent)
			})

			srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
			defer srv.Close()

			config := acctest.TestResourceConfig(srv.URL, tt.resourceType, "test", fmt.Sprintf(`
	name                                  = "compose-commands-test"
	project_uuid                          = "aaaa0001-0001-4000-8000-000000000001"
	server_uuid                           = "bbbb0001-0001-4000-8000-000000000001"
	git_repository                        = "https://github.com/example/repo"
	git_branch                            = "main"
	build_pack                            = "dockercompose"
	ports_exposes                         = "3000"
	docker_compose_custom_start_command   = "docker compose up -d"
	docker_compose_custom_build_command   = "docker compose build"
	%s
`, tt.extraAttrs))

			updatedConfig := acctest.TestResourceConfig(srv.URL, tt.resourceType, "test", fmt.Sprintf(`
	name                                  = "compose-commands-test"
	project_uuid                          = "aaaa0001-0001-4000-8000-000000000001"
	server_uuid                           = "bbbb0001-0001-4000-8000-000000000001"
	git_repository                        = "https://github.com/example/repo"
	git_branch                            = "main"
	build_pack                            = "dockercompose"
	ports_exposes                         = "3000"
	docker_compose_custom_start_command   = "docker compose up && docker compose logs"
	docker_compose_custom_build_command   = "docker compose build"
	%s
`, tt.extraAttrs))

			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
				Steps: []resource.TestStep{
					{
						Config: config,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr(tt.resourceType+".test", "docker_compose_custom_start_command", "docker compose up -d"),
							resource.TestCheckResourceAttr(tt.resourceType+".test", "docker_compose_custom_build_command", "docker compose build"),
						),
					},
					{
						Config: updatedConfig,
						Check: resource.TestCheckResourceAttr(
							tt.resourceType+".test",
							"docker_compose_custom_start_command",
							"docker compose up && docker compose logs",
						),
					},
				},
			})
		})
	}
}

func TestGitApplicationResource_InvalidDockerComposeCustomCommand(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.NotFoundHandler()))
	defer srv.Close()

	tooLong := "docker compose " + strings.Repeat("x", 1000)
	attrs := func(cmd string) string {
		return fmt.Sprintf(`
	name                                  = "compose-command-invalid"
	project_uuid                          = "aaaa0001-0001-4000-8000-000000000001"
	server_uuid                           = "bbbb0001-0001-4000-8000-000000000001"
	git_repository                        = "https://github.com/example/repo"
	git_branch                            = "main"
	build_pack                            = "dockercompose"
	ports_exposes                         = "3000"
	docker_compose_custom_start_command   = %q
`, cmd)
	}

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:      acctest.TestResourceConfig(srv.URL, "coolify_application", "test", attrs("docker compose up; echo pwned")),
				ExpectError: regexp.MustCompile(`Unsafe Command Detected|shell-safe command`),
			},
			{
				Config:      acctest.TestResourceConfig(srv.URL, "coolify_application", "test", attrs(tooLong)),
				ExpectError: regexp.MustCompile(`string length must be at most`),
			},
		},
	})
}

func TestGitApplicationResource_DockerfileTargetBuildPostCreate(t *testing.T) {
	t.Parallel()

	app := client.Application{
		UUID:            "git-dockerfile-target",
		Name:            "git-dockerfile-target",
		GitRepository:   "https://github.com/example/repo",
		GitBranch:       "main",
		BuildPack:       "dockerfile",
		PortsExposes:    "80",
		ProjectUUID:     "aaaa0001-0001-4000-8000-000000000001",
		ServerUUID:      "bbbb0001-0001-4000-8000-000000000001",
		EnvironmentName: "production",
	}
	var patchBody map[string]any
	var mu sync.Mutex
	var deleted atomic.Bool

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/public", func(w http.ResponseWriter, r *http.Request) {
		body, ok := decodeRequestBodyMap(t, w, r)
		if !ok {
			return
		}
		if _, has := body["dockerfile_target_build"]; has {
			t.Error("POST /api/v1/applications/public must not send dockerfile_target_build")
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"uuid": app.UUID})
	})
	mux.HandleFunc("PATCH /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		body, ok := decodeRequestBodyMap(t, w, r)
		if !ok {
			return
		}
		mu.Lock()
		patchBody = body
		if v, ok := body["dockerfile_target_build"].(string); ok {
			app.DockerfileTargetBuild = v
		}
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "updated"})
	})
	mux.HandleFunc("GET /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != app.UUID || deleted.Load() {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(app)
	})
	mux.HandleFunc("DELETE /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		deleted.Store(true)
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_application", "test", `
	name                    = "git-dockerfile-target"
	project_uuid            = "aaaa0001-0001-4000-8000-000000000001"
	server_uuid             = "bbbb0001-0001-4000-8000-000000000001"
	git_repository          = "https://github.com/example/repo"
	git_branch              = "main"
	build_pack              = "dockerfile"
	ports_exposes           = "80"
	dockerfile_target_build = "production"
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_application.test", "dockerfile_target_build", "production"),
					func(_ *terraform.State) error {
						mu.Lock()
						defer mu.Unlock()
						got, ok := patchBody["dockerfile_target_build"]
						if !ok {
							return fmt.Errorf("post-create PATCH missing dockerfile_target_build: %v", patchBody)
						}
						if got != "production" {
							return fmt.Errorf("dockerfile_target_build = %v, want production", got)
						}
						return nil
					},
				),
			},
		},
	})
}

func TestGitApplicationResource_InvalidDockerfileTargetBuild(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.NotFoundHandler()))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_application", "test", `
	name                    = "git-dockerfile-target-invalid"
	project_uuid            = "aaaa0001-0001-4000-8000-000000000001"
	server_uuid             = "bbbb0001-0001-4000-8000-000000000001"
	git_repository          = "https://github.com/example/repo"
	git_branch              = "main"
	build_pack              = "dockerfile"
	ports_exposes           = "80"
	dockerfile_target_build = "-prod"
`),
				ExpectError: regexp.MustCompile(`Invalid Docker Target|multi-stage target`),
			},
		},
	})
}
