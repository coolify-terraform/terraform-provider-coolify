package application_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestApplicationsListDataSource(t *testing.T) {
	t.Parallel()
	apps := []client.Application{
		{
			UUID:          "app-list-uuid-1",
			Name:          "app-alpha",
			Domains:       "https://alpha.example.com",
			GitRepository: "https://github.com/example/alpha",
			GitBranch:     "main",
			BuildPack:     "nixpacks",
			Status:        "running",
		},
		{
			UUID:          "app-list-uuid-2",
			Name:          "app-beta",
			Domains:       "https://beta.example.com",
			GitRepository: "https://github.com/example/beta",
			GitBranch:     "develop",
			BuildPack:     "dockerfile",
			Status:        "stopped",
		},
	}

	mockSrv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method == http.MethodGet && r.URL.Path == "/api/v1/applications" {
			json.NewEncoder(w).Encode(apps)
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

data "coolify_applications" "test" {}
`, mockSrv.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_applications.test", "applications.#", "2"),
					resource.TestCheckResourceAttr("data.coolify_applications.test", "applications.0.uuid", "app-list-uuid-1"),
					resource.TestCheckResourceAttr("data.coolify_applications.test", "applications.0.name", "app-alpha"),
					resource.TestCheckResourceAttr("data.coolify_applications.test", "applications.0.domains", "https://alpha.example.com"),
					resource.TestCheckResourceAttr("data.coolify_applications.test", "applications.0.git_repository", "https://github.com/example/alpha"),
					resource.TestCheckResourceAttr("data.coolify_applications.test", "applications.0.git_branch", "main"),
					resource.TestCheckResourceAttr("data.coolify_applications.test", "applications.0.build_pack", "nixpacks"),
					resource.TestCheckResourceAttr("data.coolify_applications.test", "applications.0.status", "running"),
					resource.TestCheckResourceAttr("data.coolify_applications.test", "applications.1.uuid", "app-list-uuid-2"),
					resource.TestCheckResourceAttr("data.coolify_applications.test", "applications.1.name", "app-beta"),
					resource.TestCheckResourceAttr("data.coolify_applications.test", "applications.1.status", "stopped"),
				),
			},
			{
				Config: fmt.Sprintf(`
provider "coolify" {
  endpoint = %q
  token    = "test-token"
}

data "coolify_applications" "filtered" {
  filter {
    name   = "name"
    values = ["app-alpha"]
  }
}
`, mockSrv.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_applications.filtered", "applications.#", "1"),
					resource.TestCheckResourceAttr("data.coolify_applications.filtered", "applications.0.name", "app-alpha"),
				),
			},
		},
	})
}
