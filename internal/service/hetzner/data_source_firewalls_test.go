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

func TestHetznerFirewallsDataSource(t *testing.T) {
	t.Parallel()

	firewalls := []client.HetznerFirewall{
		{ID: 38, Name: "web"},
		{ID: 39, Name: "db"},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/hetzner/firewalls", func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("cloud_provider_token_uuid"); got != "cccc0001-0001-4000-8000-000000000001" {
			t.Errorf("expected cloud_provider_token_uuid=cccc0001-0001-4000-8000-000000000001, got %q", got)
			http.Error(w, "missing cloud_provider_token_uuid", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(firewalls)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
data "coolify_hetzner_firewalls" "test" {
  cloud_provider_token_uuid = "cccc0001-0001-4000-8000-000000000001"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_hetzner_firewalls.test", "firewalls.#", "2"),
					resource.TestCheckResourceAttr("data.coolify_hetzner_firewalls.test", "firewalls.0.id", "38"),
					resource.TestCheckResourceAttr("data.coolify_hetzner_firewalls.test", "firewalls.0.name", "web"),
					resource.TestCheckResourceAttr("data.coolify_hetzner_firewalls.test", "firewalls.1.id", "39"),
					resource.TestCheckResourceAttr("data.coolify_hetzner_firewalls.test", "firewalls.1.name", "db"),
				),
			},
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
data "coolify_hetzner_firewalls" "filtered" {
  cloud_provider_token_uuid = "cccc0001-0001-4000-8000-000000000001"
  filter {
    name   = "name"
    values = ["db"]
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_hetzner_firewalls.filtered", "firewalls.#", "1"),
					resource.TestCheckResourceAttr("data.coolify_hetzner_firewalls.filtered", "firewalls.0.name", "db"),
				),
			},
		},
	})
}
