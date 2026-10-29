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
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestApplicationResource_RollingUpdateDrainSeconds(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		version string
		send    bool
	}{
		{version: "4.5.1", send: true},
		{version: "4.5.0", send: false},
		{version: "4.4.2", send: false},
		{version: "", send: true},
	} {
		t.Run(tc.version, func(t *testing.T) {
			label := tc.version
			if label == "" {
				label = "empty"
			}
			const appUUID = "v451-app-uuid"
			var mu sync.Mutex
			var patches []map[string]any
			deleted := false
			mux := http.NewServeMux()
			mux.HandleFunc("POST /api/v1/applications/public", func(w http.ResponseWriter, r *http.Request) {
				if _, ok := decodeRequestBodyMap(t, w, r); !ok {
					return
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				_ = json.NewEncoder(w).Encode(map[string]string{"uuid": appUUID})
			})
			mux.HandleFunc("GET /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				gone := deleted
				mu.Unlock()
				if gone {
					http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				body := map[string]any{
					"uuid": appUUID, "name": "my-app",
					"git_repository": "https://github.com/example/repo",
					"git_branch":     "main", "build_pack": "nixpacks", "ports_exposes": "3000",
				}
				if tc.send {
					body["rolling_update_drain_seconds"] = 12
				}
				_ = json.NewEncoder(w).Encode(body)
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
			mux.HandleFunc("DELETE /api/v1/applications/{uuid}", func(w http.ResponseWriter, _ *http.Request) {
				mu.Lock()
				deleted = true
				mu.Unlock()
				w.WriteHeader(http.StatusNoContent)
			})
			srv := httptest.NewServer(acctest.WithVersionEndpointVersion(mux, tc.version))
			defer srv.Close()

			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
				Steps: []resource.TestStep{
					{
						Config: testApplicationResourceConfig(srv.URL, `
				project_uuid                   = "aaaa0002-0002-4000-8000-000000000002"
				server_uuid                    = "bbbb0002-0002-4000-8000-000000000002"
				git_repository                 = "https://github.com/example/repo"
				build_pack                     = "nixpacks"
				ports_exposes                  = "3000"
				rolling_update_drain_seconds   = 12
			`),
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("coolify_application.test", "rolling_update_drain_seconds", "12"),
							func(*terraform.State) error {
								mu.Lock()
								defer mu.Unlock()
								for _, body := range patches {
									_, ok := body["rolling_update_drain_seconds"]
									if ok != tc.send {
										return fmt.Errorf("Coolify %s drain seconds present=%v, want %v", label, ok, tc.send)
									}
								}
								if len(patches) == 0 {
									return fmt.Errorf("Coolify %s sent no settings PATCH", label)
								}
								return nil
							},
						),
					},
				},
			})
		})
	}
}
