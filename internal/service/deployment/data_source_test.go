package deployment_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SebTardif/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestDeploymentsDataSource_Read(t *testing.T) {
	t.Parallel()
	deployments := []map[string]interface{}{
		{"uuid": "dep-001", "status": "finished", "server_uuid": "srv-001"},
		{"uuid": "dep-002", "status": "queued", "server_uuid": "srv-001"},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/deployments", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(deployments)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
data "coolify_deployments" "test" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_deployments.test", "deployments.#", "2"),
					resource.TestCheckResourceAttr("data.coolify_deployments.test", "deployments.0.uuid", "dep-001"),
					resource.TestCheckResourceAttr("data.coolify_deployments.test", "deployments.0.status", "finished"),
					resource.TestCheckResourceAttr("data.coolify_deployments.test", "deployments.0.server_uuid", "srv-001"),
					resource.TestCheckResourceAttr("data.coolify_deployments.test", "deployments.1.uuid", "dep-002"),
					resource.TestCheckResourceAttr("data.coolify_deployments.test", "deployments.1.status", "queued"),
				),
			},
		},
	})
}

func TestDeploymentsDataSource_ReadByApplication(t *testing.T) {
	t.Parallel()
	appDeployments := []map[string]interface{}{
		{"uuid": "dep-app-001", "status": "finished", "server_uuid": "srv-002"},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/deployments/applications/{appUUID}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(appDeployments)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
data "coolify_deployments" "test" {
  application_uuid = "cccc0001-0001-4000-8000-000000000001"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_deployments.test", "deployments.#", "1"),
					resource.TestCheckResourceAttr("data.coolify_deployments.test", "deployments.0.uuid", "dep-app-001"),
					resource.TestCheckResourceAttr("data.coolify_deployments.test", "deployments.0.status", "finished"),
					resource.TestCheckResourceAttr("data.coolify_deployments.test", "deployments.0.server_uuid", "srv-002"),
				),
			},
		},
	})
}

func TestDeploymentsDataSource_Empty(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/deployments", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]interface{}{})
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
data "coolify_deployments" "test" {}
`,
				Check: resource.TestCheckResourceAttr("data.coolify_deployments.test", "deployments.#", "0"),
			},
		},
	})
}
