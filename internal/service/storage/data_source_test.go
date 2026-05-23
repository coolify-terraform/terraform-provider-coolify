package storage_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestStoragesDataSource_Application(t *testing.T) {
	t.Parallel()
	storages := []client.Storage{
		{UUID: "st-1", Name: "vol-a", MountPath: "/data/a", HostPath: "/host/a"},
		{UUID: "st-2", Name: "vol-b", MountPath: "/data/b"},
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
data "coolify_storages" "test" {
  application_uuid = "cccc0001-0001-4000-8000-000000000001"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_storages.test", "storages.#", "2"),
					resource.TestCheckResourceAttr("data.coolify_storages.test", "storages.0.name", "vol-a"),
					resource.TestCheckResourceAttr("data.coolify_storages.test", "storages.0.mount_path", "/data/a"),
					resource.TestCheckResourceAttr("data.coolify_storages.test", "storages.0.host_path", "/host/a"),
					resource.TestCheckResourceAttr("data.coolify_storages.test", "storages.1.name", "vol-b"),
					resource.TestCheckResourceAttr("data.coolify_storages.test", "storages.1.mount_path", "/data/b"),
				),
			},
		},
	})
}

func TestStoragesDataSource_Database(t *testing.T) {
	t.Parallel()
	storages := []client.Storage{
		{UUID: "st-d1", Name: "db-data", MountPath: "/var/lib/postgresql"},
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
data "coolify_storages" "test" {
  database_uuid = "dddd0001-0001-4000-8000-000000000001"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_storages.test", "storages.#", "1"),
					resource.TestCheckResourceAttr("data.coolify_storages.test", "storages.0.name", "db-data"),
					resource.TestCheckResourceAttr("data.coolify_storages.test", "storages.0.mount_path", "/var/lib/postgresql"),
				),
			},
		},
	})
}

func TestStoragesDataSource_InvalidUUID(t *testing.T) {
	t.Parallel()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: `data "coolify_storages" "test" {
  application_uuid = "not-a-valid-uuid"
}
`,
			ExpectError: acctest.UUIDValidationError(),
		}},
	})
}

func TestStoragesDataSource_Service(t *testing.T) {
	t.Parallel()
	storages := []client.Storage{
		{UUID: "st-s1", Name: "svc-config", MountPath: "/etc/config", HostPath: "/opt/config"},
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
data "coolify_storages" "test" {
  service_uuid = "ffff0001-0001-4000-8000-000000000001"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_storages.test", "storages.#", "1"),
					resource.TestCheckResourceAttr("data.coolify_storages.test", "storages.0.name", "svc-config"),
					resource.TestCheckResourceAttr("data.coolify_storages.test", "storages.0.host_path", "/opt/config"),
				),
			},
		},
	})
}
