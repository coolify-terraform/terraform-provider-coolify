package team_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestTeamMembersDataSource(t *testing.T) {
	t.Parallel()

	mockSrv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method == http.MethodGet && r.URL.Path == "/api/v1/teams/1/members" {
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

data "coolify_team_members" "test" {
  id = 1
}
`, mockSrv.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_team_members.test", "members.#", "2"),
					resource.TestCheckResourceAttr("data.coolify_team_members.test", "members.0.id", "10"),
					resource.TestCheckResourceAttr("data.coolify_team_members.test", "members.0.name", "Alice"),
					resource.TestCheckResourceAttr("data.coolify_team_members.test", "members.0.email", "alice@example.com"),
					resource.TestCheckResourceAttr("data.coolify_team_members.test", "members.1.id", "20"),
					resource.TestCheckResourceAttr("data.coolify_team_members.test", "members.1.name", "Bob"),
					resource.TestCheckResourceAttr("data.coolify_team_members.test", "members.1.email", "bob@example.com"),
				),
			},
		},
	})
}

func TestTeamMembersDataSource_Current(t *testing.T) {
	t.Parallel()

	mockSrv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method == http.MethodGet && r.URL.Path == "/api/v1/teams/current/members" {
			json.NewEncoder(w).Encode([]map[string]interface{}{
				{
					"id":    30,
					"name":  "Charlie",
					"email": "charlie@example.com",
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

data "coolify_team_members" "current" {}
`, mockSrv.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_team_members.current", "members.#", "1"),
					resource.TestCheckResourceAttr("data.coolify_team_members.current", "members.0.id", "30"),
					resource.TestCheckResourceAttr("data.coolify_team_members.current", "members.0.name", "Charlie"),
					resource.TestCheckResourceAttr("data.coolify_team_members.current", "members.0.email", "charlie@example.com"),
				),
			},
		},
	})
}
