package server_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SebTardif/terraform-provider-coolify/internal/acctest"
	"github.com/SebTardif/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestServersDataSource(t *testing.T) {
	servers := []client.Server{
		{
			UUID:           "srv-list-uuid-1",
			Name:           "server-alpha",
			Description:    "First server",
			IP:             "10.0.0.1",
			Port:           22,
			User:           "root",
			PrivateKeyUUID: "pk-1",
			IsBuildServer:  false,
			IsReachable:    true,
			IsUsable:       true,
		},
		{
			UUID:           "srv-list-uuid-2",
			Name:           "server-beta",
			Description:    "Second server",
			IP:             "10.0.0.2",
			Port:           2222,
			User:           "deploy",
			PrivateKeyUUID: "pk-2",
			IsBuildServer:  true,
			IsReachable:    true,
			IsUsable:       false,
		},
	}

	mockSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method == http.MethodGet && r.URL.Path == "/api/v1/servers" {
			json.NewEncoder(w).Encode(servers)
			return
		}
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
	}))
	defer mockSrv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderBlock(mockSrv.URL) + `
data "coolify_servers" "test" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_servers.test", "servers.#", "2"),
					resource.TestCheckResourceAttr("data.coolify_servers.test", "servers.0.uuid", "srv-list-uuid-1"),
					resource.TestCheckResourceAttr("data.coolify_servers.test", "servers.0.name", "server-alpha"),
					resource.TestCheckResourceAttr("data.coolify_servers.test", "servers.0.ip", "10.0.0.1"),
					resource.TestCheckResourceAttr("data.coolify_servers.test", "servers.0.port", "22"),
					resource.TestCheckResourceAttr("data.coolify_servers.test", "servers.0.user", "root"),
					resource.TestCheckResourceAttr("data.coolify_servers.test", "servers.0.is_build_server", "false"),
					resource.TestCheckResourceAttr("data.coolify_servers.test", "servers.0.is_reachable", "true"),
					resource.TestCheckResourceAttr("data.coolify_servers.test", "servers.1.uuid", "srv-list-uuid-2"),
					resource.TestCheckResourceAttr("data.coolify_servers.test", "servers.1.name", "server-beta"),
					resource.TestCheckResourceAttr("data.coolify_servers.test", "servers.1.ip", "10.0.0.2"),
					resource.TestCheckResourceAttr("data.coolify_servers.test", "servers.1.port", "2222"),
					resource.TestCheckResourceAttr("data.coolify_servers.test", "servers.1.user", "deploy"),
					resource.TestCheckResourceAttr("data.coolify_servers.test", "servers.1.is_build_server", "true"),
					resource.TestCheckResourceAttr("data.coolify_servers.test", "servers.1.is_usable", "false"),
				),
			},
		},
	})
}
