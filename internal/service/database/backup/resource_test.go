package backup_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"testing"

	"github.com/SebTardif/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
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
	deleted     bool
}

func newMockBackupServer() (*httptest.Server, *mockBackupState) {
	state := &mockBackupState{
		id:     42,
		uuid:   "bkp-uuid-001",
		dbUUID: "eeee0001-0001-4000-8000-000000000001",
		// Deliberately differ from test configs so the POST handler must
		// receive and apply the correct values. Pre-initializing with
		// matching values masked body serialization bugs.
	}

	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
			if v, ok := body["database_backup_retention_amount_locally"].(float64); ok {
				i := int64(v)
				state.retainDays = &i
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(backupResponse(state))

		case r.Method == http.MethodGet && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s/backups/%d", state.dbUUID, state.id):
			if state.deleted {
				http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
				return
			}
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
			if v, ok := body["s3_storage_uuid"]; ok {
				if s, ok := v.(string); ok {
					state.s3StorageID = s
				}
			}
			if v, ok := body["database_backup_retention_amount_locally"]; ok {
				if f, ok := v.(float64); ok {
					i := int64(f)
					if i == 0 {
						state.retainDays = nil
					} else {
						state.retainDays = &i
					}
				}
			}
			json.NewEncoder(w).Encode(backupResponse(state))

		case r.Method == http.MethodDelete && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s/backups/%d", state.dbUUID, state.id):
			state.deleted = true
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"message": "deleted"})

		default:
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
		}
	})))
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
		resp["s3_storage_uuid"] = s.s3StorageID
	}
	if s.retainDays != nil {
		resp["database_backup_retention_amount_locally"] = *s.retainDays
	}
	return resp
}

func testBackupConfig(endpoint, attrs string) string {
	return acctest.TestResourceConfig(endpoint, "coolify_database_backup", "test", attrs)
}

func checkBackupDestroy(serverURL string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "coolify_database_backup" {
				continue
			}
			dbUUID := rs.Primary.Attributes["database_uuid"]
			backupID := rs.Primary.Attributes["id"]
			if dbUUID == "" || backupID == "" {
				continue
			}
			resp, err := http.Get(fmt.Sprintf("%s/api/v1/databases/%s/backups/%s", serverURL, dbUUID, backupID))
			if err != nil {
				return fmt.Errorf("checking backup destroy: %w", err)
			}
			resp.Body.Close()
			if resp.StatusCode != http.StatusNotFound {
				return fmt.Errorf("coolify_database_backup %s/%s still exists (status %d)", dbUUID, backupID, resp.StatusCode)
			}
		}
		return nil
	}
}

