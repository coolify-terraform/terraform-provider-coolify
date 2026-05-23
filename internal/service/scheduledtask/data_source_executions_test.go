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

func TestTaskExecutionsDataSource_Application(t *testing.T) {
	t.Parallel()

	executions := []client.TaskExecution{
		{UUID: "te-1", Status: "success", Message: "completed ok", CreatedAt: "2024-01-15T10:00:00Z"},
		{UUID: "te-2", Status: "failed", Message: "exit code 1", CreatedAt: "2024-01-16T10:00:00Z"},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/applications/{appUUID}/scheduled-tasks/{taskUUID}/executions", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(executions)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
data "coolify_task_executions" "test" {
  application_uuid = "cccc0001-0001-4000-8000-000000000001"
  task_uuid        = "aaaa0001-0001-4000-8000-000000000099"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_task_executions.test", "executions.#", "2"),
					resource.TestCheckResourceAttr("data.coolify_task_executions.test", "executions.0.uuid", "te-1"),
					resource.TestCheckResourceAttr("data.coolify_task_executions.test", "executions.0.status", "success"),
					resource.TestCheckResourceAttr("data.coolify_task_executions.test", "executions.0.message", "completed ok"),
					resource.TestCheckResourceAttr("data.coolify_task_executions.test", "executions.0.created_at", "2024-01-15T10:00:00Z"),
					resource.TestCheckResourceAttr("data.coolify_task_executions.test", "executions.1.uuid", "te-2"),
					resource.TestCheckResourceAttr("data.coolify_task_executions.test", "executions.1.status", "failed"),
				),
			},
		},
	})
}

func TestTaskExecutionsDataSource_Service(t *testing.T) {
	t.Parallel()

	executions := []client.TaskExecution{
		{UUID: "te-s1", Status: "success", Message: "ok", CreatedAt: "2024-02-01T12:00:00Z"},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/services/{svcUUID}/scheduled-tasks/{taskUUID}/executions", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(executions)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
data "coolify_task_executions" "test" {
  service_uuid = "ffff0001-0001-4000-8000-000000000001"
  task_uuid    = "aaaa0002-0002-4000-8000-000000000099"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_task_executions.test", "executions.#", "1"),
					resource.TestCheckResourceAttr("data.coolify_task_executions.test", "executions.0.uuid", "te-s1"),
					resource.TestCheckResourceAttr("data.coolify_task_executions.test", "executions.0.status", "success"),
					resource.TestCheckResourceAttr("data.coolify_task_executions.test", "executions.0.message", "ok"),
					resource.TestCheckResourceAttr("data.coolify_task_executions.test", "executions.0.created_at", "2024-02-01T12:00:00Z"),
				),
			},
		},
	})
}
