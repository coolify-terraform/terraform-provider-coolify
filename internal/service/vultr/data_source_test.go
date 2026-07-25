package vultr_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const vultrToken = "cccc0001-0001-4000-8000-000000000001"

func TestVultrRegionsDataSource(t *testing.T) {
	t.Parallel()
	regions := []client.VultrRegion{
		{ID: "ewr", City: "New Jersey", Country: "US", Continent: "North America"},
		{ID: "lax", City: "Los Angeles", Country: "US", Continent: "North America"},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/vultr/regions", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("cloud_provider_token_uuid") != vultrToken {
			t.Errorf("token mismatch")
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
data "coolify_vultr_regions" "test" {
  cloud_provider_token_uuid = "` + vultrToken + `"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_vultr_regions.test", "regions.#", "2"),
					resource.TestCheckResourceAttr("data.coolify_vultr_regions.test", "regions.0.id", "ewr"),
					resource.TestCheckResourceAttr("data.coolify_vultr_regions.test", "regions.0.city", "New Jersey"),
					resource.TestCheckResourceAttr("data.coolify_vultr_regions.test", "regions.1.id", "lax"),
				),
			},
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
data "coolify_vultr_regions" "filtered" {
  cloud_provider_token_uuid = "` + vultrToken + `"
  filter {
    name   = "id"
    values = ["lax"]
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_vultr_regions.filtered", "regions.#", "1"),
					resource.TestCheckResourceAttr("data.coolify_vultr_regions.filtered", "regions.0.city", "Los Angeles"),
				),
			},
		},
	})
}

func TestVultrPlansDataSource(t *testing.T) {
	t.Parallel()
	plans := []client.VultrPlan{
		{ID: "vc2-1c-1gb", VCPUCount: 1, RAM: 1024, Disk: 25, MonthlyCost: 5, Type: "vc2"},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/vultr/plans", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("cloud_provider_token_uuid") != vultrToken {
			http.Error(w, "bad token", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(plans)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
data "coolify_vultr_plans" "test" {
  cloud_provider_token_uuid = "` + vultrToken + `"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_vultr_plans.test", "plans.#", "1"),
					resource.TestCheckResourceAttr("data.coolify_vultr_plans.test", "plans.0.id", "vc2-1c-1gb"),
					resource.TestCheckResourceAttr("data.coolify_vultr_plans.test", "plans.0.vcpu_count", "1"),
					resource.TestCheckResourceAttr("data.coolify_vultr_plans.test", "plans.0.ram", "1024"),
				),
			},
		},
	})
}

func TestVultrOSDataSource(t *testing.T) {
	t.Parallel()
	oses := []client.VultrOS{
		{ID: 1743, Name: "Ubuntu 22.04 LTS", Arch: "x64", Family: "ubuntu"},
		{ID: 2136, Name: "Debian 12", Arch: "x64", Family: "debian"},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/vultr/os", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("cloud_provider_token_uuid") != vultrToken {
			http.Error(w, "bad token", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(oses)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
data "coolify_vultr_os" "test" {
  cloud_provider_token_uuid = "` + vultrToken + `"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_vultr_os.test", "operating_systems.#", "2"),
					resource.TestCheckResourceAttr("data.coolify_vultr_os.test", "operating_systems.0.id", "1743"),
					resource.TestCheckResourceAttr("data.coolify_vultr_os.test", "operating_systems.0.name", "Ubuntu 22.04 LTS"),
					resource.TestCheckResourceAttr("data.coolify_vultr_os.test", "operating_systems.0.family", "ubuntu"),
				),
			},
		},
	})
}

func TestVultrSSHKeysDataSource(t *testing.T) {
	t.Parallel()
	keys := []client.VultrSSHKey{
		{ID: "key-1", Name: "laptop", DateCreated: "2024-01-01"},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/vultr/ssh-keys", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("cloud_provider_token_uuid") != vultrToken {
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
data "coolify_vultr_ssh_keys" "test" {
  cloud_provider_token_uuid = "` + vultrToken + `"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_vultr_ssh_keys.test", "ssh_keys.#", "1"),
					resource.TestCheckResourceAttr("data.coolify_vultr_ssh_keys.test", "ssh_keys.0.id", "key-1"),
					resource.TestCheckResourceAttr("data.coolify_vultr_ssh_keys.test", "ssh_keys.0.name", "laptop"),
				),
			},
		},
	})
}
