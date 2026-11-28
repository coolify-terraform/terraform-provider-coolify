package sharedenv_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestSharedEnvResource_TeamCRUD(t *testing.T) {
	t.Parallel()
	store := map[string]map[string]any{}
	var mu sync.Mutex
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/team/envs":
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["key"] != "GLOBAL_FLAG" {
				t.Errorf("expected key GLOBAL_FLAG got %v", body)
			}
			body["id"] = float64(3)
			store["3"] = body
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(body)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/team/envs":
			out := []any{}
			for _, v := range store {
				out = append(out, v)
			}
			_ = json.NewEncoder(w).Encode(out)
		case r.Method == http.MethodPatch:
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			for _, v := range store {
				for k, val := range body {
					v[k] = val
				}
				_ = json.NewEncoder(w).Encode(v)
				return
			}
		case r.Method == http.MethodDelete:
			store = map[string]map[string]any{}
			w.WriteHeader(http.StatusOK)
		default:
			http.Error(w, r.URL.Path, 404)
		}
	})))
	defer srv.Close()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_shared_environment_variable" "test" {
  scope = "team"
  key   = "GLOBAL_FLAG"
  value = "on"
}`,
				Check: resource.TestCheckResourceAttr("coolify_shared_environment_variable.test", "key", "GLOBAL_FLAG"),
			},
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_shared_environment_variable" "test" {
  scope = "team"
  key   = "GLOBAL_FLAG"
  value = "off"
}`,
				Check: resource.TestCheckResourceAttr("coolify_shared_environment_variable.test", "value", "off"),
			},
		},
	})
}

func TestSharedEnvResource_Disappears(t *testing.T) {
	t.Parallel()
	store := map[string]map[string]any{}
	var mu sync.Mutex
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/team/envs":
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			body["id"] = float64(3)
			store["3"] = body
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(body)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/team/envs":
			out := []any{}
			for _, v := range store {
				out = append(out, v)
			}
			_ = json.NewEncoder(w).Encode(out)
		case r.Method == http.MethodDelete:
			store = map[string]map[string]any{}
			w.WriteHeader(http.StatusOK)
		default:
			http.Error(w, r.URL.Path, 404)
		}
	})))
	defer srv.Close()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_shared_environment_variable" "test" {
  scope = "team"
  key   = "GLOBAL_FLAG"
  value = "on"
}`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("coolify_shared_environment_variable.test", "env_id", "3"),
				acctest.CheckPathDisappears(srv.URL, "/api/v1/team/envs/3"),
			),
			ExpectNonEmptyPlan: true,
		}},
	})
}

func TestSharedEnvResource_ScopeCRUD(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		path   string
		create string
		update string
	}{
		{
			name: "project",
			path: "/api/v1/projects/proj-uuid/envs",
			create: `
resource "coolify_shared_environment_variable" "test" {
  scope        = "project"
  project_uuid = "proj-uuid"
  key          = "PROJ_FLAG"
  value        = "on"
}`,
			update: `
resource "coolify_shared_environment_variable" "test" {
  scope        = "project"
  project_uuid = "proj-uuid"
  key          = "PROJ_FLAG"
  value        = "off"
}`,
		},
		{
			name: "environment",
			path: "/api/v1/projects/proj-uuid/environments/production/envs",
			create: `
resource "coolify_shared_environment_variable" "test" {
  scope        = "environment"
  project_uuid = "proj-uuid"
  environment  = "production"
  key          = "ENV_FLAG"
  value        = "on"
}`,
			update: `
resource "coolify_shared_environment_variable" "test" {
  scope        = "environment"
  project_uuid = "proj-uuid"
  environment  = "production"
  key          = "ENV_FLAG"
  value        = "off"
}`,
		},
		{
			name: "server",
			path: "/api/v1/servers/srv-uuid/envs",
			create: `
resource "coolify_shared_environment_variable" "test" {
  scope       = "server"
  server_uuid = "srv-uuid"
  key         = "SRV_FLAG"
  value       = "on"
}`,
			update: `
