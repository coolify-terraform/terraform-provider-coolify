package service_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/SebTardif/terraform-provider-coolify/internal/acctest"
	"github.com/SebTardif/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestServiceDataSource(t *testing.T) {
	t.Parallel()
	svc := &client.Service{
		UUID:            "svc-ds-uuid-1",
		Name:            "my-plausible",
		Description:     "Analytics service",
		Type:            "plausible",
		ServerUUID:      "bbbb0001-0001-4000-8000-000000000001",
		ProjectUUID:     "aaaa0001-0001-4000-8000-000000000001",
		EnvironmentName: "production",
	}

	mockSrv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v1/services/") {
			uuid := strings.TrimPrefix(r.URL.Path, "/api/v1/services/")
			if uuid == svc.UUID {
				json.NewEncoder(w).Encode(svc)
				return
			}
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

data "coolify_service" "test" {
  uuid = %q
}
`, mockSrv.URL, svc.UUID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_service.test", "uuid", "svc-ds-uuid-1"),
					resource.TestCheckResourceAttr("data.coolify_service.test", "name", "my-plausible"),
					resource.TestCheckResourceAttr("data.coolify_service.test", "description", "Analytics service"),
					resource.TestCheckResourceAttr("data.coolify_service.test", "type", "plausible"),
					resource.TestCheckResourceAttr("data.coolify_service.test", "server_uuid", "bbbb0001-0001-4000-8000-000000000001"),
					resource.TestCheckResourceAttr("data.coolify_service.test", "project_uuid", "aaaa0001-0001-4000-8000-000000000001"),
					resource.TestCheckResourceAttr("data.coolify_service.test", "environment_name", "production"),
				),
			},
		},
	})
}
