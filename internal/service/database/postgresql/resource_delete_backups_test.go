package postgresql_test

import (
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/service/database/dbtest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestPostgresqlDatabaseResource_DeleteBackupsTrue(t *testing.T) {
	t.Parallel()
	srv, state := dbtest.NewMockServerVersion("postgresql", "pg-test-db", "postgres:16", map[string]interface{}{
		"postgres_user":     "postgres",
		"postgres_password": "secret123",
		"postgres_db":       "defaultdb",
	}, "4.8.0")
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.CheckDestroy(srv.URL, "coolify_database_postgresql", "/api/v1/databases/"),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_database_postgresql" "test" {
  project_uuid   = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid    = "bbbb0001-0001-4000-8000-000000000001"
  delete_backups = true
}
`,
				Check: resource.TestCheckResourceAttr("coolify_database_postgresql.test", "delete_backups", "true"),
			},
		},
	})

	if state.DeleteQuery() != "delete_backups=true" {
		t.Fatalf("DELETE query = %q", state.DeleteQuery())
	}
}
