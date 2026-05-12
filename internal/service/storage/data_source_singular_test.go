package storage_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"regexp"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/acctest"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestStorageDataSource_Application(t *testing.T) {
	t.Parallel()

	storages := []client.Storage{
		{UUID: "st-1", Name: "vol-a", MountPath: "/data/a", HostPath: "/host/a"},
		{UUID: "st-2", Name: "vol-b", MountPath: "/data/b"},
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
  uuid             = "st-1"
  application_uuid = "cccc0001-0001-4000-8000-000000000001"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_storage.test", "uuid", "st-1"),
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
		{UUID: "st-s1", Name: "svc-config", MountPath: "/etc/config", HostPath: "/opt/config"},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/services/{svcUUID}/storages", func(w http.ResponseWriter, r *http.Request) {
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
  uuid         = "st-s1"
  service_uuid = "ffff0001-0001-4000-8000-000000000001"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_storage.test", "uuid", "st-s1"),
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
		{UUID: "st-d1", Name: "db-data", MountPath: "/var/lib/postgresql"},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/databases/{dbUUID}/storages", func(w http.ResponseWriter, r *http.Request) {
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
  uuid          = "st-d1"
  database_uuid = "dddd0001-0001-4000-8000-000000000001"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_storage.test", "uuid", "st-d1"),
					resource.TestCheckResourceAttr("data.coolify_storage.test", "name", "db-data"),
					resource.TestCheckResourceAttr("data.coolify_storage.test", "mount_path", "/var/lib/postgresql"),
					resource.TestCheckResourceAttr("data.coolify_storage.test", "host_path", ""),
				),
			},
		},
	})
}

func TestStorageDataSource_NotFound(t *testing.T) {
	t.Parallel()

	storages := []client.Storage{
		{UUID: "st-1", Name: "vol-a", MountPath: "/data/a"},
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
  uuid             = "nonexistent-uuid"
  application_uuid = "cccc0001-0001-4000-8000-000000000001"
}
`,
				ExpectError: regexp.MustCompile(`not found`),
			},
		},
	})
}
