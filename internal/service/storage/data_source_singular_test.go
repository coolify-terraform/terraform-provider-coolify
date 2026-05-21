package storage_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/acctest"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestStorageDataSource_Application(t *testing.T) {
	t.Parallel()

	storages := []client.Storage{
		{UUID: "11111111-1111-4111-8111-111111111111", Name: "vol-a", MountPath: "/data/a", HostPath: "/host/a"},
		{UUID: "22222222-2222-4222-8222-222222222222", Name: "vol-b", MountPath: "/data/b"},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/applications/{appUUID}/storages", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("appUUID") != "cccc0001-0001-4000-8000-000000000001" {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string][]client.Storage{"persistent_storages": storages, "file_storages": {}})
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
data "coolify_storage" "test" {
  uuid             = "11111111-1111-4111-8111-111111111111"
  application_uuid = "cccc0001-0001-4000-8000-000000000001"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_storage.test", "uuid", "11111111-1111-4111-8111-111111111111"),
					resource.TestCheckResourceAttr("data.coolify_storage.test", "name", "vol-a"),
					resource.TestCheckResourceAttr("data.coolify_storage.test", "mount_path", "/data/a"),
					resource.TestCheckResourceAttr("data.coolify_storage.test", "host_path", "/host/a"),
				),
			},
		},
	})
}

func TestStorageDataSource_Service(t *testing.T) {
	t.Parallel()

	storages := []client.Storage{
		{UUID: "33333333-3333-4333-8333-333333333333", Name: "svc-config", MountPath: "/etc/config", HostPath: "/opt/config"},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/services/{svcUUID}/storages", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("svcUUID") != "ffff0001-0001-4000-8000-000000000001" {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string][]client.Storage{"persistent_storages": storages, "file_storages": {}})
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
data "coolify_storage" "test" {
  uuid         = "33333333-3333-4333-8333-333333333333"
  service_uuid = "ffff0001-0001-4000-8000-000000000001"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_storage.test", "uuid", "33333333-3333-4333-8333-333333333333"),
					resource.TestCheckResourceAttr("data.coolify_storage.test", "name", "svc-config"),
					resource.TestCheckResourceAttr("data.coolify_storage.test", "mount_path", "/etc/config"),
					resource.TestCheckResourceAttr("data.coolify_storage.test", "host_path", "/opt/config"),
				),
			},
		},
	})
}

func TestStorageDataSource_Database(t *testing.T) {
	t.Parallel()

	storages := []client.Storage{
		{UUID: "44444444-4444-4444-8444-444444444444", Name: "db-data", MountPath: "/var/lib/postgresql"},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/databases/{dbUUID}/storages", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("dbUUID") != "dddd0001-0001-4000-8000-000000000001" {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string][]client.Storage{"persistent_storages": storages, "file_storages": {}})
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
data "coolify_storage" "test" {
  uuid          = "44444444-4444-4444-8444-444444444444"
  database_uuid = "dddd0001-0001-4000-8000-000000000001"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_storage.test", "uuid", "44444444-4444-4444-8444-444444444444"),
					resource.TestCheckResourceAttr("data.coolify_storage.test", "name", "db-data"),
					resource.TestCheckResourceAttr("data.coolify_storage.test", "mount_path", "/var/lib/postgresql"),
					resource.TestCheckResourceAttr("data.coolify_storage.test", "host_path", ""),
				),
			},
		},
	})
}

func TestStorageDataSource_InvalidUUID(t *testing.T) {
	t.Parallel()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: `data "coolify_storage" "test" {
  uuid             = "not-a-valid-uuid"
  application_uuid = "cccc0001-0001-4000-8000-000000000001"
}
`,
			ExpectError: acctest.UUIDValidationError(),
		}},
	})
}

func TestStorageDataSource_NotFound(t *testing.T) {
	t.Parallel()

	storages := []client.Storage{
		{UUID: "11111111-1111-4111-8111-111111111111", Name: "vol-a", MountPath: "/data/a"},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/applications/{appUUID}/storages", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string][]client.Storage{"persistent_storages": storages, "file_storages": {}})
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
data "coolify_storage" "test" {
  uuid             = "55555555-5555-4555-8555-555555555555"
  application_uuid = "cccc0001-0001-4000-8000-000000000001"
}
`,
				ExpectError: acctest.NotFoundError(),
			},
		},
	})
}
