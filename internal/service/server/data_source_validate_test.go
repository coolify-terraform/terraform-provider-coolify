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

func TestServerValidationDataSource(t *testing.T) {
	t.Parallel()
	validation := client.ServerValidation{
		Valid:   true,
		Message: "Server is reachable and operational",
	}

	mockSrv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/validate") {
			json.NewEncoder(w).Encode(validation)
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
data "coolify_server_validation" "test" {
  uuid = "550e8400-e29b-41d4-a716-446655440000"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_server_validation.test", "valid", "true"),
					resource.TestCheckResourceAttr("data.coolify_server_validation.test", "message", "Server is reachable and operational"),
				),
			},
		},
	})
}
