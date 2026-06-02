package team_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestTeamsListDataSource(t *testing.T) {
	t.Parallel()

	mockSrv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method == http.MethodGet && r.URL.Path == "/api/v1/teams" {
			json.NewEncoder(w).Encode([]map[string]interface{}{
				{
					"id":          1,
					"name":        "Engineering",
					"description": "The engineering team",
				},
				{
					"id":          2,
					"name":        "Design",
					"description": "The design team",
				},
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
  endpoint  = %q
  token = "test-token"
}

data "coolify_teams" "test" {}
`, mockSrv.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_teams.test", "teams.#", "2"),
					resource.TestCheckResourceAttr("data.coolify_teams.test", "teams.0.id", "1"),
					resource.TestCheckResourceAttr("data.coolify_teams.test", "teams.0.name", "Engineering"),
					resource.TestCheckResourceAttr("data.coolify_teams.test", "teams.0.description", "The engineering team"),
					resource.TestCheckResourceAttr("data.coolify_teams.test", "teams.1.id", "2"),
					resource.TestCheckResourceAttr("data.coolify_teams.test", "teams.1.name", "Design"),
					resource.TestCheckResourceAttr("data.coolify_teams.test", "teams.1.description", "The design team"),
				),
			},
			{
				Config: fmt.Sprintf(`
provider "coolify" {
  endpoint  = %q
  token = "test-token"
}

data "coolify_teams" "filtered" {
  filter {
    name   = "name"
    values = ["Design"]
  }
}
`, mockSrv.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_teams.filtered", "teams.#", "1"),
					resource.TestCheckResourceAttr("data.coolify_teams.filtered", "teams.0.name", "Design"),
				),
			},
		},
	})
}

func TestTeamsListDataSource_APIError(t *testing.T) {
	t.Parallel()
	mockSrv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
	})))
	defer mockSrv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(mockSrv.URL) + `
data "coolify_teams" "test" {}
`,
				ExpectError: regexp.MustCompile(`Error listing teams`),
			},
		},
	})
}
