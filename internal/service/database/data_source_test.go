package database_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/SebTardif/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestDatabaseDataSource(t *testing.T) {
	dbUUID := "db-ds-uuid-1"
	var publicPort int64 = 5432

	mockSrv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/"+dbUUID) {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"uuid":             dbUUID,
				"name":             "my-postgresql",
				"description":      "Production database",
				"type":             "postgresql",
				"image":            "postgres:16",
				"is_public":        true,
				"public_port":      publicPort,
				"server_uuid":      "srv-uuid-1",
				"project_uuid":     "proj-uuid-1",
				"environment_name": "production",
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
  endpoint = %q
  token    = "test-token"
}

data "coolify_database" "test" {
  uuid = %q
}
`, mockSrv.URL, dbUUID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_database.test", "uuid", dbUUID),
					resource.TestCheckResourceAttr("data.coolify_database.test", "name", "my-postgresql"),
					resource.TestCheckResourceAttr("data.coolify_database.test", "description", "Production database"),
					resource.TestCheckResourceAttr("data.coolify_database.test", "type", "postgresql"),
					resource.TestCheckResourceAttr("data.coolify_database.test", "image", "postgres:16"),
					resource.TestCheckResourceAttr("data.coolify_database.test", "is_public", "true"),
					resource.TestCheckResourceAttr("data.coolify_database.test", "public_port", "5432"),
					resource.TestCheckResourceAttr("data.coolify_database.test", "server_uuid", "srv-uuid-1"),
					resource.TestCheckResourceAttr("data.coolify_database.test", "project_uuid", "proj-uuid-1"),
					resource.TestCheckResourceAttr("data.coolify_database.test", "environment_name", "production"),
				),
			},
		},
	})
}
