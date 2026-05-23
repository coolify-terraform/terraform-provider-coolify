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

func TestHetznerImagesDataSource(t *testing.T) {
	t.Parallel()

	images := []client.HetznerImage{
		{ID: 1, Name: "ubuntu-22.04", Description: "Ubuntu 22.04"},
		{ID: 2, Name: "debian-12", Description: "Debian 12"},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/hetzner/images", func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("cloud_provider_token_uuid"); got != "cccc0001-0001-4000-8000-000000000001" {
			t.Errorf("expected cloud_provider_token_uuid=cccc0001-0001-4000-8000-000000000001, got %q", got)
			http.Error(w, "missing cloud_provider_token_uuid", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(images)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
data "coolify_hetzner_images" "test" {
  cloud_provider_token_uuid = "cccc0001-0001-4000-8000-000000000001"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_hetzner_images.test", "images.#", "2"),
					resource.TestCheckResourceAttr("data.coolify_hetzner_images.test", "images.0.id", "1"),
					resource.TestCheckResourceAttr("data.coolify_hetzner_images.test", "images.0.name", "ubuntu-22.04"),
					resource.TestCheckResourceAttr("data.coolify_hetzner_images.test", "images.0.description", "Ubuntu 22.04"),
					resource.TestCheckResourceAttr("data.coolify_hetzner_images.test", "images.1.id", "2"),
					resource.TestCheckResourceAttr("data.coolify_hetzner_images.test", "images.1.name", "debian-12"),
					resource.TestCheckResourceAttr("data.coolify_hetzner_images.test", "images.1.description", "Debian 12"),
				),
			},
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
data "coolify_hetzner_images" "filtered" {
  cloud_provider_token_uuid = "cccc0001-0001-4000-8000-000000000001"
  filter {
    name   = "name"
    values = ["debian-12"]
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_hetzner_images.filtered", "images.#", "1"),
					resource.TestCheckResourceAttr("data.coolify_hetzner_images.filtered", "images.0.name", "debian-12"),
				),
			},
		},
	})
}
