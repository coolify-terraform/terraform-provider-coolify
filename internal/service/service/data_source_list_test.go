package service_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SebTardif/terraform-provider-coolify/internal/acctest"
	"github.com/SebTardif/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestServicesListDataSource(t *testing.T) {
	services := []client.Service{
		{
			UUID:        "svc-list-uuid-1",
			Name:        "svc-alpha",
			Description: "First service",
			Type:        "plausible",
		},
		{
			UUID:        "svc-list-uuid-2",
			Name:        "svc-beta",
			Description: "Second service",
			Type:        "minio",
		},
	}

	mockSrv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method == http.MethodGet && r.URL.Path == "/api/v1/services" {
			json.NewEncoder(w).Encode(services)
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
  endpoint = %q
  token    = "test-token"
}

data "coolify_services" "test" {}
`, mockSrv.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_services.test", "services.#", "2"),
					resource.TestCheckResourceAttr("data.coolify_services.test", "services.0.uuid", "svc-list-uuid-1"),
					resource.TestCheckResourceAttr("data.coolify_services.test", "services.0.name", "svc-alpha"),
					resource.TestCheckResourceAttr("data.coolify_services.test", "services.0.description", "First service"),
					resource.TestCheckResourceAttr("data.coolify_services.test", "services.0.type", "plausible"),
					resource.TestCheckResourceAttr("data.coolify_services.test", "services.1.uuid", "svc-list-uuid-2"),
					resource.TestCheckResourceAttr("data.coolify_services.test", "services.1.name", "svc-beta"),
					resource.TestCheckResourceAttr("data.coolify_services.test", "services.1.type", "minio"),
				),
			},
		},
	})
}
