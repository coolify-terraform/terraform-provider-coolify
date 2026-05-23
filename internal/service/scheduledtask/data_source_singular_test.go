package scheduledtask_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestScheduledTaskDataSource_Application(t *testing.T) {
	t.Parallel()

	tasks := []client.ScheduledTask{
		{UUID: "11111111-1111-4111-8111-111111111111", Name: "backup-db", Command: "pg_dump mydb", Frequency: "0 0 * * *", Enabled: true},
		{UUID: "22222222-2222-4222-8222-222222222222", Name: "cleanup-logs", Command: "rm -rf /tmp/logs/*", Frequency: "0 */6 * * *", Enabled: false},
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
  uuid             = "22222222-2222-4222-8222-222222222222"
  application_uuid = "cccc0001-0001-4000-8000-000000000001"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_scheduled_task.test", "uuid", "22222222-2222-4222-8222-222222222222"),
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
		{UUID: "33333333-3333-4333-8333-333333333333", Name: "health-check", Command: "curl http://localhost/health", Frequency: "*/5 * * * *", Enabled: true},
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
  uuid         = "33333333-3333-4333-8333-333333333333"
  service_uuid = "ffff0001-0001-4000-8000-000000000001"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_scheduled_task.test", "uuid", "33333333-3333-4333-8333-333333333333"),
					resource.TestCheckResourceAttr("data.coolify_scheduled_task.test", "name", "health-check"),
					resource.TestCheckResourceAttr("data.coolify_scheduled_task.test", "command", "curl http://localhost/health"),
					resource.TestCheckResourceAttr("data.coolify_scheduled_task.test", "frequency", "*/5 * * * *"),
					resource.TestCheckResourceAttr("data.coolify_scheduled_task.test", "enabled", "true"),
				),
			},
		},
	})
}

func TestScheduledTaskDataSource_InvalidUUID(t *testing.T) {
	t.Parallel()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: `data "coolify_scheduled_task" "test" {
  uuid             = "not-a-valid-uuid"
  application_uuid = "cccc0001-0001-4000-8000-000000000001"
}
`,
			ExpectError: acctest.UUIDValidationError(),
		}},
	})
}

func TestScheduledTaskDataSource_NotFound(t *testing.T) {
	t.Parallel()

	tasks := []client.ScheduledTask{
		{UUID: "11111111-1111-4111-8111-111111111111", Name: "backup-db", Command: "pg_dump mydb", Frequency: "0 0 * * *", Enabled: true},
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
  uuid             = "55555555-5555-4555-8555-555555555555"
  application_uuid = "cccc0001-0001-4000-8000-000000000001"
}
`,
				ExpectError: acctest.NotFoundError(),
			},
		},
	})
}
