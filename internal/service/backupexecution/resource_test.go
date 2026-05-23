package backupexecution_test

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync/atomic"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestBackupExecutionResource_Create(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /api/v1/databases/550e8400-e29b-41d4-a716-446655440001/backups/550e8400-e29b-41d4-a716-446655440002/executions/550e8400-e29b-41d4-a716-446655440003", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_backup_execution", "test", `
					database_uuid  = "550e8400-e29b-41d4-a716-446655440001"
					backup_uuid    = "550e8400-e29b-41d4-a716-446655440002"
					execution_uuid = "550e8400-e29b-41d4-a716-446655440003"
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_backup_execution.test", "database_uuid", "550e8400-e29b-41d4-a716-446655440001"),
					resource.TestCheckResourceAttr("coolify_backup_execution.test", "backup_uuid", "550e8400-e29b-41d4-a716-446655440002"),
					resource.TestCheckResourceAttr("coolify_backup_execution.test", "execution_uuid", "550e8400-e29b-41d4-a716-446655440003"),
				),
			},
		},
	})
}

func TestBackupExecutionResource_DeleteCalled(t *testing.T) {
	t.Parallel()
	var deleted atomic.Bool
	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /api/v1/databases/550e8400-e29b-41d4-a716-446655440004/backups/550e8400-e29b-41d4-a716-446655440005/executions/550e8400-e29b-41d4-a716-446655440006", func(w http.ResponseWriter, _ *http.Request) {
		deleted.Store(true)
		w.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_backup_execution", "test", `
					database_uuid  = "550e8400-e29b-41d4-a716-446655440004"
					backup_uuid    = "550e8400-e29b-41d4-a716-446655440005"
					execution_uuid = "550e8400-e29b-41d4-a716-446655440006"
				`),
			},
		},
	})
	if !deleted.Load() {
		t.Error("expected DELETE to be called on destroy")
	}
}

func TestBackupExecutionResource_Import(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /api/v1/databases/550e8400-e29b-41d4-a716-446655440007/backups/550e8400-e29b-41d4-a716-446655440008/executions/550e8400-e29b-41d4-a716-446655440009", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_backup_execution", "test", `
					database_uuid  = "550e8400-e29b-41d4-a716-446655440007"
					backup_uuid    = "550e8400-e29b-41d4-a716-446655440008"
					execution_uuid = "550e8400-e29b-41d4-a716-446655440009"
				`),
			},
			{
				ResourceName:                         "coolify_backup_execution.test",
				ImportState:                          true,
				ImportStateId:                        "550e8400-e29b-41d4-a716-446655440007:550e8400-e29b-41d4-a716-446655440008:550e8400-e29b-41d4-a716-446655440009",
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "execution_uuid",
			},
		},
	})
}

func newBackupExecDeleteMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /api/v1/databases/550e8400-e29b-41d4-a716-446655440007/backups/550e8400-e29b-41d4-a716-446655440008/executions/550e8400-e29b-41d4-a716-446655440009", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	return mux
}

func TestBackupExecutionResource_ImportBadFormat(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(acctest.WithVersionEndpoint(newBackupExecDeleteMux()))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_backup_execution", "test", `
					database_uuid  = "550e8400-e29b-41d4-a716-446655440007"
					backup_uuid    = "550e8400-e29b-41d4-a716-446655440008"
					execution_uuid = "550e8400-e29b-41d4-a716-446655440009"
				`),
			},
			{
				ResourceName:  "coolify_backup_execution.test",
				ImportState:   true,
				ImportStateId: "bad-format",
				ExpectError:   regexp.MustCompile(`Invalid import ID`),
			},
		},
	})
}

func TestBackupExecutionResource_ImportBadDatabaseUUID(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(acctest.WithVersionEndpoint(newBackupExecDeleteMux()))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_backup_execution", "test", `
					database_uuid  = "550e8400-e29b-41d4-a716-446655440007"
					backup_uuid    = "550e8400-e29b-41d4-a716-446655440008"
					execution_uuid = "550e8400-e29b-41d4-a716-446655440009"
				`),
			},
			{
				ResourceName:  "coolify_backup_execution.test",
				ImportState:   true,
				ImportStateId: "not-a-uuid:550e8400-e29b-41d4-a716-446655440008:550e8400-e29b-41d4-a716-446655440009",
				ExpectError:   regexp.MustCompile(`(?s)Invalid Import ID.*database UUID segment`),
			},
		},
	})
}

func TestBackupExecutionResource_ImportBadBackupUUID(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(acctest.WithVersionEndpoint(newBackupExecDeleteMux()))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_backup_execution", "test", `
					database_uuid  = "550e8400-e29b-41d4-a716-446655440007"
					backup_uuid    = "550e8400-e29b-41d4-a716-446655440008"
					execution_uuid = "550e8400-e29b-41d4-a716-446655440009"
				`),
			},
			{
				ResourceName:  "coolify_backup_execution.test",
				ImportState:   true,
				ImportStateId: "550e8400-e29b-41d4-a716-446655440007:not-a-uuid:550e8400-e29b-41d4-a716-446655440009",
				ExpectError:   regexp.MustCompile(`(?s)Invalid Import ID.*backup UUID segment`),
			},
		},
	})
}

func TestBackupExecutionResource_ImportBadExecutionUUID(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(acctest.WithVersionEndpoint(newBackupExecDeleteMux()))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_backup_execution", "test", `
					database_uuid  = "550e8400-e29b-41d4-a716-446655440007"
					backup_uuid    = "550e8400-e29b-41d4-a716-446655440008"
					execution_uuid = "550e8400-e29b-41d4-a716-446655440009"
				`),
			},
			{
				ResourceName:  "coolify_backup_execution.test",
				ImportState:   true,
				ImportStateId: "550e8400-e29b-41d4-a716-446655440007:550e8400-e29b-41d4-a716-446655440008:not-a-uuid",
				ExpectError:   regexp.MustCompile(`(?s)Invalid Import ID.*execution UUID segment`),
			},
		},
	})
}
