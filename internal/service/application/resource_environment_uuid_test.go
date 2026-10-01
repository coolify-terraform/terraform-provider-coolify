//go:build !ci_app_b

package application_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestApplicationResource_EnvironmentUUIDCreateAndUpdate(t *testing.T) {
	t.Parallel()
	const appUUID = "app-env-uuid-0001"
	const envUUID = "eeee0001-0001-4000-8000-000000000001"
	var mu sync.Mutex
	var posts []map[string]any
	var patches []map[string]any
	description := ""
	deleted := false

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/public", func(w http.ResponseWriter, r *http.Request) {
		body, ok := decodeRequestBodyMap(t, w, r)
		if !ok {
			return
		}
		mu.Lock()
		posts = append(posts, body)
		deleted = false
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"uuid": appUUID})
	})
	mux.HandleFunc("GET /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if deleted {
			http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
			return
		}
		payload := map[string]any{
			"uuid":             appUUID,
			"name":             "uuid-app",
			"git_repository":   "https://github.com/example/repo",
			"git_branch":       "main",
			"build_pack":       "nixpacks",
			"ports_exposes":    "3000",
			"project_uuid":     "aaaa0001-0001-4000-8000-000000000001",
			"server_uuid":      "bbbb0001-0001-4000-8000-000000000001",
			"environment_name": "staging",
		}
		if description != "" {
			payload["description"] = description
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(payload)
	})
	mux.HandleFunc("PATCH /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		_ = json.Unmarshal(raw, &body)
		mu.Lock()
		patches = append(patches, body)
		if desc, ok := body["description"].(string); ok {
			description = desc
		}
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"uuid": appUUID})
	})
	mux.HandleFunc("DELETE /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		deleted = true
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	base := acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_application" "test" {
  project_uuid     = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid      = "bbbb0001-0001-4000-8000-000000000001"
  environment_uuid = "` + envUUID + `"
  git_repository   = "https://github.com/example/repo"
  git_branch       = "main"
  build_pack       = "nixpacks"
  ports_exposes    = "3000"
`
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: base + "\n}\n",
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_application.test", "environment_uuid", envUUID),
					resource.TestCheckNoResourceAttr("coolify_application.test", "environment_name"),
				),
			},
			{
				Config: base + "\n  description = \"renamed\"\n}\n",
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("coolify_application.test", plancheck.ResourceActionUpdate),
					},
				},
			},
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_application" "test" {
  project_uuid     = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid      = "bbbb0001-0001-4000-8000-000000000001"
  environment_uuid = "eeee0002-0002-4000-8000-000000000002"
  git_repository   = "https://github.com/example/repo"
  git_branch       = "main"
  build_pack       = "nixpacks"
  ports_exposes    = "3000"
  description      = "renamed"
}
`,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("coolify_application.test", plancheck.ResourceActionReplace),
					},
				},
			},
		},
	})

	mu.Lock()
	defer mu.Unlock()
	if len(posts) < 1 {
		t.Fatal("expected a create POST")
	}
	if _, ok := posts[0]["environment_name"]; ok {
		t.Fatalf("create sent environment_name: %#v", posts[0]["environment_name"])
	}
	if posts[0]["environment_uuid"] != envUUID {
		t.Fatalf("environment_uuid = %#v", posts[0]["environment_uuid"])
	}
	if len(patches) == 0 {
		t.Fatal("expected an update PATCH")
	}
	if _, ok := patches[0]["environment_uuid"]; ok {
		t.Fatalf("update sent environment_uuid: %#v", patches[0]["environment_uuid"])
	}
}

func TestApplicationResource_EnvironmentNameAndUUIDConflict(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.NotFoundHandler()))
	defer srv.Close()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_application" "test" {
  project_uuid     = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid      = "bbbb0001-0001-4000-8000-000000000001"
  environment_name = "production"
  environment_uuid = "eeee0001-0001-4000-8000-000000000001"
  git_repository   = "https://github.com/example/repo"
  git_branch       = "main"
  build_pack       = "nixpacks"
  ports_exposes    = "3000"
}
`,
			ExpectError: regexp.MustCompile(`Conflicting environment identity`),
		}},
	})
}

func TestApplicationResource_ImportThenEnvironmentUUIDDoesNotReplace(t *testing.T) {
	t.Parallel()
	const appUUID = "aaaa0003-0003-4000-8000-000000000003"
	const envUUID = "eeee0001-0001-4000-8000-000000000001"
	const projectUUID = "aaaa0001-0001-4000-8000-000000000001"
	const serverUUID = "bbbb0001-0001-4000-8000-000000000001"
	var mu sync.Mutex
	deleted := false

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/public", func(w http.ResponseWriter, r *http.Request) {
		if _, ok := decodeRequestBodyMap(t, w, r); !ok {
			return
		}
		mu.Lock()
		deleted = false
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"uuid": appUUID})
	})
	mux.HandleFunc("GET /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if deleted {
			http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"uuid":             appUUID,
			"name":             "imported-app",
			"git_repository":   "https://github.com/example/repo",
			"git_branch":       "main",
			"build_pack":       "nixpacks",
			"ports_exposes":    "3000",
			"project_uuid":     projectUUID,
			"server_uuid":      serverUUID,
			"environment_name": "production",
		})
	})
	mux.HandleFunc("PATCH /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"uuid": appUUID})
	})
	mux.HandleFunc("DELETE /api/v1/applications/{uuid}", func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		deleted = true
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	named := testApplicationResourceConfig(srv.URL, `
		project_uuid     = "`+projectUUID+`"
		server_uuid      = "`+serverUUID+`"
		environment_name = "production"
		git_repository   = "https://github.com/example/repo"
		git_branch       = "main"
		build_pack       = "nixpacks"
		ports_exposes    = "3000"
	`)
	byUUID := testApplicationResourceConfig(srv.URL, `
		project_uuid     = "`+projectUUID+`"
		server_uuid      = "`+serverUUID+`"
		environment_uuid = "`+envUUID+`"
		git_repository   = "https://github.com/example/repo"
		git_branch       = "main"
		build_pack       = "nixpacks"
		ports_exposes    = "3000"
	`)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{Config: named},
			{
				ResourceName:  "coolify_application.test",
				ImportState:   true,
				ImportStateId: appUUID,
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					if len(states) != 1 {
						return fmt.Errorf("imported states = %d", len(states))
					}
					if states[0].Attributes["environment_name"] == "" {
						return fmt.Errorf("simple import stored an empty environment_name")
					}
					if states[0].Attributes["environment_uuid"] != "" {
						return fmt.Errorf("simple import stored environment_uuid %q", states[0].Attributes["environment_uuid"])
					}
					return nil
				},
			},
			{
				Config: byUUID,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("coolify_application.test", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_application.test", "environment_uuid", envUUID),
					resource.TestCheckNoResourceAttr("coolify_application.test", "environment_name"),
				),
			},
			{
				Config: byUUID,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("coolify_application.test", plancheck.ResourceActionNoop),
					},
				},
			},
		},
	})
}
