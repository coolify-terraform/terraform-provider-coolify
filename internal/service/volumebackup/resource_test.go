package volumebackup_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

const (
	appUUID   = "aaaa0001-0001-4000-8000-000000000001"
	storUUID  = "bbbb0002-0002-4000-8000-000000000002"
	schedUUID = "cccc0003-0003-4000-8000-000000000003"
)

func newVolumeBackupMock(t *testing.T) *httptest.Server {
	t.Helper()
	var mu sync.Mutex
	var schedule *client.VolumeBackupSchedule
	return httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/applications/"+appUUID+"/storages/"+storUUID+"/backups":
			var input client.UpsertVolumeBackupInput
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				http.Error(w, `{"error":"bad json"}`, http.StatusBadRequest)
				return
			}
			if input.Frequency == "" {
				t.Errorf("expected frequency in PUT body")
				http.Error(w, `{"message":"validation failed"}`, http.StatusUnprocessableEntity)
				return
			}
			created := schedule == nil
			enabled := true
			if input.Enabled != nil {
				enabled = *input.Enabled
			}
			saveS3 := false
			if input.SaveS3 != nil {
				saveS3 = *input.SaveS3
			}
			timeout := int64(3600)
			if input.Timeout != nil {
				timeout = *input.Timeout
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
			schedule = &client.VolumeBackupSchedule{
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
			_ = json.NewEncoder(w).Encode(schedule)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/applications/"+appUUID+"/storages":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"persistent_storages": []client.Storage{{UUID: storUUID, Name: "data", MountPath: "/data"}},
				"file_storages":       []client.Storage{},
			})
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/applications/"+appUUID+"/storages/"+storUUID+"/backups":
			if schedule == nil {
				http.Error(w, `{"message":"Storage backup schedule not found."}`, http.StatusNotFound)
				return
			}
			schedule = nil
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "deleted"})
		default:
			http.Error(w, `{}`, http.StatusNotFound)
		}
	})))
}

func TestStorageBackupResource_CreateUpdateDelete(t *testing.T) {
	t.Parallel()
	srv := newVolumeBackupMock(t)
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
  enabled          = true
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_storage_backup.test", "uuid", schedUUID),
					resource.TestCheckResourceAttr("coolify_storage_backup.test", "frequency", "0 2 * * *"),
					resource.TestCheckResourceAttr("coolify_storage_backup.test", "storage_type", "persistent"),
					resource.TestCheckResourceAttr("coolify_storage_backup.test", "enabled", "true"),
					resource.TestCheckResourceAttr("coolify_storage_backup.test", "timeout", "3600"),
				),
			},
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
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
				Config: acctest.ProviderBlockForURL(srv.URL) + `
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
	srv := newVolumeBackupMock(t)
	defer srv.Close()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_storage_backup" "test" {
  application_uuid = "` + appUUID + `"
  storage_uuid     = "` + storUUID + `"
  frequency        = "not-a-cron"
}`,
			ExpectError: regexp.MustCompile(`must be a valid cron`),
		}},
	})
}

func TestStorageBackupResource_DisableLocalWithoutS3(t *testing.T) {
	t.Parallel()
	srv := newVolumeBackupMock(t)
	defer srv.Close()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.ProviderBlockForURL(srv.URL) + `
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

func TestStorageBackupResource_Import(t *testing.T) {
	t.Parallel()
	srv := newVolumeBackupMock(t)
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
