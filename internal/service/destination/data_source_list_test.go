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

func TestDestinationListDataSource(t *testing.T) {
	t.Parallel()
	items := []client.Destination{
		{
			UUID:       "dddd0001-0001-4000-8000-000000000001",
			Name:       "first-net",
			Network:    "net-a",
			Type:       "standalone",
			ServerUUID: "bbbb0001-0001-4000-8000-000000000001",
		},
		{
			UUID:       "dddd0002-0002-4000-8000-000000000002",
			Name:       "second-net",
			Network:    "net-b",
			Type:       "swarm",
			ServerUUID: "bbbb0001-0001-4000-8000-000000000001",
		},
	}
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet && r.URL.Path == "/api/v1/destinations" {
			_ = json.NewEncoder(w).Encode(items)
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
data "coolify_destinations" "all" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_destinations.all", "destinations.#", "2"),
					resource.TestCheckResourceAttr("data.coolify_destinations.all", "destinations.0.network", "net-a"),
					resource.TestCheckResourceAttr("data.coolify_destinations.all", "destinations.1.type", "swarm"),
				),
			},
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
data "coolify_destinations" "filtered" {
  filter {
    name   = "name"
    values = ["first-net"]
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_destinations.filtered", "destinations.#", "1"),
					resource.TestCheckResourceAttr("data.coolify_destinations.filtered", "destinations.0.name", "first-net"),
					resource.TestCheckResourceAttr("data.coolify_destinations.filtered", "destinations.0.network", "net-a"),
				),
			},
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
data "coolify_destinations" "by_type" {
  filter {
    name   = "type"
    values = ["swarm"]
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_destinations.by_type", "destinations.#", "1"),
					resource.TestCheckResourceAttr("data.coolify_destinations.by_type", "destinations.0.uuid", "dddd0002-0002-4000-8000-000000000002"),
				),
			},
		},
	})
}

func TestDestinationListDataSource_APIError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
	})))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
data "coolify_destinations" "test" {}
`,
				ExpectError: regexp.MustCompile(`Error listing destinations`),
			},
		},
	})
}
