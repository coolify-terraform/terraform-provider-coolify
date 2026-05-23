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

func TestHetznerLocationsDataSource(t *testing.T) {
	t.Parallel()

	locations := []client.HetznerLocation{
		{ID: 1, Name: "fsn1", Description: "Falkenstein DC Park 1", City: "Falkenstein", Country: "DE"},
		{ID: 2, Name: "nbg1", Description: "Nuremberg DC Park 1", City: "Nuremberg", Country: "DE"},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/hetzner/locations", func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("cloud_provider_token_uuid"); got != "cccc0001-0001-4000-8000-000000000001" {
			t.Errorf("expected cloud_provider_token_uuid=cccc0001-0001-4000-8000-000000000001, got %q", got)
			http.Error(w, "missing cloud_provider_token_uuid", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(locations)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
data "coolify_hetzner_locations" "test" {
  cloud_provider_token_uuid = "cccc0001-0001-4000-8000-000000000001"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_hetzner_locations.test", "locations.#", "2"),
					resource.TestCheckResourceAttr("data.coolify_hetzner_locations.test", "locations.0.id", "1"),
					resource.TestCheckResourceAttr("data.coolify_hetzner_locations.test", "locations.0.name", "fsn1"),
					resource.TestCheckResourceAttr("data.coolify_hetzner_locations.test", "locations.0.description", "Falkenstein DC Park 1"),
					resource.TestCheckResourceAttr("data.coolify_hetzner_locations.test", "locations.0.city", "Falkenstein"),
					resource.TestCheckResourceAttr("data.coolify_hetzner_locations.test", "locations.0.country", "DE"),
					resource.TestCheckResourceAttr("data.coolify_hetzner_locations.test", "locations.1.name", "nbg1"),
				),
			},
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
data "coolify_hetzner_locations" "filtered" {
  cloud_provider_token_uuid = "cccc0001-0001-4000-8000-000000000001"
  filter {
    name   = "city"
    values = ["Nuremberg"]
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_hetzner_locations.filtered", "locations.#", "1"),
					resource.TestCheckResourceAttr("data.coolify_hetzner_locations.filtered", "locations.0.name", "nbg1"),
				),
			},
		},
	})
}
