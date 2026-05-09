package backup_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/SebTardif/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

type mockBackupState struct {
	mu          sync.Mutex
	id          int
	uuid        string
	dbUUID      string
	frequency   string
	enabled     bool
	s3StorageID string
	retainDays  *int64
}

func newMockBackupServer() (*httptest.Server, *mockBackupState) {
	retain := int64(7)
	state := &mockBackupState{
		id:         42,
		uuid:       "bkp-uuid-001",
		dbUUID:     "db-uuid-001",
		frequency:  "0 2 * * *",
		enabled:    true,
		retainDays: &retain,
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		state.mu.Lock()
		defer state.mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s/backups", state.dbUUID):
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)
			if v, ok := body["frequency"].(string); ok {
				state.frequency = v
			}
			if v, ok := body["enabled"].(bool); ok {
				state.enabled = v
			}
			if v, ok := body["number_of_backups_locally"].(float64); ok {
				i := int64(v)
				state.retainDays = &i
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(backupResponse(state))

		case r.Method == http.MethodGet && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s/backups/%d", state.dbUUID, state.id):
			json.NewEncoder(w).Encode(backupResponse(state))

		case r.Method == http.MethodPatch && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s/backups/%d", state.dbUUID, state.id):
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)
			if v, ok := body["frequency"].(string); ok {
				state.frequency = v
			}
			if v, ok := body["enabled"].(bool); ok {
				state.enabled = v
			}
			if v, ok := body["number_of_backups_locally"].(float64); ok {
				i := int64(v)
				state.retainDays = &i
			}
			json.NewEncoder(w).Encode(backupResponse(state))

		case r.Method == http.MethodDelete && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s/backups/%d", state.dbUUID, state.id):
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"message": "deleted"})

		default:
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
		}
	}))
	return srv, state
}

func backupResponse(s *mockBackupState) map[string]interface{} {
	resp := map[string]interface{}{
		"id":            s.id,
		"uuid":          s.uuid,
		"database_uuid": s.dbUUID,
		"frequency":     s.frequency,
		"enabled":       s.enabled,
	}
	if s.s3StorageID != "" {
		resp["s3_storage_id"] = s.s3StorageID
	}
	if s.retainDays != nil {
		resp["number_of_backups_locally"] = *s.retainDays
	}
	return resp
}

func testBackupConfig(endpoint, attrs string) string {
	return fmt.Sprintf(`
provider "coolify" {
  endpoint = %q
  token    = "test-token"
}

resource "coolify_database_backup" "test" {
  %s
}
`, endpoint, attrs)
}

func TestDatabaseBackupResource_Create(t *testing.T) {
	srv, _ := newMockBackupServer()
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testBackupConfig(srv.URL, `
					database_uuid = "db-uuid-001"
					frequency     = "0 2 * * *"
					enabled       = true
					retain_days   = 7
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_database_backup.test", "id", "42"),
					resource.TestCheckResourceAttr("coolify_database_backup.test", "uuid", "bkp-uuid-001"),
					resource.TestCheckResourceAttr("coolify_database_backup.test", "database_uuid", "db-uuid-001"),
					resource.TestCheckResourceAttr("coolify_database_backup.test", "frequency", "0 2 * * *"),
					resource.TestCheckResourceAttr("coolify_database_backup.test", "enabled", "true"),
					resource.TestCheckResourceAttr("coolify_database_backup.test", "retain_days", "7"),
				),
			},
		},
	})
}

func TestDatabaseBackupResource_Update(t *testing.T) {
	srv, _ := newMockBackupServer()
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testBackupConfig(srv.URL, `
					database_uuid = "db-uuid-001"
					frequency     = "0 2 * * *"
					enabled       = true
					retain_days   = 7
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_database_backup.test", "frequency", "0 2 * * *"),
					resource.TestCheckResourceAttr("coolify_database_backup.test", "enabled", "true"),
				),
			},
			{
				Config: testBackupConfig(srv.URL, `
					database_uuid = "db-uuid-001"
					frequency     = "0 4 * * *"
					enabled       = false
					retain_days   = 14
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_database_backup.test", "frequency", "0 4 * * *"),
					resource.TestCheckResourceAttr("coolify_database_backup.test", "enabled", "false"),
					resource.TestCheckResourceAttr("coolify_database_backup.test", "retain_days", "14"),
				),
			},
		},
	})
}

func TestDatabaseBackupResource_Import(t *testing.T) {
	srv, _ := newMockBackupServer()
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testBackupConfig(srv.URL, `
					database_uuid = "db-uuid-001"
					frequency     = "0 2 * * *"
					enabled       = true
					retain_days   = 7
				`),
			},
			{
				ResourceName:                         "coolify_database_backup.test",
				ImportState:                          true,
				ImportStateId:                        "db-uuid-001:42",
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
			},
		},
	})
}