resource "coolify_shared_environment_variable" "test" {
  scope       = "server"
  server_uuid = "srv-uuid"
  key         = "SRV_FLAG"
  value       = "off"
}`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			store := map[string]map[string]any{}
			var mu sync.Mutex
			var seenPaths []string
			srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				seenPaths = append(seenPaths, r.Method+" "+r.URL.Path)
				switch {
				case r.Method == http.MethodPost && r.URL.Path == tc.path:
					var body map[string]any
					_ = json.NewDecoder(r.Body).Decode(&body)
					if body["key"] == nil || body["key"] == "" {
						t.Errorf("expected key in create body, got %v", body)
					}
					if body["value"] != "on" {
						t.Errorf("expected value on, got %v", body)
					}
					body["id"] = float64(9)
					store["9"] = body
					w.WriteHeader(http.StatusCreated)
					_ = json.NewEncoder(w).Encode(body)
				case r.Method == http.MethodGet && r.URL.Path == tc.path:
					out := []any{}
					for _, v := range store {
						out = append(out, v)
					}
					_ = json.NewEncoder(w).Encode(out)
				case r.Method == http.MethodPatch && strings.HasPrefix(r.URL.Path, tc.path+"/"):
					var body map[string]any
					_ = json.NewDecoder(r.Body).Decode(&body)
					for _, v := range store {
						for k, val := range body {
							v[k] = val
						}
						_ = json.NewEncoder(w).Encode(v)
						return
					}
				case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, tc.path+"/"):
					store = map[string]map[string]any{}
					w.WriteHeader(http.StatusOK)
				default:
					http.Error(w, r.URL.Path, 404)
				}
			})))
			defer srv.Close()
			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
				Steps: []resource.TestStep{
					{
						Config: acctest.ProviderBlockForURL(srv.URL) + tc.create,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("coolify_shared_environment_variable.test", "scope", tc.name),
							resource.TestCheckResourceAttr("coolify_shared_environment_variable.test", "env_id", "9"),
						),
					},
					{
						Config: acctest.ProviderBlockForURL(srv.URL) + tc.update,
						Check:  resource.TestCheckResourceAttr("coolify_shared_environment_variable.test", "value", "off"),
					},
				},
			})
			mu.Lock()
			defer mu.Unlock()
			foundCreate := false
			for _, p := range seenPaths {
				if p == "POST "+tc.path {
					foundCreate = true
					break
				}
			}
			if !foundCreate {
				t.Errorf("never POSTed %s; saw %v", tc.path, seenPaths)
			}
		})
	}
}

func TestSharedEnvResource_RejectsHyphenKey(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.NewServeMux()))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_shared_environment_variable" "test" {
  scope = "team"
  key   = "GLOBAL-FLAG"
  value = "on"
}`,
			ExpectError: regexp.MustCompile(`hyphens are not allowed`),
		}},
	})
}

func TestSharedEnvResource_CreateAPIError(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/team/envs", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"message":"validation failed"}`, http.StatusUnprocessableEntity)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_shared_environment_variable" "test" {
  scope = "team"
  key   = "GLOBAL_FLAG"
  value = "on"
}`,
			ExpectError: regexp.MustCompile(`Error creating shared environment variable`),
		}},
	})
}

func TestSharedEnvResource_CommentOmittedDoesNotClear(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	var patches int
	var lastPatch map[string]any
	store := map[string]any{
		"id":      float64(3),
		"key":     "GLOBAL_FLAG",
		"value":   "on",
		"comment": "ui note",
	}
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/team/envs":
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if _, ok := body["comment"]; ok {
				t.Errorf("create omitted comment, body has %#v", body["comment"])
			}
			body["id"] = float64(3)
			body["comment"] = "ui note"
			store = body
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"id": float64(3)})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/team/envs":
			_ = json.NewEncoder(w).Encode([]any{store})
		case r.Method == http.MethodPatch:
			patches++
			_ = json.NewDecoder(r.Body).Decode(&lastPatch)
			for k, val := range lastPatch {
				store[k] = val
			}
			_ = json.NewEncoder(w).Encode(store)
		default:
			http.Error(w, r.URL.Path, http.StatusNotFound)
		}
	})))
	defer srv.Close()

	base := acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_shared_environment_variable" "test" {
  scope = "team"
  key   = "GLOBAL_FLAG"
  value = "on"
}
`
	cleared := acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_shared_environment_variable" "test" {
  scope   = "team"
  key     = "GLOBAL_FLAG"
  value   = "on"
  comment = ""
}
`
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: base,
				Check: func(_ *terraform.State) error {
					mu.Lock()
					defer mu.Unlock()
					if patches != 0 {
						return fmt.Errorf("create patched %d times", patches)
					}
					if store["comment"] != "ui note" {
						return fmt.Errorf("stored comment = %#v", store["comment"])
					}
					return nil
				},
			},
			{
				Config: base,
				Check: func(_ *terraform.State) error {
					mu.Lock()
					defer mu.Unlock()
					if patches != 0 {
						return fmt.Errorf("omitted comment patched %d times, last %#v", patches, lastPatch)
					}
					return nil
				},
			},
			{
				Config: cleared,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_shared_environment_variable.test", "comment", ""),
					func(_ *terraform.State) error {
						mu.Lock()
						defer mu.Unlock()
						if patches != 1 {
							return fmt.Errorf("clear patched %d times", patches)
						}
						if lastPatch["comment"] != "" {
							return fmt.Errorf("clear comment = %#v", lastPatch["comment"])
						}
						return nil
					},
				),
			},
		},
	})
}
