package database_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/acctest"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestDatabasesListDataSource(t *testing.T) {
	t.Parallel()
	databases := []client.Database{
		{
			UUID:        "db-list-uuid-1",
			Name:        "db-alpha",
			Description: "First database",
			Type:        "postgresql",
			Image:       "postgres:16",
			IsPublic:    false,
		},
		{
			UUID:        "db-list-uuid-2",
			Name:        "db-beta",
			Description: "Second database",
			Type:        "mysql",
			Image:       "mysql:8",
			IsPublic:    true,
		},
	}

	mockSrv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method == http.MethodGet && r.URL.Path == "/api/v1/databases" {
			json.NewEncoder(w).Encode(databases)
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

data "coolify_databases" "test" {}
`, mockSrv.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_databases.test", "databases.#", "2"),
					resource.TestCheckResourceAttr("data.coolify_databases.test", "databases.0.uuid", "db-list-uuid-1"),
					resource.TestCheckResourceAttr("data.coolify_databases.test", "databases.0.name", "db-alpha"),
					resource.TestCheckResourceAttr("data.coolify_databases.test", "databases.0.description", "First database"),
					resource.TestCheckResourceAttr("data.coolify_databases.test", "databases.0.type", "postgresql"),
					resource.TestCheckResourceAttr("data.coolify_databases.test", "databases.0.image", "postgres:16"),
					resource.TestCheckResourceAttr("data.coolify_databases.test", "databases.0.is_public", "false"),
					resource.TestCheckResourceAttr("data.coolify_databases.test", "databases.1.uuid", "db-list-uuid-2"),
					resource.TestCheckResourceAttr("data.coolify_databases.test", "databases.1.name", "db-beta"),
					resource.TestCheckResourceAttr("data.coolify_databases.test", "databases.1.type", "mysql"),
					resource.TestCheckResourceAttr("data.coolify_databases.test", "databases.1.image", "mysql:8"),
					resource.TestCheckResourceAttr("data.coolify_databases.test", "databases.1.is_public", "true"),
				),
			},
		},
	})
}
