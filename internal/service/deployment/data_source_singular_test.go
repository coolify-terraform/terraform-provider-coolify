package deployment_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"regexp"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestDeploymentDataSource_Read(t *testing.T) {
	t.Parallel()

	dep := map[string]interface{}{
		"deployment_uuid": "11111111-1111-4111-8111-111111111111",
		"status":          "finished",
		"server_uuid":     "22222222-2222-4222-8222-222222222222",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/deployments/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != "11111111-1111-4111-8111-111111111111" {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(dep)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
data "coolify_deployment" "test" {
  uuid = "11111111-1111-4111-8111-111111111111"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_deployment.test", "uuid", "11111111-1111-4111-8111-111111111111"),
					resource.TestCheckResourceAttr("data.coolify_deployment.test", "status", "finished"),
					resource.TestCheckResourceAttr("data.coolify_deployment.test", "server_uuid", "22222222-2222-4222-8222-222222222222"),
				),
			},
		},
	})
}

func TestDeploymentDataSource_InvalidUUID(t *testing.T) {
	t.Parallel()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: `data "coolify_deployment" "test" {
  uuid = "not-a-valid-uuid"
}
`,
			ExpectError: acctest.UUIDValidationError(),
		}},
	})
}

func TestDeploymentDataSource_NotFound(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/deployments/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
data "coolify_deployment" "test" {
  uuid = "55555555-5555-4555-8555-555555555555"
}
`,
				ExpectError: regexp.MustCompile(`Error reading deployment`),
			},
		},
	})
}
