package server_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/acctest"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestServerDataSource(t *testing.T) {
	t.Parallel()
	srv := &client.Server{
		UUID:          "cccc0002-0002-4000-8000-000000000001",
		Name:          "data-source-server",
		Description:   "A server for testing",
		IP:            "192.168.1.100",
		Port:          22,
		User:          "root",
		IsBuildServer: false,
		IsReachable:   true,
		IsUsable:      true,
		Settings: &client.ServerSettings{
			ConcurrentBuilds:                     2,
			DynamicTimeout:                       3600,
			DeploymentQueueLimit:                 0,
			ServerDiskUsageNotificationThreshold: 80,
			ServerDiskUsageCheckFrequency:        "*/5 * * * *",
		},
	}

	mockSrv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v1/servers/") {
			uuid := strings.TrimPrefix(r.URL.Path, "/api/v1/servers/")
			if uuid == srv.UUID {
				json.NewEncoder(w).Encode(srv)
				return
			}
		}
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
	})))
	defer mockSrv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(mockSrv.URL) + `
data "coolify_server" "test" {
  uuid = "cccc0002-0002-4000-8000-000000000001"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_server.test", "uuid", "cccc0002-0002-4000-8000-000000000001"),
					resource.TestCheckResourceAttr("data.coolify_server.test", "name", "data-source-server"),
					resource.TestCheckResourceAttr("data.coolify_server.test", "description", "A server for testing"),
					resource.TestCheckResourceAttr("data.coolify_server.test", "ip", "192.168.1.100"),
					resource.TestCheckResourceAttr("data.coolify_server.test", "port", "22"),
					resource.TestCheckResourceAttr("data.coolify_server.test", "user", "root"),
					resource.TestCheckNoResourceAttr("data.coolify_server.test", "private_key_uuid"),
					resource.TestCheckResourceAttr("data.coolify_server.test", "is_build_server", "false"),
					resource.TestCheckResourceAttr("data.coolify_server.test", "is_reachable", "true"),
					resource.TestCheckResourceAttr("data.coolify_server.test", "is_usable", "true"),
					resource.TestCheckResourceAttr("data.coolify_server.test", "concurrent_builds", "2"),
					resource.TestCheckResourceAttr("data.coolify_server.test", "dynamic_timeout", "3600"),
					resource.TestCheckResourceAttr("data.coolify_server.test", "deployment_queue_limit", "0"),
					resource.TestCheckResourceAttr("data.coolify_server.test", "server_disk_usage_notification_threshold", "80"),
					resource.TestCheckResourceAttr("data.coolify_server.test", "server_disk_usage_check_frequency", "*/5 * * * *"),
				),
			},
		},
	})
}

func TestServerDataSource_NotFound(t *testing.T) {
	t.Parallel()
	mockSrv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
	})))
	defer mockSrv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(mockSrv.URL) + `
data "coolify_server" "test" {
  uuid = "00000000-0000-4000-8000-000000000000"
}`,
				ExpectError: regexp.MustCompile(`Error reading server`),
			},
		},
	})
}
