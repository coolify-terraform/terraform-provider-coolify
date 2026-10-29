package s3storage_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestS3StorageResource_PathStyleEndpoint(t *testing.T) {
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
			var mu sync.Mutex
			var saw *bool
			srv := httptest.NewServer(acctest.WithVersionEndpointVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				body, _ := io.ReadAll(r.Body)
				var payload map[string]any
				_ = json.Unmarshal(body, &payload)
				mu.Lock()
				if _, ok := payload["use_path_style_endpoint"]; ok && r.Method == http.MethodPost {
					v, _ := payload["use_path_style_endpoint"].(bool)
					saw = &v
				}
				mu.Unlock()
				switch r.Method {
				case http.MethodPost:
					w.WriteHeader(http.StatusCreated)
					_ = json.NewEncoder(w).Encode(map[string]any{
						"uuid": "s3-uuid", "name": "backups", "endpoint": "https://s3.example.com",
						"bucket": "b", "region": "us-east-1", "use_path_style_endpoint": false,
					})
				case http.MethodGet:
					stored := map[string]any{
						"uuid": "s3-uuid", "name": "backups", "endpoint": "https://s3.example.com",
						"bucket": "b", "region": "us-east-1",
					}
					if tc.send {
						stored["use_path_style_endpoint"] = false
					}
					_ = json.NewEncoder(w).Encode(stored)
				case http.MethodDelete:
					w.WriteHeader(http.StatusNoContent)
				default:
					http.Error(w, `{"message":"method"}`, http.StatusMethodNotAllowed)
				}
			}), tc.version))
			defer srv.Close()

			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
				Steps: []resource.TestStep{
					{
						Config: acctest.TestResourceConfig(srv.URL, "coolify_s3_storage", "test", `
  name                     = "backups"
  endpoint                 = "https://s3.example.com"
  bucket                   = "b"
  region                   = "us-east-1"
  key                      = "access"
  secret                   = "secret"
  use_path_style_endpoint  = false
`),
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("coolify_s3_storage.test", "use_path_style_endpoint", "false"),
							func(*terraform.State) error {
								mu.Lock()
								defer mu.Unlock()
								if tc.send {
									if saw == nil || *saw {
										return fmt.Errorf("Coolify %s create path style = %v, want false", label, saw)
									}
									return nil
								}
								if saw != nil {
									return fmt.Errorf("Coolify %s sent use_path_style_endpoint", label)
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
