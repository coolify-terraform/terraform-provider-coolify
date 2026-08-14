package hetzner_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestHetznerNetworksDataSource(t *testing.T) {
	t.Parallel()

	networks := []client.HetznerNetwork{
		{ID: 456, Name: "acme-private", IPRange: "10.0.0.0/16"},
		{ID: 457, Name: "acme-public", IPRange: "10.1.0.0/16"},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/hetzner/networks", func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("cloud_provider_token_uuid"); got != "cccc0001-0001-4000-8000-000000000001" {
			t.Errorf("expected cloud_provider_token_uuid=cccc0001-0001-4000-8000-000000000001, got %q", got)
			http.Error(w, "missing cloud_provider_token_uuid", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(networks)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
data "coolify_hetzner_networks" "test" {
  cloud_provider_token_uuid = "cccc0001-0001-4000-8000-000000000001"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_hetzner_networks.test", "networks.#", "2"),
					resource.TestCheckResourceAttr("data.coolify_hetzner_networks.test", "networks.0.id", "456"),
					resource.TestCheckResourceAttr("data.coolify_hetzner_networks.test", "networks.0.name", "acme-private"),
					resource.TestCheckResourceAttr("data.coolify_hetzner_networks.test", "networks.0.ip_range", "10.0.0.0/16"),
					resource.TestCheckResourceAttr("data.coolify_hetzner_networks.test", "networks.1.id", "457"),
				),
			},
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
data "coolify_hetzner_networks" "filtered" {
  cloud_provider_token_uuid = "cccc0001-0001-4000-8000-000000000001"
  filter {
    name   = "name"
    values = ["acme-private"]
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_hetzner_networks.filtered", "networks.#", "1"),
					resource.TestCheckResourceAttr("data.coolify_hetzner_networks.filtered", "networks.0.name", "acme-private"),
					resource.TestCheckResourceAttr("data.coolify_hetzner_networks.filtered", "networks.0.ip_range", "10.0.0.0/16"),
				),
			},
		},
	})
}
