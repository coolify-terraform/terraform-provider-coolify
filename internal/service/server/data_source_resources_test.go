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

func TestServerResourcesDataSource(t *testing.T) {
	t.Parallel()
	resources := []client.ServerResource{
		{
			UUID: "res-uuid-1",
			Name: "my-app",
			Type: "application",
		},
		{
			UUID: "res-uuid-2",
			Name: "my-db",
			Type: "database",
		},
	}

	mockSrv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/resources") {
			json.NewEncoder(w).Encode(resources)
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
data "coolify_server_resources" "test" {
  server_uuid = "bbbb0001-0001-4000-8000-000000000001"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_server_resources.test", "resources.#", "2"),
					resource.TestCheckResourceAttr("data.coolify_server_resources.test", "resources.0.uuid", "res-uuid-1"),
					resource.TestCheckResourceAttr("data.coolify_server_resources.test", "resources.0.name", "my-app"),
					resource.TestCheckResourceAttr("data.coolify_server_resources.test", "resources.0.type", "application"),
					resource.TestCheckResourceAttr("data.coolify_server_resources.test", "resources.1.uuid", "res-uuid-2"),
					resource.TestCheckResourceAttr("data.coolify_server_resources.test", "resources.1.name", "my-db"),
					resource.TestCheckResourceAttr("data.coolify_server_resources.test", "resources.1.type", "database"),
				),
			},
		},
	})
}
