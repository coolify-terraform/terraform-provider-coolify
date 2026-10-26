//go:build !ci_app_a

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
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestDockerfileApplicationResource_InPlaceUpdateOnCoolify45(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		version string
		inPlace bool
	}{
		{version: "4.5.0", inPlace: true},
		{version: "4.4.2", inPlace: false},
		{version: "", inPlace: true},
	} {
		t.Run(tc.version, func(t *testing.T) {
			label := tc.version
			if label == "" {
				label = "empty"
			}
			app := client.Application{
				UUID:            "dockerfile-v45-uuid",
				Name:            "docker-app",
				PortsExposes:    "80",
				ProjectUUID:     "aaaa0001-0001-4000-8000-000000000001",
				ServerUUID:      "bbbb0001-0001-4000-8000-000000000001",
				EnvironmentName: "production",
			}
			var mu sync.Mutex
			deleted := false
			var patchHadDockerfile []bool
			mux := http.NewServeMux()
			mux.HandleFunc("POST /api/v1/applications/dockerfile", func(w http.ResponseWriter, r *http.Request) {
				body, ok := decodeRequestBodyMap(t, w, r)
				if !ok {
					return
				}
				mu.Lock()
				deleted = false
				mu.Unlock()
				if _, ok := body["dockerfile"]; !ok {
					t.Errorf("create POST omitted dockerfile")
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				_ = json.NewEncoder(w).Encode(map[string]string{"uuid": app.UUID})
			})
			mux.HandleFunc("GET /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
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
				_, has := body["dockerfile"]
				mu.Lock()
				patchHadDockerfile = append(patchHadDockerfile, has)
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

			base := `
				project_uuid        = "aaaa0001-0001-4000-8000-000000000001"
				server_uuid         = "bbbb0001-0001-4000-8000-000000000001"
				ports_exposes       = "80"
				dockerfile_location = "FROM nginx:1.25"
			`
			updated := `
				project_uuid        = "aaaa0001-0001-4000-8000-000000000001"
				server_uuid         = "bbbb0001-0001-4000-8000-000000000001"
				ports_exposes       = "80"
				dockerfile_location = "FROM nginx:1.26"
			`
			action := plancheck.ResourceActionDestroyBeforeCreate
			if tc.inPlace {
				action = plancheck.ResourceActionUpdate
			}
			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
				Steps: []resource.TestStep{
					{Config: testDockerfileResourceConfig(srv.URL, base)},
					{
						Config: testDockerfileResourceConfig(srv.URL, updated),
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply: []plancheck.PlanCheck{
								plancheck.ExpectResourceAction("coolify_application_dockerfile.test", action),
							},
						},
						Check: func(*terraform.State) error {
							mu.Lock()
							defer mu.Unlock()
							saw := false
							for _, has := range patchHadDockerfile {
								if has {
									saw = true
								}
							}
							if saw != tc.inPlace {
								return fmt.Errorf("Coolify %s dockerfile PATCH=%v, want %v", label, saw, tc.inPlace)
							}
							return nil
						},
					},
				},
			})
		})
	}
}
