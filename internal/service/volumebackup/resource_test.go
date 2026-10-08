package volumebackup_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/spectest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

const (
	appUUID   = "aaaa0001-0001-4000-8000-000000000001"
	storUUID  = "bbbb0002-0002-4000-8000-000000000002"
	schedUUID = "cccc0003-0003-4000-8000-000000000003"
)

type volumeBackupMock struct {
	server            *httptest.Server
	mu                sync.Mutex
	schedule          *client.VolumeBackupSchedule
	puts              int
	createSentTimeout bool
}

func newVolumeBackupMock(t *testing.T) *volumeBackupMock {
	t.Helper()
	m := &volumeBackupMock{}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.mu.Lock()
		defer m.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/applications/"+appUUID+"/storages/"+storUUID+"/backups":
			raw, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, `{"error":"bad json"}`, http.StatusBadRequest)
				return
			}
			body, err := acctest.DecodeObject(bytes.NewReader(raw))
			if err != nil {
				http.Error(w, `{"error":"bad json"}`, http.StatusBadRequest)
				return
			}
			var absentTimeout int
			timeoutPtr, err := acctest.DecodeOptional[int64](body, "timeout", &absentTimeout)
			if err != nil {
				http.Error(w, `{"error":"bad json"}`, http.StatusBadRequest)
				return
			}
			var input client.UpsertVolumeBackupInput
			if err := json.Unmarshal(raw, &input); err != nil {
				http.Error(w, `{"error":"bad json"}`, http.StatusBadRequest)
				return
			}
			if input.Frequency == "" {
				t.Errorf("expected frequency in PUT body")
				http.Error(w, `{"message":"validation failed"}`, http.StatusUnprocessableEntity)
				return
			}
			m.puts++
			created := m.schedule == nil
			if created {
				m.createSentTimeout = timeoutPtr != nil
				if absentTimeout == 0 && timeoutPtr == nil {
					t.Errorf("omitted timeout was not counted as absent")
				}
			}
			enabled := true
			if input.Enabled != nil {
				enabled = *input.Enabled
			}
			saveS3 := false
			if input.SaveS3 != nil {
				saveS3 = *input.SaveS3
			}
			// 36000 is scheduled_volume_backups.timeout after the migrations
			// in Coolify v4.4.2. It is the response when the key is absent.
			// It is not evidence the provider sent timeout.
			timeout := int64(36000)
			if timeoutPtr != nil {
				timeout = *timeoutPtr
			}
			retLocal := int64(7)
			if input.RetentionAmountLocally != nil {
				retLocal = *input.RetentionAmountLocally
			}
			retDaysLocal, retAmtS3, retDaysS3 := int64(0), int64(7), int64(0)
			if input.RetentionDaysLocally != nil {
				retDaysLocal = *input.RetentionDaysLocally
			}
			if input.RetentionAmountS3 != nil {
				retAmtS3 = *input.RetentionAmountS3
			}
			if input.RetentionDaysS3 != nil {
				retDaysS3 = *input.RetentionDaysS3
			}
			var retMaxLocal, retMaxS3 float64
			if input.RetentionMaxStorageLocal != nil {
				retMaxLocal = *input.RetentionMaxStorageLocal
			}
			if input.RetentionMaxStorageS3 != nil {
				retMaxS3 = *input.RetentionMaxStorageS3
			}
			disableLocal, stopDuring := false, false
			if input.DisableLocalBackup != nil {
				disableLocal = *input.DisableLocalBackup
			}
			if input.StopDuringBackup != nil {
				stopDuring = *input.StopDuringBackup
			}
			m.schedule = &client.VolumeBackupSchedule{
				UUID: schedUUID, StorageUUID: storUUID, StorageType: "persistent",
				Frequency: input.Frequency, Enabled: enabled, SaveS3: saveS3,
				DisableLocalBackup: disableLocal, StopDuringBackup: stopDuring,
				S3StorageUUID:          input.S3StorageUUID,
				RetentionAmountLocally: retLocal, RetentionDaysLocally: retDaysLocal,
				RetentionMaxStorageLocal: retMaxLocal,
				RetentionAmountS3:        retAmtS3, RetentionDaysS3: retDaysS3,
				RetentionMaxStorageS3: retMaxS3, Timeout: timeout,
			}
			if created {
				w.WriteHeader(http.StatusCreated)
			} else {
				w.WriteHeader(http.StatusOK)
			}
			_ = json.NewEncoder(w).Encode(m.schedule)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/applications/"+appUUID+"/storages":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"persistent_storages": []client.Storage{{UUID: storUUID, Name: "data", MountPath: "/data"}},
				"file_storages":       []client.Storage{},
			})
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/applications/"+appUUID+"/storages/"+storUUID+"/backups":
			if m.schedule == nil {
				http.Error(w, `{"message":"Storage backup schedule not found."}`, http.StatusNotFound)
				return
			}
			m.schedule = nil
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "deleted"})
		default:
			http.Error(w, `{}`, http.StatusNotFound)
		}
	})
	m.server = httptest.NewServer(spectest.WithSpecAudit(t, "coolify-v4",
		acctest.WithVersionEndpoint(handler)))
	return m
}

