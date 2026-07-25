package digitalocean_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func newDOServerMock() *httptest.Server {
	servers := map[string]*client.Server{}
	var mu sync.Mutex
	return httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/servers/digitalocean":
			var input client.CreateDigitalOceanServerInput
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				http.Error(w, `{"error":"bad request"}`, http.StatusBadRequest)
				return
			}
			if input.Name == "" || input.Region == "" || input.Size == "" || input.Image == nil || input.PrivateKeyUUID == "" {
				http.Error(w, `{"error":"missing fields"}`, http.StatusUnprocessableEntity)
				return
			}
			srv := &client.Server{UUID: "aaaa0001-0001-4000-8000-000000000001", Name: input.Name, IP: "203.0.113.50", Port: 22, User: "root", PrivateKeyUUID: input.PrivateKeyUUID, IsReachable: true, IsUsable: true,
				Settings: &client.ServerSettings{ConcurrentBuilds: 2, DynamicTimeout: 3600, DeploymentQueueLimit: 25, ConnectionTimeout: 10, ServerDiskUsageNotificationThreshold: 80, ServerDiskUsageCheckFrequency: "*/5 * * * *"}}
			servers[srv.UUID] = srv
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(srv)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v1/servers/"):
			uuid := strings.TrimPrefix(r.URL.Path, "/api/v1/servers/")
			srv, ok := servers[uuid]
			if !ok {
				http.Error(w, `{}`, http.StatusNotFound)
				return
			}
			resp := *srv
			resp.PrivateKeyUUID = ""
			json.NewEncoder(w).Encode(resp)
		case r.Method == http.MethodPatch:
			uuid := strings.TrimPrefix(r.URL.Path, "/api/v1/servers/")
			json.NewEncoder(w).Encode(servers[uuid])
		case r.Method == http.MethodDelete:
			delete(servers, strings.TrimPrefix(r.URL.Path, "/api/v1/servers/"))
		default:
			http.Error(w, `{}`, http.StatusNotFound)
		}
	})))
}

func TestDigitalOceanServerResource_Create(t *testing.T) {
	t.Parallel()
	srv := newDOServerMock()
	defer srv.Close()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_server_digitalocean" "test" {
  name                      = "do-node"
  cloud_provider_token_uuid = "cccc0001-0001-4000-8000-000000000001"
  region                    = "nyc1"
  size                      = "s-1vcpu-1gb"
  image                     = "ubuntu-24-04-x64"
  private_key_uuid          = "dddd0002-0002-4000-8000-000000000002"
}`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("coolify_server_digitalocean.test", "uuid", "aaaa0001-0001-4000-8000-000000000001"),
				resource.TestCheckResourceAttr("coolify_server_digitalocean.test", "name", "do-node"),
				resource.TestCheckResourceAttr("coolify_server_digitalocean.test", "ip", "203.0.113.50"),
			),
		}, {
			Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_server_digitalocean" "test" {
  name                      = "do-node"
  cloud_provider_token_uuid = "cccc0001-0001-4000-8000-000000000001"
  region                    = "nyc1"
  size                      = "s-1vcpu-1gb"
  image                     = "ubuntu-24-04-x64"
  private_key_uuid          = "dddd0002-0002-4000-8000-000000000002"
}`,
			PlanOnly: true, ExpectNonEmptyPlan: false,
		}},
	})
}
