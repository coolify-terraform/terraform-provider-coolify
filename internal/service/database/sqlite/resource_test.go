package sqlite_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/service/database/dbtest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

const sqliteImage = "peakimages/sqlite:3.53.4-v0.1.0"

func TestSqliteDatabaseResource_CreateUpdateImport(t *testing.T) {
	t.Parallel()
	srv, st := dbtest.NewMockServerVersion("sqlite", "sqlite-test", sqliteImage, map[string]interface{}{
		"sqlite_databases": "database.sqlite",
	}, "4.4.0")
	defer srv.Close()

	config := func(extra string) string {
		return acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_database_sqlite" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
` + extra + "}\n"
	}

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.CheckDestroy(srv.URL, "coolify_database_sqlite", "/api/v1/databases/"),
		Steps: []resource.TestStep{
			{
				Config: config(`  sqlite_databases = "app.sqlite,cache.sqlite"` + "\n"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_database_sqlite.test", "uuid", "aaaa0001-0001-4000-8000-000000000001"),
					resource.TestCheckResourceAttr("coolify_database_sqlite.test", "sqlite_databases", "app.sqlite,cache.sqlite"),
					resource.TestCheckResourceAttr("coolify_database_sqlite.test", "environment_name", "production"),
					resource.TestCheckResourceAttr("coolify_database_sqlite.test", "is_public", "false"),
					func(*terraform.State) error {
						body := st.CreateBody()
						if _, ok := body["is_public"]; ok {
							return fmt.Errorf("POST included is_public: %v", body)
						}
						if _, ok := body["public_port"]; ok {
							return fmt.Errorf("POST included public_port: %v", body)
						}
						if body["sqlite_databases"] != "app.sqlite,cache.sqlite" {
							return fmt.Errorf("sqlite_databases = %v", body["sqlite_databases"])
						}
						return nil
					},
				),
			},
			{Config: config(`  sqlite_databases = "app.sqlite,cache.sqlite"` + "\n"), PlanOnly: true, ExpectNonEmptyPlan: false},
			{
				Config: config("  sqlite_databases = \"app.sqlite\"\n  description = \"files\"\n"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_database_sqlite.test", "sqlite_databases", "app.sqlite"),
					resource.TestCheckResourceAttr("coolify_database_sqlite.test", "description", "files"),
					func(*terraform.State) error {
						body := st.PatchBody()
						if _, ok := body["is_public"]; ok {
							return fmt.Errorf("PATCH included is_public: %v", body)
						}
						if body["sqlite_databases"] != "app.sqlite" {
							return fmt.Errorf("PATCH sqlite_databases = %v", body["sqlite_databases"])
						}
						return nil
					},
				),
			},
			{
				ResourceName:                         "coolify_database_sqlite.test",
				ImportState:                          true,
				ImportStateId:                        "aaaa0001-0001-4000-8000-000000000001",
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateVerifyIgnore:              []string{"instant_deploy"},
			},
		},
	})
}

func TestSqliteDatabaseResource_RejectsOldCoolify(t *testing.T) {
	t.Parallel()
	for _, ver := range []string{"4.3.23", "4.4-rc.1"} {
		ver := ver
		t.Run(ver, func(t *testing.T) {
			t.Parallel()
			srv, _ := dbtest.NewMockServerVersion("sqlite", "sqlite-test", sqliteImage, nil, ver)
			defer srv.Close()
			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
				Steps: []resource.TestStep{{
					Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_database_sqlite" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
}
`,
					ExpectError: regexp.MustCompile(`requires Coolify 4.4`),
				}},
			})
		})
	}
}

func TestSqliteDatabaseResource_RejectsPublicPort(t *testing.T) {
	t.Parallel()
	srv, _ := dbtest.NewMockServerVersion("sqlite", "sqlite-test", sqliteImage, nil, "4.4.0")
	defer srv.Close()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_database_sqlite" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid  = "bbbb0001-0001-4000-8000-000000000001"
  is_public    = true
  public_port  = 5432
}
`,
			ExpectError: regexp.MustCompile(`not published on a host port`),
		}},
	})
}
