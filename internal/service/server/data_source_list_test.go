package server_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestServersDataSource(t *testing.T) {
	t.Parallel()
	servers := []client.Server{
		{
			UUID:          "srv-list-uuid-1",
			Name:          "server-alpha",
			Description:   "First server",
			IP:            "10.0.0.1",
			Port:          22,
			User:          "root",
			IsBuildServer: false,
			IsReachable:   true,
			IsUsable:      true,
			Settings: &client.ServerSettings{
				ConcurrentBuilds:                     2,
				DynamicTimeout:                       3600,
				DeploymentQueueLimit:                 0,
				ConnectionTimeout:                    0,
				ServerDiskUsageNotificationThreshold: 80,
				ServerDiskUsageCheckFrequency:        "*/5 * * * *",
			},
		},
		{
			UUID:          "srv-list-uuid-2",
			Name:          "server-beta",
			Description:   "Second server",
			IP:            "10.0.0.2",
			Port:          2222,
			User:          "deploy",
			IsBuildServer: true,
			IsReachable:   true,
			IsUsable:      false,
			Settings: &client.ServerSettings{
				ConcurrentBuilds:                     4,
				DynamicTimeout:                       7200,
				DeploymentQueueLimit:                 10,
				ConnectionTimeout:                    45,
				ServerDiskUsageNotificationThreshold: 90,
				ServerDiskUsageCheckFrequency:        "0 * * * *",
			},
		},
	}

	mockSrv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method == http.MethodGet && r.URL.Path == "/api/v1/servers" {
			json.NewEncoder(w).Encode(servers)
			return
		}
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
	})))
	defer mockSrv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(mockSrv.URL) + `
data "coolify_servers" "test" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_servers.test", "servers.#", "2"),
					resource.TestCheckResourceAttr("data.coolify_servers.test", "servers.0.uuid", "srv-list-uuid-1"),
					resource.TestCheckResourceAttr("data.coolify_servers.test", "servers.0.name", "server-alpha"),
					resource.TestCheckResourceAttr("data.coolify_servers.test", "servers.0.ip", "10.0.0.1"),
					resource.TestCheckResourceAttr("data.coolify_servers.test", "servers.0.port", "22"),
					resource.TestCheckResourceAttr("data.coolify_servers.test", "servers.0.user", "root"),
					resource.TestCheckNoResourceAttr("data.coolify_servers.test", "servers.0.private_key_uuid"),
					resource.TestCheckResourceAttr("data.coolify_servers.test", "servers.0.is_build_server", "false"),
					resource.TestCheckResourceAttr("data.coolify_servers.test", "servers.0.is_reachable", "true"),
					resource.TestCheckResourceAttr("data.coolify_servers.test", "servers.1.uuid", "srv-list-uuid-2"),
					resource.TestCheckResourceAttr("data.coolify_servers.test", "servers.1.name", "server-beta"),
					resource.TestCheckResourceAttr("data.coolify_servers.test", "servers.1.ip", "10.0.0.2"),
					resource.TestCheckResourceAttr("data.coolify_servers.test", "servers.1.port", "2222"),
					resource.TestCheckResourceAttr("data.coolify_servers.test", "servers.1.user", "deploy"),
					resource.TestCheckNoResourceAttr("data.coolify_servers.test", "servers.1.private_key_uuid"),
					resource.TestCheckResourceAttr("data.coolify_servers.test", "servers.1.is_build_server", "true"),
					resource.TestCheckResourceAttr("data.coolify_servers.test", "servers.1.is_usable", "false"),
					resource.TestCheckResourceAttr("data.coolify_servers.test", "servers.0.concurrent_builds", "2"),
					resource.TestCheckResourceAttr("data.coolify_servers.test", "servers.0.dynamic_timeout", "3600"),
					resource.TestCheckResourceAttr("data.coolify_servers.test", "servers.0.deployment_queue_limit", "0"),
					resource.TestCheckResourceAttr("data.coolify_servers.test", "servers.0.connection_timeout", "10"),
					resource.TestCheckResourceAttr("data.coolify_servers.test", "servers.0.server_disk_usage_notification_threshold", "80"),
					resource.TestCheckResourceAttr("data.coolify_servers.test", "servers.0.server_disk_usage_check_frequency", "*/5 * * * *"),
					resource.TestCheckResourceAttr("data.coolify_servers.test", "servers.1.concurrent_builds", "4"),
					resource.TestCheckResourceAttr("data.coolify_servers.test", "servers.1.dynamic_timeout", "7200"),
					resource.TestCheckResourceAttr("data.coolify_servers.test", "servers.1.deployment_queue_limit", "10"),
					resource.TestCheckResourceAttr("data.coolify_servers.test", "servers.1.connection_timeout", "45"),
					resource.TestCheckResourceAttr("data.coolify_servers.test", "servers.1.server_disk_usage_notification_threshold", "90"),
					resource.TestCheckResourceAttr("data.coolify_servers.test", "servers.1.server_disk_usage_check_frequency", "0 * * * *"),
				),
			},
			{
				Config: acctest.ProviderBlockForURL(mockSrv.URL) + `
data "coolify_servers" "filtered" {
  filter {
    name   = "name"
    values = ["server-alpha"]
  }
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_servers.filtered", "servers.#", "1"),
					resource.TestCheckResourceAttr("data.coolify_servers.filtered", "servers.0.name", "server-alpha"),
					resource.TestCheckResourceAttr("data.coolify_servers.filtered", "servers.0.ip", "10.0.0.1"),
				),
			},
		},
	})
}

func TestServersDataSource_APIError(t *testing.T) {
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
data "coolify_servers" "test" {}
`,
				ExpectError: regexp.MustCompile(`Error listing servers`),
			},
		},
	})
}
