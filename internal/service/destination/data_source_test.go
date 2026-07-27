package destination_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestDestinationDataSource(t *testing.T) {
	t.Parallel()
	dest := &client.Destination{
		UUID:       "dddd0001-0001-4000-8000-000000000001",
		Name:       "my-net",
		Network:    "coolify-net",
		Type:       "standalone",
		ServerUUID: "bbbb0001-0001-4000-8000-000000000001",
	}
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet && r.URL.Path == "/api/v1/destinations/"+dest.UUID {
			_ = json.NewEncoder(w).Encode(dest)
			return
		}
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
	})))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
data "coolify_destination" "test" {
  uuid = "dddd0001-0001-4000-8000-000000000001"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_destination.test", "uuid", dest.UUID),
					resource.TestCheckResourceAttr("data.coolify_destination.test", "name", "my-net"),
					resource.TestCheckResourceAttr("data.coolify_destination.test", "network", "coolify-net"),
					resource.TestCheckResourceAttr("data.coolify_destination.test", "type", "standalone"),
					resource.TestCheckResourceAttr("data.coolify_destination.test", "server_uuid", dest.ServerUUID),
				),
			},
		},
	})
}

func TestDestinationDataSource_NotFound(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
	})))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
data "coolify_destination" "test" {
  uuid = "00000000-0000-4000-8000-000000000000"
}`,
				ExpectError: regexp.MustCompile(`Error reading destination`),
			},
		},
	})
}
