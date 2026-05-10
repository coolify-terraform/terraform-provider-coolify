package server_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/SebTardif/terraform-provider-coolify/internal/acctest"
	"github.com/SebTardif/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestServerDomainsDataSource(t *testing.T) {
	t.Parallel()
	domains := []client.ServerDomain{
		{
			Domain: "app.example.com",
			IP:     "10.0.0.1",
		},
		{
			Domain: "api.example.com",
			IP:     "10.0.0.2",
		},
	}

	mockSrv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/domains") {
			json.NewEncoder(w).Encode(domains)
			return
		}
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
	})))
	defer mockSrv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderBlock(mockSrv.URL) + `
data "coolify_server_domains" "test" {
  server_uuid = "bbbb0001-0001-4000-8000-000000000001"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_server_domains.test", "domains.#", "2"),
					resource.TestCheckResourceAttr("data.coolify_server_domains.test", "domains.0.domain", "app.example.com"),
					resource.TestCheckResourceAttr("data.coolify_server_domains.test", "domains.0.ip", "10.0.0.1"),
					resource.TestCheckResourceAttr("data.coolify_server_domains.test", "domains.1.domain", "api.example.com"),
					resource.TestCheckResourceAttr("data.coolify_server_domains.test", "domains.1.ip", "10.0.0.2"),
				),
			},
		},
	})
}
