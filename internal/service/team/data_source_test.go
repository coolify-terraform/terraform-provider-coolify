package team_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SebTardif/terraform-provider-coolify/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testProtoV6ProviderFactories() map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"coolify": providerserver.NewProtocol6WithError(provider.New("test")()),
	}
}

func newMockTeamServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/teams/1":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":          1,
				"name":        "Engineering",
				"description": "The engineering team",
			})

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/teams/1/members":
			json.NewEncoder(w).Encode([]map[string]interface{}{
				{
					"id":    10,
					"name":  "Alice",
					"email": "alice@example.com",
				},
				{
					"id":    20,
					"name":  "Bob",
					"email": "bob@example.com",
				},
			})

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestTeamDataSource_Read(t *testing.T) {
	srv := newMockTeamServer()
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
provider "coolify" {
  endpoint  = %q
  token = "test-token"
}

data "coolify_team" "test" {
  id = 1
}
`, srv.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_team.test", "id", "1"),
					resource.TestCheckResourceAttr("data.coolify_team.test", "name", "Engineering"),
					resource.TestCheckResourceAttr("data.coolify_team.test", "description", "The engineering team"),
					resource.TestCheckResourceAttr("data.coolify_team.test", "members.#", "2"),
					resource.TestCheckResourceAttr("data.coolify_team.test", "members.0.id", "10"),
					resource.TestCheckResourceAttr("data.coolify_team.test", "members.0.name", "Alice"),
					resource.TestCheckResourceAttr("data.coolify_team.test", "members.0.email", "alice@example.com"),
					resource.TestCheckResourceAttr("data.coolify_team.test", "members.1.id", "20"),
					resource.TestCheckResourceAttr("data.coolify_team.test", "members.1.name", "Bob"),
					resource.TestCheckResourceAttr("data.coolify_team.test", "members.1.email", "bob@example.com"),
				),
			},
		},
	})
}
