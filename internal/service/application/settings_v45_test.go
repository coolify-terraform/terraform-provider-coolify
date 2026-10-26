//go:build !ci_app_b

package application_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestApplicationResource_Coolify45Settings(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		version string
		send    bool
	}{
		{version: "4.5.0", send: true},
		{version: "4.4.2", send: false},
		{version: "", send: true},
	} {
		t.Run(tc.version, func(t *testing.T) {
			label := tc.version
			if label == "" {
				label = "empty"
			}
			app := client.Application{
				UUID:            "v45-app-uuid",
				Name:            "my-app",
				GitRepository:   "https://github.com/example/repo",
				GitBranch:       "main",
				BuildPack:       "nixpacks",
				PortsExposes:    "3000",
				ProjectUUID:     "aaaa0002-0002-4000-8000-000000000002",
				ServerUUID:      "bbbb0002-0002-4000-8000-000000000002",
				EnvironmentName: "production",
			}
			var mu sync.Mutex
			deleted := false
			var patches []map[string]any
			mux := http.NewServeMux()
			mux.HandleFunc("POST /api/v1/applications/public", func(w http.ResponseWriter, r *http.Request) {
				if _, ok := decodeRequestBodyMap(t, w, r); !ok {
					return
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				_ = json.NewEncoder(w).Encode(map[string]string{"uuid": app.UUID})
			})
			mux.HandleFunc("GET /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
				if r.PathValue("uuid") != app.UUID {
					http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
					return
				}
				mu.Lock()
				defer mu.Unlock()
				if deleted {
					http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(app)
			})
			mux.HandleFunc("PATCH /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
				body, ok := decodeRequestBodyMap(t, w, r)
				if !ok {
					return
				}
				mu.Lock()
				patches = append(patches, body)
				mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]string{"message": "updated"})
			})
			mux.HandleFunc("DELETE /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				deleted = true
				mu.Unlock()
				w.WriteHeader(http.StatusNoContent)
			})
			srv := httptest.NewServer(acctest.WithVersionEndpointVersion(mux, tc.version))
			defer srv.Close()

			attrs := `
				project_uuid                        = "aaaa0002-0002-4000-8000-000000000002"
				server_uuid                         = "bbbb0002-0002-4000-8000-000000000002"
				git_repository                      = "https://github.com/example/repo"
				build_pack                          = "nixpacks"
				ports_exposes                       = "3000"
				http_basic_auth_hash_algorithm      = "argon2id"
				http_basic_auth_argon2id_memory_cost = 65536
				http_basic_auth_argon2id_time_cost  = "2"
				http_basic_auth_bcrypt_cost         = "10"
				is_permanent_redirect_enabled       = true
				redact_all_env_values_in_logs       = false
			`
			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
				Steps: []resource.TestStep{
					{
						Config: testApplicationResourceConfig(srv.URL, attrs),
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("coolify_application.test", "http_basic_auth_hash_algorithm", "argon2id"),
							resource.TestCheckResourceAttr("coolify_application.test", "is_permanent_redirect_enabled", "true"),
							resource.TestCheckResourceAttr("coolify_application.test", "redact_all_env_values_in_logs", "false"),
							func(*terraform.State) error {
								return assertV45Keys(label, patches, tc.send, 65536.0)
							},
						),
					},
					{
						Config: testApplicationResourceConfig(srv.URL, `
				project_uuid                        = "aaaa0002-0002-4000-8000-000000000002"
				server_uuid                         = "bbbb0002-0002-4000-8000-000000000002"
				git_repository                      = "https://github.com/example/repo"
				build_pack                          = "nixpacks"
				ports_exposes                       = "3000"
				http_basic_auth_hash_algorithm      = "argon2id"
				http_basic_auth_argon2id_memory_cost = 32768
				http_basic_auth_argon2id_time_cost  = "2"
				http_basic_auth_bcrypt_cost         = "10"
				is_permanent_redirect_enabled       = true
				redact_all_env_values_in_logs       = false
			`),
						Check: func(*terraform.State) error {
							mu.Lock()
							defer mu.Unlock()
							if len(patches) == 0 {
								return fmt.Errorf("Coolify %s sent no update PATCH", label)
							}
							last := patches[len(patches)-1]
							_, ok := last["http_basic_auth_argon2id_memory_cost"]
							if ok != tc.send {
								return fmt.Errorf("Coolify %s update memory cost present=%v, want %v", label, ok, tc.send)
							}
							if tc.send && last["http_basic_auth_argon2id_memory_cost"] != 32768.0 {
								return fmt.Errorf("update memory cost = %v", last["http_basic_auth_argon2id_memory_cost"])
							}
							for key := range last {
								if !tc.send && (key == "http_basic_auth_hash_algorithm" || key == "redact_all_env_values_in_logs") {
									return fmt.Errorf("Coolify %s update sent %s", label, key)
								}
							}
							return nil
						},
					},
				},
			})
		})
	}
}

func assertV45Keys(label string, patches []map[string]any, want bool, memory float64) error {
	keys := []string{
		"http_basic_auth_hash_algorithm",
		"http_basic_auth_argon2id_memory_cost",
		"http_basic_auth_argon2id_time_cost",
		"http_basic_auth_bcrypt_cost",
		"is_permanent_redirect_enabled",
		"redact_all_env_values_in_logs",
	}
	seen := map[string]any{}
	for _, body := range patches {
		for key, value := range body {
			seen[key] = value
		}
	}
	for _, key := range keys {
		_, ok := seen[key]
		if ok != want {
			return fmt.Errorf("Coolify %s key %s present=%v, want %v", label, key, ok, want)
		}
	}
	if !want {
		return nil
	}
	if seen["http_basic_auth_argon2id_memory_cost"] != memory {
		return fmt.Errorf("memory cost = %v (%T), want %v", seen["http_basic_auth_argon2id_memory_cost"], seen["http_basic_auth_argon2id_memory_cost"], memory)
	}
	if seen["redact_all_env_values_in_logs"] != false {
		return fmt.Errorf("redact = %v, want false", seen["redact_all_env_values_in_logs"])
	}
	return nil
}
