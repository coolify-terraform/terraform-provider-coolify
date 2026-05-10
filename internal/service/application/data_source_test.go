package application_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SebTardif/terraform-provider-coolify/internal/acctest"
	"github.com/SebTardif/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestApplicationDataSource_Read(t *testing.T) {
	t.Parallel()
	app := client.Application{
		UUID:          "ds-app-uuid",
		Name:          "data-source-app",
		Description:   "a test app",
		GitRepository: "https://github.com/example/repo",
		GitBranch:     "main",
		BuildPack:     "nixpacks",
		PortsExposes:  "8080",
		ProjectUUID:   "aaaa0002-0002-4000-8000-000000000002",
		ServerUUID:    "bbbb0002-0002-4000-8000-000000000002",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(app)
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
provider "coolify" {
  endpoint  = %q
  token = "test-token"
}

data "coolify_application" "test" {
  uuid = "ds-app-uuid"
}
`, srv.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_application.test", "uuid", "ds-app-uuid"),
					resource.TestCheckResourceAttr("data.coolify_application.test", "name", "data-source-app"),
					resource.TestCheckResourceAttr("data.coolify_application.test", "description", "a test app"),
					resource.TestCheckResourceAttr("data.coolify_application.test", "git_repository", "https://github.com/example/repo"),
					resource.TestCheckResourceAttr("data.coolify_application.test", "git_branch", "main"),
					resource.TestCheckResourceAttr("data.coolify_application.test", "build_pack", "nixpacks"),
					resource.TestCheckResourceAttr("data.coolify_application.test", "ports_exposes", "8080"),
					resource.TestCheckResourceAttr("data.coolify_application.test", "project_uuid", "aaaa0002-0002-4000-8000-000000000002"),
					resource.TestCheckResourceAttr("data.coolify_application.test", "server_uuid", "bbbb0002-0002-4000-8000-000000000002"),
				),
			},
		},
	})
}