func TestDatabaseBackupResource_Create(t *testing.T) {
	t.Parallel()
	srv, _ := newMockBackupServer()
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             checkBackupDestroy(srv.URL),
		Steps: []resource.TestStep{
			{
				Config: testBackupConfig(srv.URL, `
					database_uuid = "eeee0001-0001-4000-8000-000000000001"
					frequency     = "0 2 * * *"
					enabled       = true
					retain_days   = 7
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_database_backup.test", "id", "42"),
					resource.TestCheckResourceAttr("coolify_database_backup.test", "uuid", "bkp-uuid-001"),
					resource.TestCheckResourceAttr("coolify_database_backup.test", "database_uuid", "eeee0001-0001-4000-8000-000000000001"),
					resource.TestCheckResourceAttr("coolify_database_backup.test", "frequency", "0 2 * * *"),
					resource.TestCheckResourceAttr("coolify_database_backup.test", "enabled", "true"),
					resource.TestCheckResourceAttr("coolify_database_backup.test", "retain_days", "7"),
				),
			},
			{
				Config: testBackupConfig(srv.URL, `
					database_uuid = "eeee0001-0001-4000-8000-000000000001"
					frequency     = "0 2 * * *"
					enabled       = true
					retain_days   = 7
				`),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestDatabaseBackupResource_Update(t *testing.T) {
	t.Parallel()
	srv, _ := newMockBackupServer()
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testBackupConfig(srv.URL, `
					database_uuid = "eeee0001-0001-4000-8000-000000000001"
					frequency     = "0 2 * * *"
					enabled       = true
					retain_days   = 7
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_database_backup.test", "frequency", "0 2 * * *"),
					resource.TestCheckResourceAttr("coolify_database_backup.test", "enabled", "true"),
					resource.TestCheckResourceAttr("coolify_database_backup.test", "retain_days", "7"),
				),
			},
			{
				Config: testBackupConfig(srv.URL, `
					database_uuid = "eeee0001-0001-4000-8000-000000000001"
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
			// Remove retain_days from config: should clear to 0, not perpetual diff
			{
				Config: testBackupConfig(srv.URL, `
					database_uuid = "eeee0001-0001-4000-8000-000000000001"
					frequency     = "0 4 * * *"
					enabled       = false
				`),
				Check: resource.TestCheckNoResourceAttr("coolify_database_backup.test", "retain_days"),
			},
			// Plan idempotency after removal
			{
				Config: testBackupConfig(srv.URL, `
					database_uuid = "eeee0001-0001-4000-8000-000000000001"
					frequency     = "0 4 * * *"
					enabled       = false
				`),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestDatabaseBackupResource_Import(t *testing.T) {
	t.Parallel()
	srv, _ := newMockBackupServer()
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testBackupConfig(srv.URL, `
					database_uuid = "eeee0001-0001-4000-8000-000000000001"
					frequency     = "0 2 * * *"
					enabled       = true
					retain_days   = 7
				`),
			},
			{
				ResourceName:                         "coolify_database_backup.test",
				ImportState:                          true,
				ImportStateId:                        "eeee0001-0001-4000-8000-000000000001:42",
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
			},
		},
	})
}

func TestDatabaseBackupResource_Disappears(t *testing.T) {
	t.Parallel()
	mu := sync.Mutex{}
	deleted := false
	dbUUID := "eeee0002-0002-4000-8000-000000000002"
	backupID := 99

	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s/backups", dbUUID):
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id": backupID, "uuid": "bkp-disappear-uuid",
				"database_uuid": dbUUID, "frequency": "0 2 * * *",
				"enabled": true, "database_backup_retention_amount_locally": 7,
			})
		case r.Method == http.MethodGet && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s/backups/%d", dbUUID, backupID):
			if deleted {
				http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
				return
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id": backupID, "uuid": "bkp-disappear-uuid",
				"database_uuid": dbUUID, "frequency": "0 2 * * *",
				"enabled": true, "database_backup_retention_amount_locally": 7,
			})
		case r.Method == http.MethodDelete && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s/backups/%d", dbUUID, backupID):
			deleted = true
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testBackupConfig(srv.URL, `
					database_uuid = "eeee0002-0002-4000-8000-000000000002"
					frequency     = "0 2 * * *"
					enabled       = true
					retain_days   = 7
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_database_backup.test", "uuid"),
					func(s *terraform.State) error {
						req, _ := http.NewRequest(http.MethodDelete,
							fmt.Sprintf("%s/api/v1/databases/%s/backups/%d", srv.URL, dbUUID, backupID), nil)
						resp, err := http.DefaultClient.Do(req)
						if err != nil {
							return err
						}
						resp.Body.Close()
						return nil
					},
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestDatabaseBackupResource_ImportBadFormat(t *testing.T) {
	t.Parallel()
	srv, _ := newMockBackupServer()
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testBackupConfig(srv.URL, `
					database_uuid = "eeee0001-0001-4000-8000-000000000001"
					frequency     = "0 2 * * *"
					enabled       = true
					retain_days   = 7
				`),
			},
			{
				ResourceName:  "coolify_database_backup.test",
				ImportState:   true,
				ImportStateId: "missing-colon",
				ExpectError:   regexp.MustCompile(`Invalid import ID format`),
			},
		},
	})
}