func (m *volumeBackupMock) Close() { m.server.Close() }

func (m *volumeBackupMock) URL() string { return m.server.URL }

// checkVolumeBackupDestroy fails if the mock still holds a schedule after destroy.
// Coolify has no GET for schedules; mock state is the only signal.
func (m *volumeBackupMock) checkDestroy() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		m.mu.Lock()
		defer m.mu.Unlock()
		if m.schedule != nil {
			return fmt.Errorf("volume backup schedule still present after destroy (uuid=%s)", m.schedule.UUID)
		}
		return nil
	}
}

func TestStorageBackupResource_CreateUpdateDelete(t *testing.T) {
	t.Parallel()
	mock := newVolumeBackupMock(t)
	defer mock.Close()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             mock.checkDestroy(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(mock.URL()) + `
resource "coolify_storage_backup" "test" {
  application_uuid = "` + appUUID + `"
  storage_uuid     = "` + storUUID + `"
  frequency        = "0 2 * * *"
  enabled          = true
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_storage_backup.test", "uuid", schedUUID),
					resource.TestCheckResourceAttr("coolify_storage_backup.test", "frequency", "0 2 * * *"),
					resource.TestCheckResourceAttr("coolify_storage_backup.test", "storage_type", "persistent"),
					resource.TestCheckResourceAttr("coolify_storage_backup.test", "enabled", "true"),
					resource.TestCheckResourceAttr("coolify_storage_backup.test", "timeout", "36000"),
					resource.TestCheckResourceAttr("coolify_storage_backup.test", "retention_amount_locally", "7"),
				),
			},
			{
				Config: acctest.ProviderBlockForURL(mock.URL()) + `
resource "coolify_storage_backup" "test" {
  application_uuid = "` + appUUID + `"
  storage_uuid     = "` + storUUID + `"
  frequency        = "0 3 * * *"
  enabled          = false
  timeout          = 600
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_storage_backup.test", "frequency", "0 3 * * *"),
					resource.TestCheckResourceAttr("coolify_storage_backup.test", "enabled", "false"),
					resource.TestCheckResourceAttr("coolify_storage_backup.test", "timeout", "600"),
				),
			},
			{
				Config: acctest.ProviderBlockForURL(mock.URL()) + `
resource "coolify_storage_backup" "test" {
  application_uuid = "` + appUUID + `"
  storage_uuid     = "` + storUUID + `"
  frequency        = "0 3 * * *"
  enabled          = false
  timeout          = 600
}`,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestStorageBackupResource_MissingBackupNotificationDays(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		version  string
		wantSent bool
	}{
		{version: "4.4.1", wantSent: true},
		{version: "4.4.0", wantSent: false},
	} {
		t.Run(tc.version, func(t *testing.T) {
			t.Parallel()
			var sent *int64
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch {
				case r.Method == http.MethodPut && r.URL.Path == "/api/v1/applications/"+appUUID+"/storages/"+storUUID+"/backups":
					var input client.UpsertVolumeBackupInput
					if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
						http.Error(w, `{"error":"bad json"}`, http.StatusBadRequest)
						return
					}
					sent = input.MissingBackupNotificationDays
					w.WriteHeader(http.StatusCreated)
					_ = json.NewEncoder(w).Encode(client.VolumeBackupSchedule{
						UUID:                          schedUUID,
						StorageUUID:                   storUUID,
						StorageType:                   "persistent",
						Frequency:                     input.Frequency,
						Enabled:                       true,
						Timeout:                       3600,
						RetentionAmountLocally:        7,
						RetentionAmountS3:             7,
						MissingBackupNotificationDays: input.MissingBackupNotificationDays,
					})
				case r.Method == http.MethodGet && r.URL.Path == "/api/v1/applications/"+appUUID+"/storages":
					_ = json.NewEncoder(w).Encode(map[string]interface{}{
						"persistent_storages": []client.Storage{{UUID: storUUID, Name: "data", MountPath: "/data"}},
						"file_storages":       []client.Storage{},
					})
				case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/applications/"+appUUID+"/storages/"+storUUID+"/backups":
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode(map[string]string{"message": "deleted"})
				default:
					http.Error(w, `{}`, http.StatusNotFound)
				}
			})
			srv := httptest.NewServer(acctest.WithVersionEndpointVersion(handler, tc.version))
			defer srv.Close()

			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
				Steps: []resource.TestStep{{
					Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_storage_backup" "test" {
  application_uuid                  = "` + appUUID + `"
  storage_uuid                      = "` + storUUID + `"
  frequency                         = "0 2 * * *"
  missing_backup_notification_days  = 5
}`,
					Check: resource.TestCheckResourceAttr("coolify_storage_backup.test", "missing_backup_notification_days", "5"),
				}},
			})
			if tc.wantSent {
				if sent == nil || *sent != 5 {
					t.Fatalf("Coolify %s: sent missing_backup_notification_days = %v, want 5", tc.version, sent)
				}
				return
			}
			if sent != nil {
				t.Fatalf("Coolify %s: sent missing_backup_notification_days = %v, want omitted", tc.version, *sent)
			}
		})
	}
}

func TestStorageBackupResource_CreateAPIError(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("PUT /api/v1/applications/"+appUUID+"/storages/"+storUUID+"/backups", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"message":"validation failed"}`, http.StatusUnprocessableEntity)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_storage_backup" "test" {
  application_uuid = "` + appUUID + `"
  storage_uuid     = "` + storUUID + `"
  frequency        = "0 2 * * *"
}`,
			ExpectError: regexp.MustCompile(`Error creating storage backup schedule`),
		}},
	})
}

func TestStorageBackupResource_InvalidFrequency(t *testing.T) {
	t.Parallel()
	mock := newVolumeBackupMock(t)
	defer mock.Close()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.ProviderBlockForURL(mock.URL()) + `
resource "coolify_storage_backup" "test" {
  application_uuid = "` + appUUID + `"
  storage_uuid     = "` + storUUID + `"
  frequency        = "not-a-cron"
}`,
			ExpectError: regexp.MustCompile(`must be a valid cron`),
		}},
	})
}

func TestStorageBackupResource_HumanFrequencyDaily(t *testing.T) {
	t.Parallel()
	mock := newVolumeBackupMock(t)
	defer mock.Close()
	// Coolify VALID_CRON_STRINGS accepts "daily" without "@".
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             mock.checkDestroy(),
		Steps: []resource.TestStep{{
			Config: acctest.ProviderBlockForURL(mock.URL()) + `
resource "coolify_storage_backup" "test" {
  application_uuid = "` + appUUID + `"
  storage_uuid     = "` + storUUID + `"
  frequency        = "daily"
}`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("coolify_storage_backup.test", "frequency", "daily"),
				resource.TestCheckResourceAttr("coolify_storage_backup.test", "uuid", schedUUID),
			),
		}},
	})
}

func TestStorageBackupResource_DisableLocalWithoutS3(t *testing.T) {
	t.Parallel()
	mock := newVolumeBackupMock(t)
	defer mock.Close()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.ProviderBlockForURL(mock.URL()) + `
resource "coolify_storage_backup" "test" {
  application_uuid     = "` + appUUID + `"
  storage_uuid         = "` + storUUID + `"
  frequency            = "0 2 * * *"
  disable_local_backup = true
  save_s3              = false
}`,
			ExpectError: regexp.MustCompile(`disable_local_backup requires save_s3`),
		}},
	})
}

func TestStorageBackupResource_SaveS3WithoutUUID(t *testing.T) {
	t.Parallel()
	mock := newVolumeBackupMock(t)
	defer mock.Close()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.ProviderBlockForURL(mock.URL()) + `
resource "coolify_storage_backup" "test" {
  application_uuid = "` + appUUID + `"
  storage_uuid     = "` + storUUID + `"
  frequency        = "0 2 * * *"
  save_s3          = true
}`,
			ExpectError: regexp.MustCompile(`s3_storage_uuid is required when save_s3 is true`),
		}},
	})
}

func TestStorageBackupResource_Import(t *testing.T) {
	t.Parallel()
	mock := newVolumeBackupMock(t)
	defer mock.Close()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             mock.checkDestroy(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(mock.URL()) + `
resource "coolify_storage_backup" "test" {
  application_uuid = "` + appUUID + `"
  storage_uuid     = "` + storUUID + `"
  frequency        = "0 2 * * *"
}`,
			},
			{
				ResourceName:                         "coolify_storage_backup.test",
				ImportState:                          true,
				ImportStateId:                        "application:" + appUUID + ":" + storUUID,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "storage_uuid",
				// Schedule fields are not re-read (no GET); uuid may be empty after import-only.
				ImportStateVerifyIgnore: []string{
					"uuid", "frequency", "enabled", "save_s3", "disable_local_backup",
					"stop_during_backup", "s3_storage_uuid", "storage_type",
					"retention_amount_locally", "retention_days_locally", "retention_max_storage_locally",
					"retention_amount_s3", "retention_days_s3", "retention_max_storage_s3", "timeout",
				},
			},
		},
	})
}

func TestStorageBackupResource_ImportInvalidID(t *testing.T) {
	t.Parallel()
	mock := newVolumeBackupMock(t)
	defer mock.Close()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(mock.URL()) + `
resource "coolify_storage_backup" "test" {
  application_uuid = "` + appUUID + `"
  storage_uuid     = "` + storUUID + `"
  frequency        = "0 2 * * *"
}`,
			},
			{
				ResourceName:  "coolify_storage_backup.test",
				ImportState:   true,
				ImportStateId: "not-enough-parts",
				ExpectError:   regexp.MustCompile(`Invalid import ID format`),
			},
			{
				ResourceName:  "coolify_storage_backup.test",
				ImportState:   true,
				ImportStateId: "widget:" + appUUID + ":" + storUUID,
				ExpectError:   regexp.MustCompile(`must be application, service, or database[\s\S]*` + regexp.QuoteMeta(storUUID)),
			},
		},
	})
}

func TestStorageBackupResource_Delete400NotFoundIsError(t *testing.T) {
	t.Parallel()
	var deleteCalls int
	mux := http.NewServeMux()
	mux.HandleFunc("PUT /api/v1/applications/"+appUUID+"/storages/"+storUUID+"/backups", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(client.VolumeBackupSchedule{
			UUID: schedUUID, StorageUUID: storUUID, StorageType: "persistent",
			Frequency: "0 2 * * *", Enabled: true, Timeout: 3600,
			RetentionAmountLocally: 7, RetentionAmountS3: 7,
		})
	})
	mux.HandleFunc("GET /api/v1/applications/"+appUUID+"/storages", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"persistent_storages": []client.Storage{{UUID: storUUID, Name: "data", MountPath: "/data"}},
			"file_storages":       []client.Storage{},
		})
	})
	mux.HandleFunc("DELETE /api/v1/applications/"+appUUID+"/storages/"+storUUID+"/backups", func(w http.ResponseWriter, _ *http.Request) {
		deleteCalls++
		if deleteCalls == 1 {
			// 400 body contains "not found" but is not a 404. Must not be treated as success.
			http.Error(w, `{"message":"Storage backup schedule not found."}`, http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "ok"})
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()
	cfg := acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_storage_backup" "test" {
  application_uuid = "` + appUUID + `"
  storage_uuid     = "` + storUUID + `"
  frequency        = "0 2 * * *"
}`
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{Config: cfg},
			{
				Config:      cfg,
				Destroy:     true,
				ExpectError: regexp.MustCompile(`Error deleting storage backup schedule`),
			},
		},
	})
}

