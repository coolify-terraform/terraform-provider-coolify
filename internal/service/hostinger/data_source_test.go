package hostinger_test

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestHostingerDataCentersDataSource_OldCoolify(t *testing.T) {
	t.Parallel()
	called := false
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/hostinger/data-centers", func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusNotFound)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(mux, "4.4.2"))
	defer srv.Close()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.ProviderBlockForURL(srv.URL) + `
data "coolify_hostinger_data_centers" "test" {
  cloud_provider_token_uuid = "cccc0001-0001-4000-8000-000000000001"
}
`,
			ExpectError: regexp.MustCompile(`Coolify >= v4\.5\.0`),
		}},
	})
	if called {
		t.Fatal("old Coolify called the Hostinger list route")
	}
}

func TestHostingerDataCentersDataSource(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/hostinger/data-centers", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("cloud_provider_token_uuid") != "cccc0001-0001-4000-8000-000000000001" {
			t.Errorf("token = %q", r.URL.Query().Get("cloud_provider_token_uuid"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id":1,"name":"New York","location":"United States","city":"New York","continent":"North America"}]`))
	})
	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(mux, "4.5.0"))
	defer srv.Close()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.ProviderBlockForURL(srv.URL) + `
data "coolify_hostinger_data_centers" "test" {
  cloud_provider_token_uuid = "cccc0001-0001-4000-8000-000000000001"
}
`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("data.coolify_hostinger_data_centers.test", "data_centers.#", "1"),
				resource.TestCheckResourceAttr("data.coolify_hostinger_data_centers.test", "data_centers.0.id", "1"),
				resource.TestCheckResourceAttr("data.coolify_hostinger_data_centers.test", "data_centers.0.city", "New York"),
			),
		}},
	})
}

func TestHostingerCatalogDataSource(t *testing.T) {
	t.Parallel()
	testHostingerList(t, "coolify_hostinger_catalog", "/api/v1/hostinger/catalog", "items.0.id", "hostingercom-vps-kvm2",
		`[{"id":"hostingercom-vps-kvm2","name":"KVM 2","category":"VPS","prices":[{"id":"p1","name":"monthly","currency":"USD","price":1099,"first_period_price":999,"period":1,"period_unit":"month"}]}]`)
}

func TestHostingerTemplatesDataSource(t *testing.T) {
	t.Parallel()
	testHostingerList(t, "coolify_hostinger_templates", "/api/v1/hostinger/templates", "templates.0.name", "Ubuntu 24.04",
		`[{"id":7,"name":"Ubuntu 24.04","description":"LTS","documentation":"https://example.com/ubuntu"}]`)
}

func TestHostingerSSHKeysDataSource(t *testing.T) {
	t.Parallel()
	testHostingerList(t, "coolify_hostinger_ssh_keys", "/api/v1/hostinger/ssh-keys", "ssh_keys.0.name", "laptop",
		`[{"id":3,"name":"laptop","key":"ssh-ed25519 TESTKEY"}]`)
}

func TestHostingerPostInstallScriptsDataSource(t *testing.T) {
	t.Parallel()
	testHostingerList(t, "coolify_hostinger_post_install_scripts", "/api/v1/hostinger/post-install-scripts", "scripts.0.name", "docker",
		`[{"id":9,"name":"docker","content":"apt-get update","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-02T00:00:00Z"}]`)
}

func testHostingerList(t *testing.T, dataType, path, attr, want, body string) {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("GET "+path, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("cloud_provider_token_uuid") != "cccc0001-0001-4000-8000-000000000001" {
			t.Errorf("token = %q", r.URL.Query().Get("cloud_provider_token_uuid"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	})
	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(mux, "4.5.0"))
	defer srv.Close()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.ProviderBlockForURL(srv.URL) + `
data "` + dataType + `" "test" {
  cloud_provider_token_uuid = "cccc0001-0001-4000-8000-000000000001"
}
`,
			Check: resource.TestCheckResourceAttr("data."+dataType+".test", attr, want),
		}},
	})
}
