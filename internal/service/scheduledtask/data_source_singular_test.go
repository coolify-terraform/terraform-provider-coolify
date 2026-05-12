package scheduledtask_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"regexp"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/acctest"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestScheduledTaskDataSource_Application(t *testing.T) {
	t.Parallel()

	tasks := []client.ScheduledTask{
		{UUID: "st-1", Name: "backup-db", Command: "pg_dump mydb", Frequency: "0 0 * * *", Enabled: true},
		{UUID: "st-2", Name: "cleanup-logs", Command: "rm -rf /tmp/logs/*", Frequency: "0 */6 * * *", Enabled: false},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/applications/{appUUID}/scheduled-tasks", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tasks)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
data "coolify_scheduled_task" "test" {
  uuid             = "st-2"
  application_uuid = "cccc0001-0001-4000-8000-000000000001"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_scheduled_task.test", "uuid", "st-2"),
					resource.TestCheckResourceAttr("data.coolify_scheduled_task.test", "name", "cleanup-logs"),
					resource.TestCheckResourceAttr("data.coolify_scheduled_task.test", "command", "rm -rf /tmp/logs/*"),
					resource.TestCheckResourceAttr("data.coolify_scheduled_task.test", "frequency", "0 */6 * * *"),
					resource.TestCheckResourceAttr("data.coolify_scheduled_task.test", "enabled", "false"),
				),
			},
		},
	})
}

func TestScheduledTaskDataSource_Service(t *testing.T) {
	t.Parallel()

	tasks := []client.ScheduledTask{
		{UUID: "st-s1", Name: "health-check", Command: "curl http://localhost/health", Frequency: "*/5 * * * *", Enabled: true},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/services/{svcUUID}/scheduled-tasks", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tasks)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
data "coolify_scheduled_task" "test" {
  uuid         = "st-s1"
  service_uuid = "ffff0001-0001-4000-8000-000000000001"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_scheduled_task.test", "uuid", "st-s1"),
					resource.TestCheckResourceAttr("data.coolify_scheduled_task.test", "name", "health-check"),
					resource.TestCheckResourceAttr("data.coolify_scheduled_task.test", "command", "curl http://localhost/health"),
					resource.TestCheckResourceAttr("data.coolify_scheduled_task.test", "frequency", "*/5 * * * *"),
					resource.TestCheckResourceAttr("data.coolify_scheduled_task.test", "enabled", "true"),
				),
			},
		},
	})
}

func TestScheduledTaskDataSource_NotFound(t *testing.T) {
	t.Parallel()

	tasks := []client.ScheduledTask{
		{UUID: "st-1", Name: "backup-db", Command: "pg_dump mydb", Frequency: "0 0 * * *", Enabled: true},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/applications/{appUUID}/scheduled-tasks", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tasks)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
data "coolify_scheduled_task" "test" {
  uuid             = "nonexistent-uuid"
  application_uuid = "cccc0001-0001-4000-8000-000000000001"
}
`,
				ExpectError: regexp.MustCompile(`not found`),
			},
		},
	})
}