func TestStorageBackupResource_DeleteNotFound(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("PUT /api/v1/applications/"+appUUID+"/storages/"+storUUID+"/backups", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(client.VolumeBackupSchedule{
			UUID: schedUUID, StorageUUID: storUUID, StorageType: "persistent",
			Frequency: "0 2 * * *", Enabled: true, Timeout: 3600,
			RetentionAmountLocally: 7, RetentionAmountS3: 7,
		})
	})
	mux.HandleFunc("GET /api/v1/applications/"+appUUID+"/storages", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"persistent_storages": []client.Storage{{UUID: storUUID, Name: "data", MountPath: "/data"}},
			"file_storages":       []client.Storage{},
		})
	})
	mux.HandleFunc("DELETE /api/v1/applications/"+appUUID+"/storages/"+storUUID+"/backups", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"message":"Storage backup schedule not found."}`, http.StatusNotFound)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_storage_backup" "test" {
  application_uuid = "` + appUUID + `"
  storage_uuid     = "` + storUUID + `"
  frequency        = "0 2 * * *"
}`,
			// Destroy runs after step; DELETE 404 must be treated as success.
		}},
	})
}

func TestStorageBackupResource_DisappearsStorage(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	storageGone := false
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPut:
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(client.VolumeBackupSchedule{
				UUID: schedUUID, StorageUUID: storUUID, StorageType: "persistent",
				Frequency: "0 2 * * *", Enabled: true, Timeout: 3600,
				RetentionAmountLocally: 7, RetentionAmountS3: 7,
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/applications/"+appUUID+"/storages":
			if storageGone {
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"persistent_storages": []client.Storage{},
					"file_storages":       []client.Storage{},
				})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"persistent_storages": []client.Storage{{UUID: storUUID, Name: "data", MountPath: "/data"}},
				"file_storages":       []client.Storage{},
			})
		case r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "ok"})
		default:
			http.Error(w, `{}`, http.StatusNotFound)
		}
	})))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_storage_backup" "test" {
  application_uuid = "` + appUUID + `"
  storage_uuid     = "` + storUUID + `"
  frequency        = "0 2 * * *"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_storage_backup.test", "uuid", schedUUID),
					func(s *terraform.State) error {
						mu.Lock()
						storageGone = true
						mu.Unlock()
						return nil
					},
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestStorageBackupResource_ImportDoesNotApplyDefaults(t *testing.T) {
	t.Parallel()
	mock := newVolumeBackupMock(t)
	defer mock.Close()
	mock.schedule = &client.VolumeBackupSchedule{
		UUID: schedUUID, StorageUUID: storUUID, StorageType: "persistent",
		Frequency: "0 2 * * *", Enabled: true, RetentionAmountLocally: 30,
		RetentionAmountS3: 7, Timeout: 1800,
	}
	cfg := acctest.ProviderBlockForURL(mock.URL()) + `
resource "coolify_storage_backup" "test" {
  application_uuid = "` + appUUID + `"
  storage_uuid     = "` + storUUID + `"
  frequency        = "0 2 * * *"
}
`
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:             cfg,
				ResourceName:       "coolify_storage_backup.test",
				ImportState:        true,
				ImportStateId:      "application:" + appUUID + ":" + storUUID,
				ImportStatePersist: true,
			},
			{
				Config:      cfg,
				ExpectError: regexp.MustCompile(`Storage backup schedule is incomplete[\s\S]*` + regexp.QuoteMeta(storUUID)),
			},
		},
	})
	mock.mu.Lock()
	defer mock.mu.Unlock()
	if mock.puts != 0 {
		t.Fatalf("upsert calls = %d, want 0", mock.puts)
	}
}

