package storage_test

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sync"

	"github.com/SebTardif/terraform-provider-coolify/internal/acctest"
	"github.com/SebTardif/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
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

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/{appUUID}/storages", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": stor.UUID})
	})
	mux.HandleFunc("GET /api/v1/applications/{appUUID}/storages", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]client.Storage{stor})
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
					name             = "app-data"
					mount_path       = "/data"
					host_path        = "/host/data"
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_storage.test", "uuid", "stor-create-uuid"),
					resource.TestCheckResourceAttr("coolify_storage.test", "name", "app-data"),
					resource.TestCheckResourceAttr("coolify_storage.test", "mount_path", "/data"),
					resource.TestCheckResourceAttr("coolify_storage.test", "host_path", "/host/data"),
				),
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
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": currentStor.UUID})
	})
	mux.HandleFunc("GET /api/v1/applications/{appUUID}/storages", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]client.Storage{currentStor})
	})
	mux.HandleFunc("PATCH /api/v1/applications/{appUUID}/storages", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if v, ok := body["name"].(string); ok {
			currentStor.Name = v
		}
		if v, ok := body["mount_path"].(string); ok {
			currentStor.MountPath = v
		}
		w.WriteHeader(http.StatusOK)
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
		},
	})
}

// ---------------------------------------------------------------------------
// TestStorageResource_Import
// ---------------------------------------------------------------------------

func TestStorageResource_Import(t *testing.T) {
	t.Parallel()
	stor := client.Storage{
		UUID:      "stor-import-uuid",
		Name:      "import-vol",
		MountPath: "/mnt/import",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/{appUUID}/storages", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": stor.UUID})
	})
	mux.HandleFunc("GET /api/v1/applications/{appUUID}/storages", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]client.Storage{stor})
	})
	mux.HandleFunc("DELETE /api/v1/applications/{appUUID}/storages/{storUUID}", func(w http.ResponseWriter, _ *http.Request) {
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
				ImportStateId:                        "application:cccc0001-0001-4000-8000-000000000001:stor-import-uuid",
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
		json.NewEncoder(w).Encode([]client.Storage{{UUID: "stor-err-uuid", Name: "n", MountPath: "/m"}})
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
		json.NewEncoder(w).Encode([]client.Storage{{UUID: "stor-err2-uuid", Name: "n", MountPath: "/m"}})
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
				ImportStateId: "unknown:uuid:stor-uuid",
				ExpectError:   regexp.MustCompile(`Invalid import ID type`),
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
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": stor.UUID})
	})
	mux.HandleFunc("GET /api/v1/applications/{appUUID}/storages", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if deleted {
			json.NewEncoder(w).Encode([]client.Storage{})
		} else {
			json.NewEncoder(w).Encode([]client.Storage{stor})
		}
	})
	mux.HandleFunc("DELETE /api/v1/applications/{appUUID}/storages/{storUUID}", func(w http.ResponseWriter, _ *http.Request) {
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
// Helpers
// ---------------------------------------------------------------------------

func testStorageResourceConfig(endpoint, attrs string) string {
	return fmt.Sprintf(`
provider "coolify" {
  endpoint  = %q
  token = "test-token"
}

resource "coolify_storage" "test" {
  %s
}
`, endpoint, attrs)
}
