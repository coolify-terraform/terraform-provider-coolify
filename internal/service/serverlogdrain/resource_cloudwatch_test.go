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

func TestServerLogDrainResource_CloudWatch(t *testing.T) {
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
			const serverUUID = "aaaa0001-0001-4000-8000-000000000001"
			var mu sync.Mutex
			sawCloudWatch := false
			store := map[string]any{
				"is_logdrain_newrelic_enabled": false,
				"is_logdrain_axiom_enabled":    false,
				"is_logdrain_custom_enabled":   false,
			}
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
					if _, ok := body["is_logdrain_cloudwatch_enabled"]; ok {
						sawCloudWatch = true
					}
					for k, v := range body {
						store[k] = v
					}
					_ = json.NewEncoder(w).Encode(store)
				case http.MethodGet:
					_ = json.NewEncoder(w).Encode(store)
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
  server_uuid                    = "`+serverUUID+`"
  is_logdrain_cloudwatch_enabled = true
  logdrain_cloudwatch_region     = "us-east-1"
  logdrain_cloudwatch_group      = "coolify"
  logdrain_cloudwatch_stream_prefix = "app"
`),
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("coolify_server_log_drain.test", "is_logdrain_cloudwatch_enabled", "true"),
							resource.TestCheckResourceAttr("coolify_server_log_drain.test", "logdrain_cloudwatch_region", "us-east-1"),
							func(*terraform.State) error {
								mu.Lock()
								defer mu.Unlock()
								if sawCloudWatch != tc.send {
									return fmt.Errorf("Coolify %s saw cloudwatch=%v, want %v", label, sawCloudWatch, tc.send)
								}
								if tc.send && store["logdrain_cloudwatch_group"] != "coolify" {
									return fmt.Errorf("group=%v", store["logdrain_cloudwatch_group"])
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
