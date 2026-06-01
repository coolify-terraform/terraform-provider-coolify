package backup_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/service/database/backup"
	"github.com/hashicorp/terraform-plugin-framework/path"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

type mockBackupState struct {
	mu                sync.Mutex
	id                int
	uuid              string
	dbUUID            string
	frequency         string
	enabled           bool
	saveS3            bool
	s3StorageID       string
	databasesToBackup string
	retainDays        *int64
	deleted           bool
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
			if v, ok := body["save_s3"].(bool); ok {
				state.saveS3 = v
			}
			if v, ok := body["s3_storage_uuid"].(string); ok {
				state.s3StorageID = v
			}
			if v, ok := body["database_backup_retention_amount_locally"].(float64); ok {
				i := int64(v)
				state.retainDays = &i
			}
			// Real Coolify API returns only uuid+message on create.
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"uuid":    state.uuid,
				"message": "Backup configuration created successfully.",
			})

		case r.Method == http.MethodGet && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s/backups", state.dbUUID):
			// List endpoint used by Create to resolve the backup ID.
			if state.deleted {
				json.NewEncoder(w).Encode([]map[string]interface{}{})
				return
			}
			json.NewEncoder(w).Encode([]map[string]interface{}{backupResponse(state)})

		case r.Method == http.MethodGet && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s/backups/%d", state.dbUUID, state.id):
			if state.deleted {
				http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
				return
			}
			json.NewEncoder(w).Encode(backupResponse(state))

		case r.Method == http.MethodPatch && (r.URL.Path == fmt.Sprintf("/api/v1/databases/%s/backups/%d", state.dbUUID, state.id) ||
			r.URL.Path == fmt.Sprintf("/api/v1/databases/%s/backups/%s", state.dbUUID, state.uuid)):
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)
			if v, ok := body["frequency"].(string); ok {
				state.frequency = v
			}
			if v, ok := body["enabled"].(bool); ok {
				state.enabled = v
			}
			if v, ok := body["save_s3"].(bool); ok {
				state.saveS3 = v
			}
			if s, ok := body["s3_storage_uuid"].(string); ok {
				state.s3StorageID = s
			}
			if s, ok := body["databases_to_backup"].(string); ok {
				state.databasesToBackup = s
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
			// Real Coolify API returns only message on update.
			json.NewEncoder(w).Encode(map[string]string{"message": "Database backup configuration updated"})

		case r.Method == http.MethodDelete && (r.URL.Path == fmt.Sprintf("/api/v1/databases/%s/backups/%d", state.dbUUID, state.id) ||
			r.URL.Path == fmt.Sprintf("/api/v1/databases/%s/backups/%s", state.dbUUID, state.uuid)):
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
		"save_s3":       s.saveS3,
	}
	if s.s3StorageID != "" {
		// Real API returns s3_storage_id (numeric FK), not s3_storage_uuid.
		resp["s3_storage_id"] = 1
	}
	if s.databasesToBackup != "" {
		resp["databases_to_backup"] = s.databasesToBackup
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
			backupUUID := rs.Primary.Attributes["uuid"]
			if dbUUID == "" || backupUUID == "" {
				continue
			}
			resp, err := http.Get(fmt.Sprintf("%s/api/v1/databases/%s/backups/%s", serverURL, dbUUID, backupUUID))
			if err != nil {
				return fmt.Errorf("checking backup destroy: %w", err)
			}
			resp.Body.Close()
			if resp.StatusCode != http.StatusNotFound {
				return fmt.Errorf("coolify_database_backup %s/%s still exists (status %d)", dbUUID, backupUUID, resp.StatusCode)
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
					retain_amount_locally   = 7
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_database_backup.test", "id", "42"),
					resource.TestCheckResourceAttr("coolify_database_backup.test", "uuid", "bkp-uuid-001"),
					resource.TestCheckResourceAttr("coolify_database_backup.test", "database_uuid", "eeee0001-0001-4000-8000-000000000001"),
					resource.TestCheckResourceAttr("coolify_database_backup.test", "frequency", "0 2 * * *"),
					resource.TestCheckResourceAttr("coolify_database_backup.test", "enabled", "true"),
					resource.TestCheckResourceAttr("coolify_database_backup.test", "retain_amount_locally", "7"),
				),
			},
			{
				Config: testBackupConfig(srv.URL, `
					database_uuid = "eeee0001-0001-4000-8000-000000000001"
					frequency     = "0 2 * * *"
					enabled       = true
					retain_amount_locally   = 7
				`),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestDatabaseBackupResource_CreateListFailurePreservesState(t *testing.T) {
	t.Parallel()
	const dbUUID = "eeee0001-0001-4000-8000-000000000001"
	const backupUUID = "bkp-readback-uuid-001"

	var forceListFailure atomic.Bool

	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPost && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s/backups", dbUUID):
			forceListFailure.Store(true)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"uuid":    backupUUID,
				"message": "Backup configuration created successfully.",
			})

		case r.Method == http.MethodGet && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s/backups", dbUUID):
			if forceListFailure.Load() {
				http.Error(w, `{"error":"boom"}`, http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode([]map[string]interface{}{{
				"id": 42, "uuid": backupUUID, "database_uuid": dbUUID, "frequency": "0 2 * * *", "enabled": true,
			}})

		case r.Method == http.MethodDelete && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s/backups/%s", dbUUID, backupUUID):
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"message": "deleted"})

		default:
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
		}
	})))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: testBackupConfig(srv.URL, `
				database_uuid = "eeee0001-0001-4000-8000-000000000001"
				frequency     = "0 2 * * *"
				enabled       = true
			`),
			ExpectError: regexp.MustCompile(`(?s)Database backup created but refresh failed.*Could not list database backups.*partial Terraform state was saved`),
		}},
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
					retain_amount_locally   = 7
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_database_backup.test", "frequency", "0 2 * * *"),
					resource.TestCheckResourceAttr("coolify_database_backup.test", "enabled", "true"),
					resource.TestCheckResourceAttr("coolify_database_backup.test", "retain_amount_locally", "7"),
				),
			},
			{
				Config: testBackupConfig(srv.URL, `
					database_uuid = "eeee0001-0001-4000-8000-000000000001"
					frequency     = "0 4 * * *"
					enabled       = false
					retain_amount_locally   = 14
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_database_backup.test", "frequency", "0 4 * * *"),
					resource.TestCheckResourceAttr("coolify_database_backup.test", "enabled", "false"),
					resource.TestCheckResourceAttr("coolify_database_backup.test", "retain_amount_locally", "14"),
				),
			},
			// Remove retain_amount_locally from config: Computed field accepts
			// API value, plan should be idempotent
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

func TestDatabaseBackupResource_UpdateDatabasesToBackup(t *testing.T) {
	t.Parallel()
	srv, state := newMockBackupServer()
	state.frequency = "0 2 * * *"
	state.enabled = true
	state.databasesToBackup = "app,queue"
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testBackupConfig(srv.URL, `
					database_uuid        = "eeee0001-0001-4000-8000-000000000001"
					frequency            = "0 2 * * *"
					enabled              = true
					databases_to_backup  = "app,queue"
				`),
				Check: resource.TestCheckResourceAttr("coolify_database_backup.test", "databases_to_backup", "app,queue"),
			},
			// Update databases_to_backup to a different value.
			{
				Config: testBackupConfig(srv.URL, `
					database_uuid        = "eeee0001-0001-4000-8000-000000000001"
					frequency            = "0 2 * * *"
					enabled              = true
					databases_to_backup  = "orders"
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_database_backup.test", "databases_to_backup", "orders"),
					func(_ *terraform.State) error {
						state.mu.Lock()
						defer state.mu.Unlock()

						if state.databasesToBackup != "orders" {
							return fmt.Errorf("expected remote databases_to_backup to be \"orders\", got %q", state.databasesToBackup)
						}

						return nil
					},
				),
			},
			// Verify idempotent plan.
			{
				Config: testBackupConfig(srv.URL, `
					database_uuid        = "eeee0001-0001-4000-8000-000000000001"
					frequency            = "0 2 * * *"
					enabled              = true
					databases_to_backup  = "orders"
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
					retain_amount_locally   = 7
				`),
			},
			{
				ResourceName:                         "coolify_database_backup.test",
				ImportState:                          true,
				ImportStateId:                        "eeee0001-0001-4000-8000-000000000001:42",
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateVerifyIgnore: []string{
					"retain_amount_locally", "retain_days_locally", "retain_max_storage_locally",
					"retain_amount_s3", "retain_days_s3", "retain_max_storage_s3",
					"timeout", "databases_to_backup",
				},
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
				"uuid":    "bkp-disappear-uuid",
				"message": "Backup configuration created successfully.",
			})
		case r.Method == http.MethodGet && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s/backups", dbUUID):
			if deleted {
				json.NewEncoder(w).Encode([]map[string]interface{}{})
				return
			}
			json.NewEncoder(w).Encode([]map[string]interface{}{{
				"id": backupID, "uuid": "bkp-disappear-uuid",
				"database_uuid": dbUUID, "frequency": "0 2 * * *",
				"enabled": true, "database_backup_retention_amount_locally": 7,
			}})
		case r.Method == http.MethodGet && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s/backups/%d", dbUUID, backupID),
			r.Method == http.MethodGet && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s/backups/bkp-disappear-uuid", dbUUID):
			if deleted {
				http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
				return
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id": backupID, "uuid": "bkp-disappear-uuid",
				"database_uuid": dbUUID, "frequency": "0 2 * * *",
				"enabled": true, "database_backup_retention_amount_locally": 7,
			})
		case r.Method == http.MethodDelete && (r.URL.Path == fmt.Sprintf("/api/v1/databases/%s/backups/%d", dbUUID, backupID) ||
			r.URL.Path == fmt.Sprintf("/api/v1/databases/%s/backups/bkp-disappear-uuid", dbUUID)):
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
					retain_amount_locally   = 7
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_database_backup.test", "uuid"),
					func(s *terraform.State) error {
						req, _ := http.NewRequest(http.MethodDelete,
							fmt.Sprintf("%s/api/v1/databases/%s/backups/bkp-disappear-uuid", srv.URL, dbUUID), nil)
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
					retain_amount_locally   = 7
				`),
			},
			{
				ResourceName:  "coolify_database_backup.test",
				ImportState:   true,
				ImportStateId: "missing-colon",
				ExpectError:   regexp.MustCompile(`Invalid Import ID`),
			},
		},
	})
}

func TestDatabaseBackupResource_ImportBadDatabaseUUID(t *testing.T) {
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
					retain_amount_locally   = 7
				`),
			},
			{
				ResourceName:  "coolify_database_backup.test",
				ImportState:   true,
				ImportStateId: "not-a-uuid:1",
				ExpectError:   regexp.MustCompile(`(?s)Invalid Import ID.*database UUID segment`),
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
					retain_amount_locally   = 7
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

func TestDatabaseBackupResource_ImportZeroID(t *testing.T) {
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
					retain_amount_locally   = 7
				`),
			},
			{
				ResourceName:  "coolify_database_backup.test",
				ImportState:   true,
				ImportStateId: "eeee0001-0001-4000-8000-000000000001:0",
				ExpectError:   regexp.MustCompile(`backup_id must be a positive integer`),
			},
		},
	})
}

func TestDatabaseBackupResource_InvalidRetainAmountLocally(t *testing.T) {
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
					retain_amount_locally   = -1
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
					retain_amount_locally   = 7
				`),
				Check: resource.TestCheckResourceAttr("coolify_database_backup.test", "frequency", "@daily"),
			},
		},
	})
}