func TestStorageBackupResource_UpdateKeepsOmittedRetention(t *testing.T) {
	t.Parallel()
	mock := newVolumeBackupMock(t)
	defer mock.Close()
	provider := acctest.ProviderBlockForURL(mock.URL())
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: provider + `
resource "coolify_storage_backup" "test" {
  application_uuid         = "` + appUUID + `"
  storage_uuid             = "` + storUUID + `"
  frequency                = "0 2 * * *"
  retention_amount_locally = 30
}
`,
			},
			{
				Config: provider + `
resource "coolify_storage_backup" "test" {
  application_uuid = "` + appUUID + `"
  storage_uuid     = "` + storUUID + `"
  frequency        = "0 4 * * *"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_storage_backup.test", "retention_amount_locally", "30"),
					func(*terraform.State) error {
						mock.mu.Lock()
						defer mock.mu.Unlock()
						if mock.schedule == nil || mock.schedule.RetentionAmountLocally != 30 || mock.schedule.Frequency != "0 4 * * *" {
							return fmt.Errorf("schedule = %+v, want frequency 0 4 * * * and retention 30", mock.schedule)
						}
						return nil
					},
				),
			},
		},
	})
}

func TestStorageBackupResource_CreateOmitsTimeout(t *testing.T) {
	t.Parallel()
	mock := newVolumeBackupMock(t)
	defer mock.Close()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(mock.URL()) + `
resource "coolify_storage_backup" "test" {
  application_uuid = "` + appUUID + `"
  storage_uuid     = "` + storUUID + `"
  frequency        = "0 2 * * *"
}
`,
				Check: resource.TestCheckResourceAttr("coolify_storage_backup.test", "timeout", "36000"),
			},
		},
	})
	mock.mu.Lock()
	defer mock.mu.Unlock()
	if mock.createSentTimeout {
		t.Fatal("create PUT included timeout; an omitted timeout must stay off the request")
	}
}
