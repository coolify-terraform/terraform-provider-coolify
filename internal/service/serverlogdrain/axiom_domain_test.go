package serverlogdrain_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestServerLogDrainResource_AxiomDomain(t *testing.T) {
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
			const serverUUID = "aaaa0001-0001-4000-8000-000000000001"
			var mu sync.Mutex
			saw := false
			srv := httptest.NewServer(acctest.WithVersionEndpointVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				if !strings.HasSuffix(r.URL.Path, "/log-drains") {
					http.Error(w, r.URL.Path, http.StatusNotFound)
					return
				}
				switch r.Method {
				case http.MethodPatch:
					var body map[string]any
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Errorf("decode log drain patch: %v", err)
						return
					}
					if _, ok := body["logdrain_axiom_domain"]; ok {
						saw = true
					}
					if tc.send {
						body["logdrain_axiom_domain"] = "eu-central-1.aws.edge.axiom.co"
					}
					_ = json.NewEncoder(w).Encode(body)
				case http.MethodGet:
					out := map[string]any{"is_logdrain_axiom_enabled": false}
					if tc.send {
						out["logdrain_axiom_domain"] = "eu-central-1.aws.edge.axiom.co"
					}
					_ = json.NewEncoder(w).Encode(out)
				default:
					http.Error(w, `{"message":"method"}`, http.StatusMethodNotAllowed)
				}
			}), tc.version))
			defer srv.Close()

			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
				Steps: []resource.TestStep{
					{
						Config: acctest.TestResourceConfig(srv.URL, "coolify_server_log_drain", "test", `
  server_uuid            = "`+serverUUID+`"
  logdrain_axiom_domain  = "eu-central-1.aws.edge.axiom.co"
`),
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("coolify_server_log_drain.test", "logdrain_axiom_domain", "eu-central-1.aws.edge.axiom.co"),
							func(*terraform.State) error {
								mu.Lock()
								defer mu.Unlock()
								if saw != tc.send {
									return fmt.Errorf("Coolify %s saw axiom domain=%v, want %v", label, saw, tc.send)
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
