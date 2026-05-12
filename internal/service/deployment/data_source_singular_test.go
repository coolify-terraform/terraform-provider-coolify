package deployment_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"regexp"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestDeploymentDataSource_Read(t *testing.T) {
	t.Parallel()

	dep := map[string]interface{}{
		"deployment_uuid": "dep-001",
		"status":          "finished",
		"server_uuid":     "srv-001",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/deployments/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != "dep-001" {
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
  uuid = "dep-001"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_deployment.test", "uuid", "dep-001"),
					resource.TestCheckResourceAttr("data.coolify_deployment.test", "status", "finished"),
					resource.TestCheckResourceAttr("data.coolify_deployment.test", "server_uuid", "srv-001"),
				),
			},
		},
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
  uuid = "nonexistent-uuid"
}
`,
				ExpectError: regexp.MustCompile(`Error reading deployment`),
			},
		},
	})
}