// TestDatabaseBackupResource_S3RoundTrip verifies that creating a backup with
// save_s3=true and s3_storage_uuid preserves the UUID across plan/apply cycles.
// The real Coolify API returns s3_storage_id (numeric FK) not s3_storage_uuid,
// so the provider must preserve the user-configured UUID from state.
func TestDatabaseBackupResource_S3RoundTrip(t *testing.T) {
	t.Parallel()
	srv, _ := newMockBackupServer()
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             checkBackupDestroy(srv.URL),
		Steps: []resource.TestStep{
			{
				Config: testBackupConfig(srv.URL, `
					database_uuid    = "eeee0001-0001-4000-8000-000000000001"
					frequency        = "0 2 * * *"
					enabled          = true
					save_s3          = true
					s3_storage_uuid  = "aaaa1111-2222-3333-4444-555566667777"
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_database_backup.test", "save_s3", "true"),
					resource.TestCheckResourceAttr("coolify_database_backup.test", "s3_storage_uuid", "aaaa1111-2222-3333-4444-555566667777"),
				),
			},
			// Idempotency: no plan diff on re-apply
			{
				Config: testBackupConfig(srv.URL, `
					database_uuid    = "eeee0001-0001-4000-8000-000000000001"
					frequency        = "0 2 * * *"
					enabled          = true
					save_s3          = true
					s3_storage_uuid  = "aaaa1111-2222-3333-4444-555566667777"
				`),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
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

func TestDatabaseBackupResource_SaveS3WithoutUUID(t *testing.T) {
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
					save_s3       = true
				`),
				ExpectError: regexp.MustCompile(`Missing S3 Storage UUID`),
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
			// Real Coolify API returns only uuid+message on create.
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"uuid":    backupUUID,
				"message": "Backup configuration created successfully.",
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

		case r.Method == http.MethodDelete && (r.URL.Path == fmt.Sprintf("/api/v1/databases/%s/backups/%d", dbUUID, realID) ||
			strings.HasPrefix(r.URL.Path, fmt.Sprintf("/api/v1/databases/%s/backups/", dbUUID))):
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
					retain_amount_locally   = 7
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
					retain_amount_locally   = 7
				`, dbUUID)),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

// TestBackupResource_PartialStateOnListFailure verifies that when Create
// succeeds but the follow-up list call fails, the error message includes
// the backup UUID (proving partial state was saved before the error).
func TestBackupResource_PartialStateOnListFailure(t *testing.T) {
	t.Parallel()
	dbUUID := "eeee0001-0001-4000-8000-000000000001"
	backupUUID := "partial-state-bkp-001"
	var listCallCount atomic.Int32

	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s/backups", dbUUID):
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"uuid":    backupUUID,
				"message": "Backup configuration created successfully.",
			})

		case r.Method == http.MethodGet && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s/backups", dbUUID):
			count := listCallCount.Add(1)
			if count == 1 {
				// First list call (during Create read-back) fails.
				http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
				return
			}
			// Subsequent calls succeed (during destroy cleanup).
			json.NewEncoder(w).Encode([]map[string]interface{}{})

		case r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"message": "deleted"})

		default:
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
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
				`, dbUUID)),
				// The error message must include the backup UUID, proving
				// partial state was set before the list failure.
				ExpectError: regexp.MustCompile(backupUUID),
			},
		},
	})
}

