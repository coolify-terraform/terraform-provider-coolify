package scheduledtask_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/acctest"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestScheduledTasksDataSource_Application(t *testing.T) {
	t.Parallel()

	tasks := []client.ScheduledTask{
		{UUID: "st-1", Name: "backup-db", Command: "pg_dump mydb", Frequency: "0 0 * * *", Enabled: true},
		{UUID: "st-2", Name: "cleanup-logs", Command: "rm -rf /tmp/logs/*", Frequency: "0 */6 * * *", Enabled: false},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/applications/{appUUID}/scheduled-tasks", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("appUUID") != "cccc0001-0001-4000-8000-000000000001" {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
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
data "coolify_scheduled_tasks" "test" {
  application_uuid = "cccc0001-0001-4000-8000-000000000001"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_scheduled_tasks.test", "tasks.#", "2"),
					resource.TestCheckResourceAttr("data.coolify_scheduled_tasks.test", "tasks.0.name", "backup-db"),
					resource.TestCheckResourceAttr("data.coolify_scheduled_tasks.test", "tasks.0.command", "pg_dump mydb"),
					resource.TestCheckResourceAttr("data.coolify_scheduled_tasks.test", "tasks.0.frequency", "0 0 * * *"),
					resource.TestCheckResourceAttr("data.coolify_scheduled_tasks.test", "tasks.0.enabled", "true"),
					resource.TestCheckResourceAttr("data.coolify_scheduled_tasks.test", "tasks.1.name", "cleanup-logs"),
					resource.TestCheckResourceAttr("data.coolify_scheduled_tasks.test", "tasks.1.enabled", "false"),
				),
			},
		},
	})
}

func TestScheduledTasksDataSource_Service(t *testing.T) {
	t.Parallel()

	tasks := []client.ScheduledTask{
		{UUID: "st-s1", Name: "health-check", Command: "curl http://localhost/health", Frequency: "*/5 * * * *", Enabled: true},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/services/{svcUUID}/scheduled-tasks", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("svcUUID") != "ffff0001-0001-4000-8000-000000000001" {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
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
data "coolify_scheduled_tasks" "test" {
  service_uuid = "ffff0001-0001-4000-8000-000000000001"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_scheduled_tasks.test", "tasks.#", "1"),
					resource.TestCheckResourceAttr("data.coolify_scheduled_tasks.test", "tasks.0.name", "health-check"),
					resource.TestCheckResourceAttr("data.coolify_scheduled_tasks.test", "tasks.0.command", "curl http://localhost/health"),
					resource.TestCheckResourceAttr("data.coolify_scheduled_tasks.test", "tasks.0.frequency", "*/5 * * * *"),
					resource.TestCheckResourceAttr("data.coolify_scheduled_tasks.test", "tasks.0.enabled", "true"),
				),
			},
		},
	})
}
