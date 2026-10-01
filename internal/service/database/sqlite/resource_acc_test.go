package sqlite_test

import (
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccSqliteDatabaseResource_CRUD(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)
	skipIfNoSQLiteAPI(t)
	serverUUID := acctest.AccTestServerUUID(t)
	name := acctest.RandomWithPrefix("tf-acc-sqlite")

	createExtra := `
  sqlite_databases = "app.sqlite"
  limits_memory = "256M"
`
	updateExtra := `
  sqlite_databases = "app.sqlite"
  limits_memory = "512M"
`
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.AccCheckDestroy("coolify_database_sqlite", "/api/v1/databases/"),
		Steps: []resource.TestStep{
			{
				Config: acctest.AccTestDatabaseConfig("coolify_database_sqlite", name, serverUUID, createExtra),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_database_sqlite.test", "sqlite_databases", "app.sqlite"),
					resource.TestCheckResourceAttr("coolify_database_sqlite.test", "limits_memory", "256M"),
				),
			},
			{
				Config: acctest.AccTestDatabaseConfig("coolify_database_sqlite", name, serverUUID, updateExtra),
				Check:  resource.TestCheckResourceAttr("coolify_database_sqlite.test", "limits_memory", "512M"),
			},
			{
				ResourceName:                         "coolify_database_sqlite.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateIdFunc:                    acctest.ImportStateIDFunc("coolify_database_sqlite.test", "uuid"),
				ImportStateVerifyIgnore:              []string{"instant_deploy", "project_uuid", "server_uuid", "environment_name"},
			},
		},
	})
}

// skipIfNoSQLiteAPI skips when POST /databases/sqlite is not routed.
// Coolify main added the route on 2026-09-18. v4.3.23 and v4.4-rc.1 do not
// have it. A missing route soft-skips even when COOLIFY_REQUIRE_TIP_APIS=1
// because the acceptance image can still be behind that commit while its
// /version string says 4.3.0.
func skipIfNoSQLiteAPI(t *testing.T) {
	t.Helper()
	endpoint := strings.TrimRight(os.Getenv("COOLIFY_ENDPOINT"), "/")
	req, err := http.NewRequest(http.MethodPost, endpoint+"/api/v1/databases/sqlite", strings.NewReader(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+os.Getenv("COOLIFY_TOKEN"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Skipf("sqlite probe failed: %v", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusMethodNotAllowed {
		t.Skipf("Coolify has no POST /databases/sqlite (HTTP %d): %s", resp.StatusCode, raw)
	}
}
