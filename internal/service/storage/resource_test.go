package storage_test

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sync"
	"sync/atomic"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/spectest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ---------------------------------------------------------------------------
// TestStorageResource_Create
// ---------------------------------------------------------------------------

func TestStorageResource_Create(t *testing.T) {
	t.Parallel()
	stor := client.Storage{
		UUID:      "stor-create-uuid",
		Name:      "app-data",
		MountPath: "/data",
		HostPath:  "/host/data",
	}

	mu := sync.Mutex{}
	deleted := false

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/{appUUID}/storages", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("appUUID") != "cccc0001-0001-4000-8000-000000000001" {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, `{"error":"bad request body"}`, http.StatusBadRequest)
			return
		}
		if body["name"] != "app-data" || body["mount_path"] != "/data" || body["host_path"] != "/host/data" {
			t.Errorf("POST body mismatch: got %v", body)
			http.Error(w, `{"error":"unexpected fields"}`, http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": stor.UUID})
	})
	mux.HandleFunc("GET /api/v1/applications/{appUUID}/storages", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("appUUID") != "cccc0001-0001-4000-8000-000000000001" {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if deleted {
			json.NewEncoder(w).Encode(map[string][]client.Storage{"persistent_storages": {}, "file_storages": {}})
		} else {
			json.NewEncoder(w).Encode(map[string][]client.Storage{"persistent_storages": {stor}, "file_storages": {}})
		}
	})
	mux.HandleFunc("DELETE /api/v1/applications/{appUUID}/storages/{storUUID}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("appUUID") != "cccc0001-0001-4000-8000-000000000001" || r.PathValue("storUUID") != stor.UUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		mu.Lock()
		deleted = true
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(spectest.WithSpecAudit(t, "coolify-v4",
		acctest.WithVersionEndpoint(mux)))
	defer srv.Close()

	config := testStorageResourceConfig(srv.URL, `
					application_uuid = "cccc0001-0001-4000-8000-000000000001"
					name             = "app-data"
					mount_path       = "/data"
					host_path        = "/host/data"
				`)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             checkStorageDestroy(srv.URL, "/api/v1/applications/cccc0001-0001-4000-8000-000000000001/storages"),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_storage.test", "uuid", "stor-create-uuid"),
					resource.TestCheckResourceAttr("coolify_storage.test", "name", "app-data"),
					resource.TestCheckResourceAttr("coolify_storage.test", "mount_path", "/data"),
					resource.TestCheckResourceAttr("coolify_storage.test", "host_path", "/host/data"),
				),
			},
			// Plan idempotency
			{
				Config:             config,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestStorageResource_Update
// ---------------------------------------------------------------------------

func TestStorageResource_Update(t *testing.T) {
	t.Parallel()
	mu := sync.Mutex{}
	currentStor := client.Storage{
		UUID:      "stor-update-uuid",
		Name:      "app-data",
		MountPath: "/data",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/{appUUID}/storages", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("appUUID") != "cccc0001-0001-4000-8000-000000000001" {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": currentStor.UUID})
	})
	mux.HandleFunc("GET /api/v1/applications/{appUUID}/storages", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("appUUID") != "cccc0001-0001-4000-8000-000000000001" {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string][]client.Storage{"persistent_storages": {currentStor}, "file_storages": {}})
	})
	mux.HandleFunc("PATCH /api/v1/applications/{appUUID}/storages", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("appUUID") != "cccc0001-0001-4000-8000-000000000001" {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, `{"error":"invalid json body"}`, http.StatusBadRequest)
			return
		}
		if v, ok := body["name"].(string); ok {
			currentStor.Name = v
		}
		if v, ok := body["mount_path"].(string); ok {
			currentStor.MountPath = v
		}
		if v, ok := body["host_path"]; ok {
			if s, ok := v.(string); ok {
				currentStor.HostPath = s
			}
		}
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("DELETE /api/v1/applications/{appUUID}/storages/{storUUID}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("appUUID") != "cccc0001-0001-4000-8000-000000000001" || r.PathValue("storUUID") != currentStor.UUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testStorageResourceConfig(srv.URL, `
					application_uuid = "cccc0001-0001-4000-8000-000000000001"
					name             = "app-data"
					mount_path       = "/data"
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_storage.test", "name", "app-data"),
					resource.TestCheckResourceAttr("coolify_storage.test", "mount_path", "/data"),
				),
			},
			{
				Config: testStorageResourceConfig(srv.URL, `
					application_uuid = "cccc0001-0001-4000-8000-000000000001"
					name             = "app-data-updated"
					mount_path       = "/data/new"
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_storage.test", "name", "app-data-updated"),
					resource.TestCheckResourceAttr("coolify_storage.test", "mount_path", "/data/new"),
				),
			},
			// Add host_path
			{
				Config: testStorageResourceConfig(srv.URL, `
					application_uuid = "cccc0001-0001-4000-8000-000000000001"
					name             = "app-data-updated"
					mount_path       = "/data/new"
					host_path        = "/host/data"
				`),
				Check: resource.TestCheckResourceAttr("coolify_storage.test", "host_path", "/host/data"),
			},
			// Remove host_path: should clear, not perpetual diff
			{
				Config: testStorageResourceConfig(srv.URL, `
					application_uuid = "cccc0001-0001-4000-8000-000000000001"
					name             = "app-data-updated"
					mount_path       = "/data/new"
				`),
				Check: resource.TestCheckNoResourceAttr("coolify_storage.test", "host_path"),
			},
			// Plan idempotency after removal
			{
				Config: testStorageResourceConfig(srv.URL, `
					application_uuid = "cccc0001-0001-4000-8000-000000000001"
					name             = "app-data-updated"
					mount_path       = "/data/new"
				`),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestStorageResource_UpdateReadBackCatchesNormalization
// ---------------------------------------------------------------------------

func TestStorageResource_UpdateReadBackCatchesNormalization(t *testing.T) {
	t.Parallel()
	mu := sync.Mutex{}
	currentStor := client.Storage{
		UUID:      "stor-norm-uuid",
		Name:      "my-vol",
		MountPath: "/data",
	}
	normalized := false

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/{appUUID}/storages", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": currentStor.UUID})
	})
	mux.HandleFunc("GET /api/v1/applications/{appUUID}/storages", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string][]client.Storage{"persistent_storages": {currentStor}, "file_storages": {}})
	})
	mux.HandleFunc("PATCH /api/v1/applications/{appUUID}/storages", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, `{"error":"invalid json body"}`, http.StatusBadRequest)
			return
		}
		if v, ok := body["mount_path"].(string); ok {
			currentStor.MountPath = v
		}
		// Simulate server-side normalization: API returns a different
		// mount_path than what the user sent (e.g. trailing slash added).
		if !normalized {
			currentStor.MountPath += "/"
			normalized = true
		}
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("DELETE /api/v1/applications/{appUUID}/storages/{storUUID}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testStorageResourceConfig(srv.URL, `
					application_uuid = "cccc0001-0001-4000-8000-000000000001"
					name             = "my-vol"
					mount_path       = "/data"
				`),
			},
			{
				Config: testStorageResourceConfig(srv.URL, `
					application_uuid = "cccc0001-0001-4000-8000-000000000001"
					name             = "my-vol"
					mount_path       = "/data/updated"
				`),
				// After PATCH, the server normalizes mount_path (appends "/").
				// The read-back catches this and Terraform surfaces the
				// inconsistency instead of silently diverging from reality.
				ExpectError: regexp.MustCompile(`inconsistent result after apply`),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestStorageResource_Import
// ---------------------------------------------------------------------------

func TestStorageResource_Import(t *testing.T) {
	t.Parallel()
	stor := client.Storage{
		UUID:      "dddd0001-0001-4000-8000-000000000001",
		Name:      "import-vol",
		MountPath: "/mnt/import",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/{appUUID}/storages", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("appUUID") != "cccc0001-0001-4000-8000-000000000001" {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": stor.UUID})
	})
	mux.HandleFunc("GET /api/v1/applications/{appUUID}/storages", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("appUUID") != "cccc0001-0001-4000-8000-000000000001" {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string][]client.Storage{"persistent_storages": {stor}, "file_storages": {}})
	})
	mux.HandleFunc("DELETE /api/v1/applications/{appUUID}/storages/{storUUID}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("appUUID") != "cccc0001-0001-4000-8000-000000000001" || r.PathValue("storUUID") != stor.UUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			// Step 1: create so the resource exists in state.
			{
				Config: testStorageResourceConfig(srv.URL, `
					application_uuid = "cccc0001-0001-4000-8000-000000000001"
					name             = "import-vol"
					mount_path       = "/mnt/import"
				`),
			},
			// Step 2: import and verify.
			{
				ResourceName:                         "coolify_storage.test",
				ImportState:                          true,
				ImportStateId:                        "application:cccc0001-0001-4000-8000-000000000001:dddd0001-0001-4000-8000-000000000001",
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestStorageResource_ImportBadFormat
// ---------------------------------------------------------------------------

func TestStorageResource_ImportBadFormat(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/{appUUID}/storages", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": "stor-err-uuid"})
	})
	mux.HandleFunc("GET /api/v1/applications/{appUUID}/storages", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string][]client.Storage{"persistent_storages": {{UUID: "stor-err-uuid", Name: "n", MountPath: "/m"}}, "file_storages": {}})
	})
	mux.HandleFunc("DELETE /api/v1/applications/{appUUID}/storages/{storUUID}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testStorageResourceConfig(srv.URL, `
					application_uuid = "cccc0001-0001-4000-8000-000000000001"
					name             = "n"
					mount_path       = "/m"
				`),
			},
			{
				ResourceName:  "coolify_storage.test",
				ImportState:   true,
				ImportStateId: "bad-format",
				ExpectError:   regexp.MustCompile(`Invalid import ID format`),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestStorageResource_ImportBadType
// ---------------------------------------------------------------------------

func TestStorageResource_ImportBadType(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/{appUUID}/storages", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": "stor-err2-uuid"})
	})
	mux.HandleFunc("GET /api/v1/applications/{appUUID}/storages", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string][]client.Storage{"persistent_storages": {{UUID: "stor-err2-uuid", Name: "n", MountPath: "/m"}}, "file_storages": {}})
	})
	mux.HandleFunc("DELETE /api/v1/applications/{appUUID}/storages/{storUUID}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testStorageResourceConfig(srv.URL, `
					application_uuid = "cccc0001-0001-4000-8000-000000000001"
					name             = "n"
					mount_path       = "/m"
				`),
			},
			{
				ResourceName:  "coolify_storage.test",
				ImportState:   true,
				ImportStateId: "unknown:aaaa0001-0001-4000-8000-000000000001:bbbb0001-0001-4000-8000-000000000001",
				ExpectError:   regexp.MustCompile(`Invalid import ID type`),
			},
		},
	})
}

func TestStorageResource_ImportBadParentUUID(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/{appUUID}/storages", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": "stor-err3-uuid"})
	})
	mux.HandleFunc("GET /api/v1/applications/{appUUID}/storages", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string][]client.Storage{"persistent_storages": {{UUID: "stor-err3-uuid", Name: "n", MountPath: "/m"}}, "file_storages": {}})
	})
	mux.HandleFunc("DELETE /api/v1/applications/{appUUID}/storages/{storUUID}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testStorageResourceConfig(srv.URL, `
					application_uuid = "cccc0001-0001-4000-8000-000000000001"
					name             = "n"
					mount_path       = "/m"
				`),
			},
			{
				ResourceName:  "coolify_storage.test",
				ImportState:   true,
				ImportStateId: "application:not-a-uuid:bbbb0001-0001-4000-8000-000000000001",
				ExpectError:   regexp.MustCompile(`(?s)Invalid Import ID.*parent UUID segment`),
			},
		},
	})
}

func TestStorageResource_ImportBadStorageUUID(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/{appUUID}/storages", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": "stor-err4-uuid"})
	})
	mux.HandleFunc("GET /api/v1/applications/{appUUID}/storages", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string][]client.Storage{"persistent_storages": {{UUID: "stor-err4-uuid", Name: "n", MountPath: "/m"}}, "file_storages": {}})
	})
	mux.HandleFunc("DELETE /api/v1/applications/{appUUID}/storages/{storUUID}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testStorageResourceConfig(srv.URL, `
					application_uuid = "cccc0001-0001-4000-8000-000000000001"
					name             = "n"
					mount_path       = "/m"
				`),
			},
			{
				ResourceName:  "coolify_storage.test",
				ImportState:   true,
				ImportStateId: "application:aaaa0001-0001-4000-8000-000000000001:not-a-uuid",
				ExpectError:   regexp.MustCompile(`(?s)Invalid Import ID.*storage UUID segment`),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestStorageResource_CreateWithServiceUUID
// ---------------------------------------------------------------------------

func TestStorageResource_CreateWithServiceUUID(t *testing.T) {
	t.Parallel()
	svcUUID := "ffff0001-0001-4000-8000-000000000001"
	resourceUUID := "eeee0001-0001-4000-8000-000000000001"
	stor := client.Storage{
		UUID:      "stor-svc-uuid",
		Name:      "svc-data",
		MountPath: "/data",
	}

	mu := sync.Mutex{}
	deleted := false

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/services/{svcUUID}/storages", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("svcUUID") != svcUUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, `{"error":"bad request body"}`, http.StatusBadRequest)
			return
		}
		if body["name"] != "svc-data" || body["mount_path"] != "/data" {
			t.Errorf("POST body mismatch: got %v", body)
			http.Error(w, `{"error":"unexpected fields"}`, http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": stor.UUID})
	})
	mux.HandleFunc("GET /api/v1/services/{svcUUID}/storages", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("svcUUID") != svcUUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if deleted {
			json.NewEncoder(w).Encode(map[string][]client.Storage{"persistent_storages": {}, "file_storages": {}})
		} else {
			json.NewEncoder(w).Encode(map[string][]client.Storage{"persistent_storages": {stor}, "file_storages": {}})
		}
	})
	mux.HandleFunc("DELETE /api/v1/services/{svcUUID}/storages/{storUUID}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("svcUUID") != svcUUID || r.PathValue("storUUID") != stor.UUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		mu.Lock()
		deleted = true
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	config := testStorageResourceConfig(srv.URL, fmt.Sprintf(`
		service_uuid  = %q
		resource_uuid = %q
		name          = "svc-data"
		mount_path    = "/data"
	`, svcUUID, resourceUUID))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_storage.test", "uuid", "stor-svc-uuid"),
					resource.TestCheckResourceAttr("coolify_storage.test", "name", "svc-data"),
					resource.TestCheckResourceAttr("coolify_storage.test", "mount_path", "/data"),
					resource.TestCheckResourceAttr("coolify_storage.test", "service_uuid", svcUUID),
					resource.TestCheckResourceAttr("coolify_storage.test", "resource_uuid", resourceUUID),
				),
			},
			{
				Config:             config,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestStorageResource_CreateWithDatabaseUUID
// ---------------------------------------------------------------------------

func TestStorageResource_CreateWithDatabaseUUID(t *testing.T) {
	t.Parallel()
	dbUUID := "dddd0002-0002-4000-8000-000000000002"
	stor := client.Storage{
		UUID:      "stor-db-uuid",
		Name:      "db-data",
		MountPath: "/var/lib/data",
	}

	mu := sync.Mutex{}
	deleted := false

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/databases/{dbUUID}/storages", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("dbUUID") != dbUUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": stor.UUID})
	})
	mux.HandleFunc("GET /api/v1/databases/{dbUUID}/storages", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("dbUUID") != dbUUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if deleted {
			json.NewEncoder(w).Encode(map[string][]client.Storage{"persistent_storages": {}, "file_storages": {}})
		} else {
			json.NewEncoder(w).Encode(map[string][]client.Storage{"persistent_storages": {stor}, "file_storages": {}})
		}
	})
	mux.HandleFunc("DELETE /api/v1/databases/{dbUUID}/storages/{storUUID}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("dbUUID") != dbUUID || r.PathValue("storUUID") != stor.UUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		mu.Lock()
		deleted = true
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	config := testStorageResourceConfig(srv.URL, fmt.Sprintf(`
		database_uuid = %q
		name          = "db-data"
		mount_path    = "/var/lib/data"
	`, dbUUID))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_storage.test", "uuid", "stor-db-uuid"),
					resource.TestCheckResourceAttr("coolify_storage.test", "name", "db-data"),
					resource.TestCheckResourceAttr("coolify_storage.test", "mount_path", "/var/lib/data"),
					resource.TestCheckResourceAttr("coolify_storage.test", "database_uuid", dbUUID),
				),
			},
			{
				Config:             config,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestStorageResource_CreateWithServiceUUIDMissingResourceUUID
// ---------------------------------------------------------------------------

func TestStorageResource_CreateWithServiceUUIDMissingResourceUUID(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.NewServeMux()))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testStorageResourceConfig(srv.URL, `
					service_uuid = "ffff0001-0001-4000-8000-000000000001"
					name         = "svc-data"
					mount_path   = "/data"
				`),
				ExpectError: regexp.MustCompile(`Missing resource_uuid for service storage`),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestStorageResource_ImportServiceRecoversResourceUUID
// ---------------------------------------------------------------------------

func TestStorageResource_ImportServiceRecoversResourceUUID(t *testing.T) {
	t.Parallel()
	svcUUID := "ffff0002-0002-4000-8000-000000000002"
	resourceUUID := "eeee0002-0002-4000-8000-000000000002"
	stor := client.Storage{
		UUID:         "dddd0003-0003-4000-8000-000000000003",
		Name:         "svc-import-data",
		MountPath:    "/data",
		ResourceUUID: resourceUUID,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/services/{svcUUID}/storages", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("svcUUID") != svcUUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": stor.UUID})
	})
	mux.HandleFunc("GET /api/v1/services/{svcUUID}/storages", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("svcUUID") != svcUUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string][]client.Storage{"persistent_storages": {stor}, "file_storages": {}})
	})
	mux.HandleFunc("DELETE /api/v1/services/{svcUUID}/storages/{storUUID}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("svcUUID") != svcUUID || r.PathValue("storUUID") != stor.UUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testStorageResourceConfig(srv.URL, fmt.Sprintf(`
					service_uuid  = %q
					resource_uuid = %q
					name          = "svc-import-data"
					mount_path    = "/data"
				`, svcUUID, resourceUUID)),
			},
			{
				ResourceName:                         "coolify_storage.test",
				ImportState:                          true,
				ImportStateId:                        fmt.Sprintf("service:%s:%s", svcUUID, stor.UUID),
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestStorageResource_Disappears
// ---------------------------------------------------------------------------

func TestStorageResource_Disappears(t *testing.T) {
	t.Parallel()
	stor := client.Storage{
		UUID:      "stor-disappear-uuid",
		Name:      "vanishing-vol",
		MountPath: "/data/vanish",
	}

	mu := sync.Mutex{}
	deleted := false

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/{appUUID}/storages", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("appUUID") != "cccc0001-0001-4000-8000-000000000001" {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": stor.UUID})
	})
	mux.HandleFunc("GET /api/v1/applications/{appUUID}/storages", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("appUUID") != "cccc0001-0001-4000-8000-000000000001" {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if deleted {
			json.NewEncoder(w).Encode(map[string][]client.Storage{"persistent_storages": {}, "file_storages": {}})
		} else {
			json.NewEncoder(w).Encode(map[string][]client.Storage{"persistent_storages": {stor}, "file_storages": {}})
		}
	})
	mux.HandleFunc("DELETE /api/v1/applications/{appUUID}/storages/{storUUID}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("appUUID") != "cccc0001-0001-4000-8000-000000000001" || r.PathValue("storUUID") != stor.UUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		deleted = true
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testStorageResourceConfig(srv.URL, `
					application_uuid = "cccc0001-0001-4000-8000-000000000001"
					name             = "vanishing-vol"
					mount_path       = "/data/vanish"
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_storage.test", "uuid"),
					acctest.CheckResourceDisappears(srv.URL, "coolify_storage.test", "/api/v1/applications/cccc0001-0001-4000-8000-000000000001/storages/"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestStorageResource_ReadParentNotFound
// ---------------------------------------------------------------------------

func TestStorageResource_ReadParentNotFound(t *testing.T) {
	t.Parallel()
	stor := client.Storage{
		UUID:      "stor-readnf-uuid",
		Name:      "readnf-vol",
		MountPath: "/data/readnf",
	}

	var forceNotFound atomic.Bool

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/{appUUID}/storages", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": stor.UUID})
	})
	mux.HandleFunc("GET /api/v1/applications/{appUUID}/storages", func(w http.ResponseWriter, _ *http.Request) {
		if forceNotFound.Load() {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string][]client.Storage{"persistent_storages": {stor}, "file_storages": {}})
	})
	mux.HandleFunc("DELETE /api/v1/applications/{appUUID}/storages/{storUUID}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testStorageResourceConfig(srv.URL, `
					application_uuid = "cccc0001-0001-4000-8000-000000000001"
					name             = "readnf-vol"
					mount_path       = "/data/readnf"
				`),
				Check: resource.TestCheckResourceAttrSet("coolify_storage.test", "uuid"),
			},
			{
				PreConfig: func() {
					forceNotFound.Store(true)
				},
				Config: testStorageResourceConfig(srv.URL, `
					application_uuid = "cccc0001-0001-4000-8000-000000000001"
					name             = "readnf-vol"
					mount_path       = "/data/readnf"
				`),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestStorageResource_ReadClientError
// ---------------------------------------------------------------------------

func TestStorageResource_ReadClientError(t *testing.T) {
	t.Parallel()
	stor := client.Storage{
		UUID:      "stor-readerr-uuid",
		Name:      "readerr-vol",
		MountPath: "/data/readerr",
	}

	var forceReadError atomic.Bool

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/{appUUID}/storages", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": stor.UUID})
	})
	mux.HandleFunc("GET /api/v1/applications/{appUUID}/storages", func(w http.ResponseWriter, _ *http.Request) {
		if forceReadError.Load() {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string][]client.Storage{"persistent_storages": {stor}, "file_storages": {}})
	})
	mux.HandleFunc("DELETE /api/v1/applications/{appUUID}/storages/{storUUID}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testStorageResourceConfig(srv.URL, `
					application_uuid = "cccc0001-0001-4000-8000-000000000001"
					name             = "readerr-vol"
					mount_path       = "/data/readerr"
				`),
				Check: resource.TestCheckResourceAttrSet("coolify_storage.test", "uuid"),
			},
			{
				PreConfig: func() {
					forceReadError.Store(true)
				},
				Config: testStorageResourceConfig(srv.URL, `
					application_uuid = "cccc0001-0001-4000-8000-000000000001"
					name             = "readerr-vol"
					mount_path       = "/data/readerr"
				`),
				ExpectError: regexp.MustCompile(`Error reading persistent storages`),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// checkStorageDestroy verifies that all coolify_storage resources have been
// removed from the mock server. The standard acctest.CheckDestroy helper does
// a GET to an individual-resource endpoint, but the storage mock only exposes
// a list endpoint, so we check the list instead.
func checkStorageDestroy(serverURL, listPath string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "coolify_storage" {
				continue
			}
			uuid := rs.Primary.Attributes["uuid"]
			if uuid == "" {
				continue
			}
			resp, err := http.Get(serverURL + listPath)
			if err != nil {
				return fmt.Errorf("error checking destroy for coolify_storage/%s: %w", uuid, err)
			}
			defer resp.Body.Close()
			var result struct {
				PersistentStorages []client.Storage `json:"persistent_storages"`
				FileStorages       []client.Storage `json:"file_storages"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return fmt.Errorf("error decoding destroy-check response for coolify_storage/%s: %w", uuid, err)
			}
			for _, stor := range result.PersistentStorages {
				if stor.UUID == uuid {
					return fmt.Errorf("coolify_storage %s still exists in persistent_storages", uuid)
				}
			}
			for _, stor := range result.FileStorages {
				if stor.UUID == uuid {
					return fmt.Errorf("coolify_storage %s still exists in file_storages", uuid)
				}
			}
		}
		return nil
	}
}

