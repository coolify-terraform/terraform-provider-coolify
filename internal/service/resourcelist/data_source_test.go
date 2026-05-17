package resourcelist_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestResourcesDataSource(t *testing.T) {
	t.Parallel()

	mockSrv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method == http.MethodGet && r.URL.Path == "/api/v1/resources" {
			json.NewEncoder(w).Encode([]map[string]interface{}{
				{
					"uuid":   "cccc0001-0001-4000-8000-000000000001",
					"name":   "my-app",
					"type":   "application",
					"status": "running",
				},
				{
					"uuid":   "cccc0002-0002-4000-8000-000000000002",
					"name":   "my-db",
					"type":   "database",
					"status": "stopped",
				},
			})
			return
		}
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
	})))
	defer mockSrv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
provider "coolify" {
  endpoint  = %q
  token = "test-token"
}

data "coolify_resources" "test" {}
`, mockSrv.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_resources.test", "resources.#", "2"),
					resource.TestCheckResourceAttr("data.coolify_resources.test", "resources.0.uuid", "cccc0001-0001-4000-8000-000000000001"),
					resource.TestCheckResourceAttr("data.coolify_resources.test", "resources.0.name", "my-app"),
					resource.TestCheckResourceAttr("data.coolify_resources.test", "resources.0.type", "application"),
					resource.TestCheckResourceAttr("data.coolify_resources.test", "resources.0.status", "running"),
					resource.TestCheckResourceAttr("data.coolify_resources.test", "resources.1.uuid", "cccc0002-0002-4000-8000-000000000002"),
					resource.TestCheckResourceAttr("data.coolify_resources.test", "resources.1.name", "my-db"),
					resource.TestCheckResourceAttr("data.coolify_resources.test", "resources.1.type", "database"),
					resource.TestCheckResourceAttr("data.coolify_resources.test", "resources.1.status", "stopped"),
				),
			},
			{
				Config: fmt.Sprintf(`
provider "coolify" {
  endpoint  = %q
  token = "test-token"
}

data "coolify_resources" "filtered" {
  filter {
    name   = "type"
    values = ["application"]
  }
}
`, mockSrv.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_resources.filtered", "resources.#", "1"),
					resource.TestCheckResourceAttr("data.coolify_resources.filtered", "resources.0.name", "my-app"),
					resource.TestCheckResourceAttr("data.coolify_resources.filtered", "resources.0.type", "application"),
				),
			},
		},
	})
}
