package service_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestServiceResource_DeleteBackupsTrue(t *testing.T) {
	t.Parallel()
	const uuid = "dddd0001-0001-4000-8000-000000000099"
	var mu sync.Mutex
	var deleteQuery string
	deleted := false

	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/services":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
				return
			}
			if _, ok := body["delete_backups"]; ok {
				t.Errorf("create sent delete_backups: %#v", body["delete_backups"])
				http.Error(w, `{"error":"delete_backups is destroy-only"}`, http.StatusUnprocessableEntity)
				return
			}
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]string{"uuid": uuid})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/services/"+uuid:
			mu.Lock()
			gone := deleted
			mu.Unlock()
			if gone {
				http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"uuid":             uuid,
				"name":             "backup-svc",
				"project_uuid":     "aaaa0001-0001-4000-8000-000000000001",
				"server_uuid":      "bbbb0001-0001-4000-8000-000000000001",
				"environment_name": "production",
				"type":             "plausible",
			})
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/services/"+uuid:
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
				return
			}
			if _, ok := body["delete_backups"]; ok {
				t.Errorf("update sent delete_backups: %#v", body["delete_backups"])
				http.Error(w, `{"error":"delete_backups is destroy-only"}`, http.StatusUnprocessableEntity)
				return
			}
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]string{"uuid": uuid})
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/services/"+uuid:
			mu.Lock()
			deleteQuery = r.URL.RawQuery
			deleted = true
			mu.Unlock()
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "deleted"})
		default:
			http.NotFound(w, r)
		}
	}), "4.8.0"))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.CheckDestroy(srv.URL, "coolify_service", "/api/v1/services/"),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_service" "test" {
  project_uuid   = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid    = "bbbb0001-0001-4000-8000-000000000001"
  type           = "plausible"
  delete_backups = true
}
`,
				Check: resource.TestCheckResourceAttr("coolify_service.test", "delete_backups", "true"),
			},
		},
	})

	mu.Lock()
	defer mu.Unlock()
	if deleteQuery != "delete_backups=true" {
		t.Fatalf("DELETE query = %q", deleteQuery)
	}
}