// TestStorageResource_NamePrefixStripping verifies that when Coolify returns
// a storage name prefixed with an internal resource UUID (for example,
// "resource-uuid-my-storage"), the provider preserves the user's original
// name to avoid a perpetual diff.
func TestStorageResource_NamePrefixStripping(t *testing.T) {
	t.Parallel()
	appUUID := "cccc0002-0002-4000-8000-000000000002"
	storUUID := "stor-prefix-uuid"
	internalResourceUUID := "resource-prefix-uuid"
	userName := "app-data"
	prefixedName := internalResourceUUID + "-" + userName

	mu := sync.Mutex{}
	deleted := false

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/{appUUID}/storages", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("appUUID") != appUUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": storUUID})
	})
	mux.HandleFunc("GET /api/v1/applications/{appUUID}/storages", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("appUUID") != appUUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if deleted {
			json.NewEncoder(w).Encode(map[string][]client.Storage{"persistent_storages": {}, "file_storages": {}})
		} else {
			// API returns the prefixed name.
			json.NewEncoder(w).Encode(map[string][]client.Storage{
				"persistent_storages": {{UUID: storUUID, Name: prefixedName, MountPath: "/data"}},
				"file_storages":       {},
			})
		}
	})
	mux.HandleFunc("DELETE /api/v1/applications/{appUUID}/storages/{storUUID}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("appUUID") != appUUID || r.PathValue("storUUID") != storUUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		mu.Lock()
		deleted = true
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	config := testStorageResourceConfig(srv.URL, fmt.Sprintf(`
		application_uuid = "%s"
		name             = "%s"
		mount_path       = "/data"
	`, appUUID, userName))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_storage.test", "name", userName),
				),
			},
			{
				Config:             config,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func testStorageResourceConfig(endpoint, attrs string) string {
	return acctest.TestResourceConfig(endpoint, "coolify_storage", "test", attrs)
}
