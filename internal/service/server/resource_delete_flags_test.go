package server_test

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

func TestServerResource_RemoveResourcesFromServerFalse(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	var gotQuery string
	srv := newVersionedDeletingServer(t, "4.8.0-alpha.0.20261121.1", &mu, &gotQuery)
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.CheckDestroy(srv.URL, "coolify_server", "/api/v1/servers/"),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_server" "test" {
  name                         = "leave-running"
  ip                           = "10.0.0.8"
  private_key_uuid             = "dddd0002-0002-4000-8000-000000000002"
  remove_resources_from_server = false
}`,
				Check: resource.TestCheckResourceAttr("coolify_server.test", "remove_resources_from_server", "false"),
			},
		},
	})

	mu.Lock()
	defer mu.Unlock()
	if gotQuery != "force=true&remove_resources_from_server=false" {
		t.Fatalf("DELETE query = %q", gotQuery)
	}
}

func TestServerResource_RemoveResourcesFromServerDefaultOmitted(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	var gotQuery string
	srv := newVersionedDeletingServer(t, "4.8.0", &mu, &gotQuery)
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.CheckDestroy(srv.URL, "coolify_server", "/api/v1/servers/"),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_server" "test" {
  name             = "default-cleanup"
  ip               = "10.0.0.9"
  private_key_uuid = "dddd0002-0002-4000-8000-000000000002"
}`,
				Check: resource.TestCheckResourceAttr("coolify_server.test", "remove_resources_from_server", "true"),
			},
		},
	})

	mu.Lock()
	defer mu.Unlock()
	if gotQuery != "force=true" {
		t.Fatalf("DELETE query = %q", gotQuery)
	}
}

func newVersionedDeletingServer(t *testing.T, version string, mu *sync.Mutex, gotQuery *string) *httptest.Server {
	t.Helper()
	servers := make(map[string]*client.Server)
	return httptest.NewServer(acctest.WithVersionEndpointVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/servers":
			var input client.CreateServerInput
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				http.Error(w, `{"error":"bad request"}`, http.StatusBadRequest)
				return
			}
			created := &client.Server{
				UUID:           "bbbb0012-0001-4000-8000-000000000001",
				Name:           input.Name,
				IP:             input.IP,
				Port:           input.Port,
				User:           input.User,
				PrivateKeyUUID: input.PrivateKeyUUID,
				IsReachable:    true,
				IsUsable:       true,
				Settings: &client.ServerSettings{
					ConcurrentBuilds:                     2,
					DynamicTimeout:                       3600,
					DeploymentQueueLimit:                 25,
					ConnectionTimeout:                    10,
					ServerDiskUsageNotificationThreshold: 80,
					ServerDiskUsageCheckFrequency:        "*/5 * * * *",
					IsTerminalEnabled:                    true,
				},
			}
			servers[created.UUID] = created
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(created)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v1/servers/"):
			uuid := strings.TrimPrefix(r.URL.Path, "/api/v1/servers/")
			srv, ok := servers[uuid]
			if !ok {
				http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
				return
			}
			resp := *srv
			resp.PrivateKeyUUID = ""
			_ = json.NewEncoder(w).Encode(resp)
		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/api/v1/servers/"):
			*gotQuery = r.URL.RawQuery
			uuid := strings.TrimPrefix(r.URL.Path, "/api/v1/servers/")
			delete(servers, uuid)
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "Server deleted."})
		default:
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		}
	}), version))
}