func TestDatabaseBackupResource_ImportBadID(t *testing.T) {
	t.Parallel()
	srv, _ := newMockBackupServer()
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testBackupConfig(srv.URL, `
					database_uuid = "eeee0001-0001-4000-8000-000000000001"
					frequency     = "0 2 * * *"
					enabled       = true
					retain_days   = 7
				`),
			},
			{
				ResourceName:  "coolify_database_backup.test",
				ImportState:   true,
				ImportStateId: "eeee0001-0001-4000-8000-000000000001:not-a-number",
				ExpectError:   regexp.MustCompile(`backup_id must be an integer`),
			},
		},
	})
}

func TestDatabaseBackupResource_InvalidRetainDays(t *testing.T) {
	t.Parallel()
	srv, _ := newMockBackupServer()
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testBackupConfig(srv.URL, `
					database_uuid = "eeee0001-0001-4000-8000-000000000001"
					frequency     = "0 2 * * *"
					enabled       = true
					retain_days   = -1
				`),
				ExpectError: regexp.MustCompile(`must be at least 0`),
			},
		},
	})
}

func TestDatabaseBackupResource_CronAlias(t *testing.T) {
	t.Parallel()
	srv, _ := newMockBackupServer()
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testBackupConfig(srv.URL, `
					database_uuid = "eeee0001-0001-4000-8000-000000000001"
					frequency     = "@daily"
					enabled       = true
					retain_days   = 7
				`),
				Check: resource.TestCheckResourceAttr("coolify_database_backup.test", "frequency", "@daily"),
			},
		},
	})
}

func TestDatabaseBackupResource_InvalidCron(t *testing.T) {
	t.Parallel()
	srv, _ := newMockBackupServer()
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testBackupConfig(srv.URL, `
					database_uuid = "eeee0001-0001-4000-8000-000000000001"
					frequency     = "not a cron"
					enabled       = true
				`),
				ExpectError: regexp.MustCompile(`must be a valid cron expression`),
			},
		},
	})
}

// TestDatabaseBackupResource_CreateWithZeroID verifies that when Coolify
// returns id=0 for a newly created backup, the provider resolves the real
// ID by listing backups and matching by UUID.
func TestDatabaseBackupResource_CreateWithZeroID(t *testing.T) {
	t.Parallel()
	dbUUID := "eeee0003-0003-4000-8000-000000000003"
	backupUUID := "bkp-zero-id-uuid"
	realID := 99

	mu := sync.Mutex{}
	deleted := false

	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s/backups", dbUUID):
			// Return id=0 to simulate the Coolify bug.
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id": 0, "uuid": backupUUID,
				"database_uuid": dbUUID, "frequency": "0 2 * * *",
				"enabled": true, "database_backup_retention_amount_locally": 7,
			})

		case r.Method == http.MethodGet && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s/backups", dbUUID):
			// List endpoint returns the backup with its real ID.
			json.NewEncoder(w).Encode([]map[string]interface{}{
				{
					"id": realID, "uuid": backupUUID,
					"database_uuid": dbUUID, "frequency": "0 2 * * *",
					"enabled": true, "database_backup_retention_amount_locally": 7,
				},
			})

		case r.Method == http.MethodGet && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s/backups/%d", dbUUID, realID):
			if deleted {
				http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
				return
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id": realID, "uuid": backupUUID,
				"database_uuid": dbUUID, "frequency": "0 2 * * *",
				"enabled": true, "database_backup_retention_amount_locally": 7,
			})

		case r.Method == http.MethodDelete && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s/backups/%d", dbUUID, realID):
			deleted = true
			w.WriteHeader(http.StatusOK)

		default:
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
		}
	})))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testBackupConfig(srv.URL, fmt.Sprintf(`
					database_uuid = "%s"
					frequency     = "0 2 * * *"
					enabled       = true
					retain_days   = 7
				`, dbUUID)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_database_backup.test", "id", fmt.Sprintf("%d", realID)),
					resource.TestCheckResourceAttr("coolify_database_backup.test", "uuid", backupUUID),
				),
			},
			{
				Config: testBackupConfig(srv.URL, fmt.Sprintf(`
					database_uuid = "%s"
					frequency     = "0 2 * * *"
					enabled       = true
					retain_days   = 7
				`, dbUUID)),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}
