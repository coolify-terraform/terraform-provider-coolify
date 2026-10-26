package serviceapplication_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestServiceApplicationsDataSource(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/services/{uuid}/applications", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
		  {"uuid":"bbbb0002-0002-4000-8000-000000000002","name":"web","human_name":"Web","fqdn":"https://app.example.com","status":"running"},
		  {"uuid":"cccc0003-0003-4000-8000-000000000003","name":"worker","status":"exited"}
		]`))
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.ProviderBlockForURL(srv.URL) + `
data "coolify_service_applications" "test" {
  service_uuid = "aaaa0001-0001-4000-8000-000000000001"
  filter {
    name   = "name"
    values = ["web"]
  }
}
`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("data.coolify_service_applications.test", "applications.#", "1"),
				resource.TestCheckResourceAttr("data.coolify_service_applications.test", "applications.0.name", "web"),
				resource.TestCheckResourceAttr("data.coolify_service_applications.test", "applications.0.uuid", "bbbb0002-0002-4000-8000-000000000002"),
			),
		}},
	})
}
