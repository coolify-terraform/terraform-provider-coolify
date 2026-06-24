package version_test

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync/atomic"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestVersionDataSource_ClientError(t *testing.T) {
	t.Parallel()
	// The provider's Configure and the data source's Read both call GET /api/v1/version.
	// Allow the first call (Configure health check) to succeed, then fail subsequent calls.
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/api/v1/version" {
			if calls.Add(1) == 1 {
				w.Header().Set("Content-Type", "text/html")
				w.Write([]byte("v4.1.0"))
				return
			}
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
	}))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
data "coolify_version" "test" {}
`,
				ExpectError: regexp.MustCompile(`Error reading Coolify version`),
			},
		},
	})
}

func TestVersionDataSource(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/api/v1/version" {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(`v4.1.0-beta.362`))
			return
		}
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
	}))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
data "coolify_version" "test" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_version.test", "version", "v4.1.0-beta.362"),
				),
			},
		},
	})
}
