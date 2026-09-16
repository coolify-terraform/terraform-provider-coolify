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
)

func TestGitApplicationResources_DockerComposeLocation(t *testing.T) {
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
				UUID:                  "compose-location-test",
				Name:                  "compose-location-test",
				GitRepository:         "https://github.com/example/repo",
				GitBranch:             "main",
				BuildPack:             "dockercompose",
				PortsExposes:          "3000",
				ProjectUUID:           "aaaa0001-0001-4000-8000-000000000001",
				ServerUUID:            "bbbb0001-0001-4000-8000-000000000001",
				EnvironmentName:       "production",
				DockerComposeLocation: "/compose.yaml",
			}

			var appMu sync.Mutex
			var deleted atomic.Bool
			mux := http.NewServeMux()

			mux.HandleFunc("POST "+tt.createPath, func(w http.ResponseWriter, r *http.Request) {
				body, ok := decodeRequestBodyMap(t, w, r)
				if !ok {
					return
				}

				got, ok := body["docker_compose_location"].(string)
				if !ok {
					t.Error("create request missing docker_compose_location")
					http.Error(w, `{"error":"missing docker_compose_location"}`, http.StatusUnprocessableEntity)
					return
				}

				if got != "/compose.yaml" {
					t.Errorf("docker_compose_location = %q, want %q", got, "/compose.yaml")
					http.Error(w, `{"error":"unexpected docker_compose_location"}`, http.StatusUnprocessableEntity)
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

				got, ok := body["docker_compose_location"].(string)
				if !ok {
					t.Error("update request missing docker_compose_location")
					http.Error(w, `{"error":"missing docker_compose_location"}`, http.StatusUnprocessableEntity)
					return
				}

				if got != "/config/compose.yaml" {
					t.Errorf("updated docker_compose_location = %q, want %q", got, "/config/compose.yaml")
					http.Error(w, `{"error":"unexpected docker_compose_location"}`, http.StatusUnprocessableEntity)
					return
				}

				appMu.Lock()
				app.DockerComposeLocation = got
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
	name                    = "compose-location-test"
	project_uuid            = "aaaa0001-0001-4000-8000-000000000001"
	server_uuid             = "bbbb0001-0001-4000-8000-000000000001"
	git_repository          = "https://github.com/example/repo"
	git_branch              = "main"
	build_pack              = "dockercompose"
	ports_exposes           = "3000"
	docker_compose_location = "/compose.yaml"
	%s
`, tt.extraAttrs))

			updatedConfig := acctest.TestResourceConfig(srv.URL, tt.resourceType, "test", fmt.Sprintf(`
	name                    = "compose-location-test"
	project_uuid            = "aaaa0001-0001-4000-8000-000000000001"
	server_uuid             = "bbbb0001-0001-4000-8000-000000000001"
	git_repository          = "https://github.com/example/repo"
	git_branch              = "main"
	build_pack              = "dockercompose"
	ports_exposes           = "3000"
	docker_compose_location = "/config/compose.yaml"
	%s
`, tt.extraAttrs))

			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
				Steps: []resource.TestStep{
					{
						Config: config,
						Check: resource.TestCheckResourceAttr(
							tt.resourceType+".test",
							"docker_compose_location",
							"/compose.yaml",
						),
					},
					{
						Config: updatedConfig,
						Check: resource.TestCheckResourceAttr(
							tt.resourceType+".test",
							"docker_compose_location",
							"/config/compose.yaml",
						),
					},
				},
			})
		})
	}
}

func TestGitApplicationResource_InvalidDockerComposeLocation(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.NotFoundHandler()))
	defer srv.Close()

	tooLong := "/" + strings.Repeat("a", 255)
	attrs := func(location string) string {
		return fmt.Sprintf(`
	name                    = "compose-location-invalid"
	project_uuid            = "aaaa0001-0001-4000-8000-000000000001"
	server_uuid             = "bbbb0001-0001-4000-8000-000000000001"
	git_repository          = "https://github.com/example/repo"
	git_branch              = "main"
	build_pack              = "dockercompose"
	ports_exposes           = "3000"
	docker_compose_location = %q
`, location)
	}

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:      acctest.TestResourceConfig(srv.URL, "coolify_application", "test", attrs("compose.yaml")),
				ExpectError: regexp.MustCompile(`must start with ` + "`/`"),
			},
			{
				Config:      acctest.TestResourceConfig(srv.URL, "coolify_application", "test", attrs(tooLong)),
				ExpectError: regexp.MustCompile(`string length must be at most 255`),
			},
		},
	})
}

func TestGitApplicationResource_DockerComposeLocationOmitKeepsState(t *testing.T) {
	t.Parallel()

	app := client.Application{
		UUID:                  "compose-location-omit",
		Name:                  "compose-location-omit",
		GitRepository:         "https://github.com/example/repo",
		GitBranch:             "main",
		BuildPack:             "dockercompose",
		PortsExposes:          "3000",
		ProjectUUID:           "aaaa0001-0001-4000-8000-000000000001",
		ServerUUID:            "bbbb0001-0001-4000-8000-000000000001",
		EnvironmentName:       "production",
		DockerComposeLocation: "/compose.yaml",
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
		t.Error("omit-after-set must not PATCH docker_compose_location")
		http.Error(w, `{"error":"unexpected patch"}`, http.StatusUnprocessableEntity)
	})
	mux.HandleFunc("DELETE /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		deleted.Store(true)
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	base := `
	name           = "compose-location-omit"
	project_uuid   = "aaaa0001-0001-4000-8000-000000000001"
	server_uuid    = "bbbb0001-0001-4000-8000-000000000001"
	git_repository = "https://github.com/example/repo"
	git_branch     = "main"
	build_pack     = "dockercompose"
	ports_exposes  = "3000"
`
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_application", "test", base+`
	docker_compose_location = "/compose.yaml"
`),
				Check: resource.TestCheckResourceAttr("coolify_application.test", "docker_compose_location", "/compose.yaml"),
			},
			{
				Config:             acctest.TestResourceConfig(srv.URL, "coolify_application", "test", base),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}
