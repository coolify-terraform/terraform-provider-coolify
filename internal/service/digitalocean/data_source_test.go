package digitalocean_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const doToken = "cccc0001-0001-4000-8000-000000000001"

func TestDigitalOceanRegionsDataSource(t *testing.T) {
	t.Parallel()
	regions := []client.DigitalOceanRegion{
		{Slug: "nyc1", Name: "New York 1", Available: true},
		{Slug: "sfo3", Name: "San Francisco 3", Available: false},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/digitalocean/regions", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("cloud_provider_token_uuid") != doToken {
			t.Errorf("token mismatch: %q", r.URL.Query().Get("cloud_provider_token_uuid"))
			http.Error(w, "bad token", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(regions)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
data "coolify_digitalocean_regions" "test" {
  cloud_provider_token_uuid = "` + doToken + `"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_digitalocean_regions.test", "regions.#", "2"),
					resource.TestCheckResourceAttr("data.coolify_digitalocean_regions.test", "regions.0.slug", "nyc1"),
					resource.TestCheckResourceAttr("data.coolify_digitalocean_regions.test", "regions.0.name", "New York 1"),
					resource.TestCheckResourceAttr("data.coolify_digitalocean_regions.test", "regions.0.available", "true"),
					resource.TestCheckResourceAttr("data.coolify_digitalocean_regions.test", "regions.1.slug", "sfo3"),
				),
			},
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
data "coolify_digitalocean_regions" "filtered" {
  cloud_provider_token_uuid = "` + doToken + `"
  filter {
    name   = "slug"
    values = ["sfo3"]
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_digitalocean_regions.filtered", "regions.#", "1"),
					resource.TestCheckResourceAttr("data.coolify_digitalocean_regions.filtered", "regions.0.slug", "sfo3"),
				),
			},
		},
	})
}

func TestDigitalOceanSizesDataSource(t *testing.T) {
	t.Parallel()
	sizes := []client.DigitalOceanSize{
		{Slug: "s-1vcpu-1gb", Memory: 1024, VCPUs: 1, Disk: 25, PriceMonthly: 6, Available: true},
		{Slug: "s-2vcpu-2gb", Memory: 2048, VCPUs: 2, Disk: 50, PriceMonthly: 18, Available: true},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/digitalocean/sizes", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("cloud_provider_token_uuid") != doToken {
			http.Error(w, "bad token", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(sizes)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
data "coolify_digitalocean_sizes" "test" {
  cloud_provider_token_uuid = "` + doToken + `"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_digitalocean_sizes.test", "sizes.#", "2"),
					resource.TestCheckResourceAttr("data.coolify_digitalocean_sizes.test", "sizes.0.slug", "s-1vcpu-1gb"),
					resource.TestCheckResourceAttr("data.coolify_digitalocean_sizes.test", "sizes.0.memory", "1024"),
					resource.TestCheckResourceAttr("data.coolify_digitalocean_sizes.test", "sizes.0.vcpus", "1"),
					resource.TestCheckResourceAttr("data.coolify_digitalocean_sizes.test", "sizes.0.available", "true"),
				),
			},
		},
	})
}

func TestDigitalOceanImagesDataSource(t *testing.T) {
	t.Parallel()
	images := []client.DigitalOceanImage{
		{ID: 1, Name: "22.04 x64", Distribution: "Ubuntu", Slug: "ubuntu-22-04-x64", Public: true, Type: "snapshot"},
		{ID: 2, Name: "12 x64", Distribution: "Debian", Slug: "debian-12-x64", Public: true, Type: "snapshot"},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/digitalocean/images", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("cloud_provider_token_uuid") != doToken {
			http.Error(w, "bad token", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(images)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
data "coolify_digitalocean_images" "test" {
  cloud_provider_token_uuid = "` + doToken + `"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_digitalocean_images.test", "images.#", "2"),
					resource.TestCheckResourceAttr("data.coolify_digitalocean_images.test", "images.0.slug", "ubuntu-22-04-x64"),
					resource.TestCheckResourceAttr("data.coolify_digitalocean_images.test", "images.0.distribution", "Ubuntu"),
					resource.TestCheckResourceAttr("data.coolify_digitalocean_images.test", "images.0.public", "true"),
				),
			},
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
data "coolify_digitalocean_images" "filtered" {
  cloud_provider_token_uuid = "` + doToken + `"
  filter {
    name   = "distribution"
    values = ["Debian"]
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_digitalocean_images.filtered", "images.#", "1"),
					resource.TestCheckResourceAttr("data.coolify_digitalocean_images.filtered", "images.0.slug", "debian-12-x64"),
				),
			},
		},
	})
}

func TestDigitalOceanSSHKeysDataSource(t *testing.T) {
	t.Parallel()
	keys := []client.DigitalOceanSSHKey{
		{ID: 10, Name: "laptop", Fingerprint: "aa:bb"},
		{ID: 11, Name: "ci", Fingerprint: "cc:dd"},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/digitalocean/ssh-keys", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("cloud_provider_token_uuid") != doToken {
			http.Error(w, "bad token", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(keys)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
data "coolify_digitalocean_ssh_keys" "test" {
  cloud_provider_token_uuid = "` + doToken + `"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_digitalocean_ssh_keys.test", "ssh_keys.#", "2"),
					resource.TestCheckResourceAttr("data.coolify_digitalocean_ssh_keys.test", "ssh_keys.0.id", "10"),
					resource.TestCheckResourceAttr("data.coolify_digitalocean_ssh_keys.test", "ssh_keys.0.name", "laptop"),
					resource.TestCheckResourceAttr("data.coolify_digitalocean_ssh_keys.test", "ssh_keys.0.fingerprint", "aa:bb"),
				),
			},
		},
	})
}