func TestDatabaseBackupResource_UpgradeStateV0(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	res := backup.NewResource()

	// Get current (v1) schema.
	var schemaResp fwresource.SchemaResponse
	res.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)

	// Get v0 upgrader.
	upgraders := res.(fwresource.ResourceWithUpgradeState).UpgradeState(ctx)
	v0Up, ok := upgraders[0]
	if !ok {
		t.Fatal("v0 state upgrader not found")
	}

	// Build v0 raw state with s3_storage_id set.
	v0Raw := tftypes.NewValue(
		v0Up.PriorSchema.Type().TerraformType(ctx),
		map[string]tftypes.Value{
			"id":                         tftypes.NewValue(tftypes.Number, 42),
			"uuid":                       tftypes.NewValue(tftypes.String, "bkp-uuid-001"),
			"database_uuid":              tftypes.NewValue(tftypes.String, "db-uuid-001"),
			"frequency":                  tftypes.NewValue(tftypes.String, "0 2 * * *"),
			"enabled":                    tftypes.NewValue(tftypes.Bool, true),
			"save_s3":                    tftypes.NewValue(tftypes.Bool, true),
			"s3_storage_id":              tftypes.NewValue(tftypes.String, "storage-uuid-001"),
			"databases_to_backup":        tftypes.NewValue(tftypes.String, nil),
			"dump_all":                   tftypes.NewValue(tftypes.Bool, false),
			"backup_now":                 tftypes.NewValue(tftypes.Bool, nil),
			"retain_amount_locally":      tftypes.NewValue(tftypes.Number, nil),
			"retain_days_locally":        tftypes.NewValue(tftypes.Number, nil),
			"retain_max_storage_locally": tftypes.NewValue(tftypes.Number, nil),
			"retain_amount_s3":           tftypes.NewValue(tftypes.Number, nil),
			"retain_days_s3":             tftypes.NewValue(tftypes.Number, nil),
			"retain_max_storage_s3":      tftypes.NewValue(tftypes.Number, nil),
			"timeout":                    tftypes.NewValue(tftypes.Number, nil),
		},
	)
	priorState := tfsdk.State{
		Schema: *v0Up.PriorSchema,
		Raw:    v0Raw,
	}

	// Prepare empty v1 state for the upgrader to populate.
	newState := tfsdk.State{
		Schema: schemaResp.Schema,
		Raw:    tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil),
	}

	req := fwresource.UpgradeStateRequest{State: &priorState}
	resp := fwresource.UpgradeStateResponse{State: newState}
	v0Up.StateUpgrader(ctx, req, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics.Errors())
	}

	// Verify s3_storage_id was renamed to s3_storage_uuid.
	var s3UUID types.String
	resp.State.GetAttribute(ctx, path.Root("s3_storage_uuid"), &s3UUID)
	if s3UUID.ValueString() != "storage-uuid-001" {
		t.Errorf("s3_storage_uuid: got %q, want %q", s3UUID.ValueString(), "storage-uuid-001")
	}

	// Verify other key fields were preserved.
	var id types.Int64
	resp.State.GetAttribute(ctx, path.Root("id"), &id)
	if id.ValueInt64() != 42 {
		t.Errorf("id: got %d, want 42", id.ValueInt64())
	}

	var dbUUID types.String
	resp.State.GetAttribute(ctx, path.Root("database_uuid"), &dbUUID)
	if dbUUID.ValueString() != "db-uuid-001" {
		t.Errorf("database_uuid: got %q, want %q", dbUUID.ValueString(), "db-uuid-001")
	}

	var freq types.String
	resp.State.GetAttribute(ctx, path.Root("frequency"), &freq)
	if freq.ValueString() != "0 2 * * *" {
		t.Errorf("frequency: got %q, want %q", freq.ValueString(), "0 2 * * *")
	}

	var saveS3 types.Bool
	resp.State.GetAttribute(ctx, path.Root("save_s3"), &saveS3)
	if !saveS3.ValueBool() {
		t.Error("save_s3: got false, want true")
	}
}
