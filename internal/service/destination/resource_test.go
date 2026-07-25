package destination_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func newDestinationMock() *httptest.Server {
	store := map[string]*client.Destination{}
	var mu sync.Mutex
	return httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/servers/bbbb0001-0001-4000-8000-000000000001/destinations":
			var input client.CreateDestinationInput
			_ = json.NewDecoder(r.Body).Decode(&input)
			d := &client.Destination{UUID: "dddd0001-0001-4000-8000-000000000001", Name: input.Name, Network: input.Network, Type: "standalone", ServerUUID: "bbbb0001-0001-4000-8000-000000000001"}
			if d.Name == "" {
				d.Name = "server-" + d.Network
			}
			store[d.UUID] = d
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(d)
		case r.Method == http.MethodGet && len(r.URL.Path) > len("/api/v1/destinations/"):
			uuid := r.URL.Path[len("/api/v1/destinations/"):]
			d, ok := store[uuid]
			if !ok {
				http.Error(w, `{}`, http.StatusNotFound)
				return
			}
			json.NewEncoder(w).Encode(d)
		case r.Method == http.MethodDelete:
			uuid := r.URL.Path[len("/api/v1/destinations/"):]
			delete(store, uuid)
			w.WriteHeader(http.StatusOK)
		default:
			http.Error(w, `{}`, http.StatusNotFound)
		}
	})))
}

func TestDestinationResource_Create(t *testing.T) {
	t.Parallel()
	srv := newDestinationMock()
	defer srv.Close()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_destination" "test" {
  server_uuid = "bbbb0001-0001-4000-8000-000000000001"
  network     = "coolify-net"
  name        = "my-net"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_destination.test", "uuid", "dddd0001-0001-4000-8000-000000000001"),
					resource.TestCheckResourceAttr("coolify_destination.test", "network", "coolify-net"),
					resource.TestCheckResourceAttr("coolify_destination.test", "name", "my-net"),
					resource.TestCheckResourceAttr("coolify_destination.test", "type", "standalone"),
				),
			},
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_destination" "test" {
  server_uuid = "bbbb0001-0001-4000-8000-000000000001"
  network     = "coolify-net"
  name        = "my-net"
}`,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}
