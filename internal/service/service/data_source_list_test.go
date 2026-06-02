package service_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestServicesListDataSource(t *testing.T) {
	t.Parallel()
	services := []client.Service{
		{
			UUID:        "svc-list-uuid-1",
			Name:        "svc-alpha",
			Description: "First service",
			Type:        "plausible",
			Status:      "running",
		},
		{
			UUID:        "svc-list-uuid-2",
			Name:        "svc-beta",
			Description: "Second service",
			Type:        "minio",
			Status:      "stopped",
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
					resource.TestCheckResourceAttr("data.coolify_services.test", "services.0.status", "running"),
					resource.TestCheckResourceAttr("data.coolify_services.test", "services.1.uuid", "svc-list-uuid-2"),
					resource.TestCheckResourceAttr("data.coolify_services.test", "services.1.name", "svc-beta"),
					resource.TestCheckResourceAttr("data.coolify_services.test", "services.1.type", "minio"),
					resource.TestCheckResourceAttr("data.coolify_services.test", "services.1.status", "stopped"),
				),
			},
			{
				Config: fmt.Sprintf(`
provider "coolify" {
  endpoint = %q
  token    = "test-token"
}

data "coolify_services" "filtered" {
  filter {
    name   = "type"
    values = ["minio"]
  }
}
`, mockSrv.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_services.filtered", "services.#", "1"),
					resource.TestCheckResourceAttr("data.coolify_services.filtered", "services.0.name", "svc-beta"),
					resource.TestCheckResourceAttr("data.coolify_services.filtered", "services.0.type", "minio"),
					resource.TestCheckResourceAttr("data.coolify_services.filtered", "services.0.status", "stopped"),
				),
			},
			{
				Config: fmt.Sprintf(`
provider "coolify" {
  endpoint = %q
  token    = "test-token"
}

data "coolify_services" "running" {
  filter {
    name   = "status"
    values = ["running"]
  }
}
`, mockSrv.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_services.running", "services.#", "1"),
					resource.TestCheckResourceAttr("data.coolify_services.running", "services.0.name", "svc-alpha"),
					resource.TestCheckResourceAttr("data.coolify_services.running", "services.0.status", "running"),
				),
			},
		},
	})
}

func TestServicesListDataSource_APIError(t *testing.T) {
	t.Parallel()
	mockSrv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
	})))
	defer mockSrv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(mockSrv.URL) + `
data "coolify_services" "test" {}
`,
				ExpectError: regexp.MustCompile(`Error listing services`),
			},
		},
	})
}
