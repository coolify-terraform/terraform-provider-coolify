package backup_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestBackupExecutionsDataSource(t *testing.T) {
	t.Parallel()

	executions := []client.BackupExecution{
		{UUID: "exec-001", Status: "success", CreatedAt: "2024-01-15T02:00:00Z", Size: 1048576},
		{UUID: "exec-002", Status: "failed", CreatedAt: "2024-01-16T02:00:00Z", Size: 0},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/databases/{dbUUID}/backups/{backupUUID}/executions", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"executions": executions})
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
data "coolify_backup_executions" "test" {
  database_uuid = "eeee0001-0001-4000-8000-000000000001"
  backup_uuid   = "ffff0001-0001-4000-8000-000000000001"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_backup_executions.test", "executions.#", "2"),
					resource.TestCheckResourceAttr("data.coolify_backup_executions.test", "executions.0.uuid", "exec-001"),
					resource.TestCheckResourceAttr("data.coolify_backup_executions.test", "executions.0.status", "success"),
					resource.TestCheckResourceAttr("data.coolify_backup_executions.test", "executions.0.created_at", "2024-01-15T02:00:00Z"),
					resource.TestCheckResourceAttr("data.coolify_backup_executions.test", "executions.0.size", "1048576"),
					resource.TestCheckResourceAttr("data.coolify_backup_executions.test", "executions.1.uuid", "exec-002"),
					resource.TestCheckResourceAttr("data.coolify_backup_executions.test", "executions.1.status", "failed"),
				),
			},
		},
	})
}
