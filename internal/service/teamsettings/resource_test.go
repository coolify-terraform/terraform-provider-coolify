package teamsettings_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestTeamSettingsResource_CreateSendsScanFlag(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	var patched map[string]any

	mux := http.NewServeMux()
	mux.HandleFunc("PATCH /api/v1/team", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
			http.Error(w, `{"message":"bad request"}`, http.StatusBadRequest)
			return
		}
		var decoded map[string]any
		if err := json.Unmarshal(body, &decoded); err != nil {
			t.Errorf("decode body: %v", err)
			http.Error(w, `{"message":"bad request"}`, http.StatusBadRequest)
			return
		}
		mu.Lock()
		if patched == nil {
			patched = decoded
		}
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":                         1,
			"name":                       "root",
			"scan_images_on_new_servers": true,
		})
	})
	mux.HandleFunc("GET /api/v1/teams/current", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":                         1,
			"name":                       "root",
			"scan_images_on_new_servers": true,
		})
	})

	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(mux, "4.7.1"))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.TestResourceConfig(srv.URL, "coolify_team_settings", "test", `
				scan_images_on_new_servers = true
			`),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("coolify_team_settings.test", "id", "current"),
				resource.TestCheckResourceAttr("coolify_team_settings.test", "scan_images_on_new_servers", "true"),
			),
		}},
	})

	mu.Lock()
	defer mu.Unlock()
	if patched == nil {
		t.Fatal("PATCH /team was not sent")
	}
	if patched["scan_images_on_new_servers"] != true {
		t.Fatalf("scan_images_on_new_servers = %v", patched["scan_images_on_new_servers"])
	}
}

func TestTeamSettingsResource_Disappears(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	gone := false
	mux := http.NewServeMux()
	mux.HandleFunc("PATCH /api/v1/team", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":1,"scan_images_on_new_servers":true}`))
	})
	mux.HandleFunc("GET /api/v1/teams/current", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		missing := gone
		mu.Unlock()
		if missing {
			http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":1,"name":"root","scan_images_on_new_servers":true}`))
	})
	mux.HandleFunc("DELETE /api/v1/teams/current", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gone = true
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(mux, "4.7.1"))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.TestResourceConfig(srv.URL, "coolify_team_settings", "test", `
				scan_images_on_new_servers = true
			`),
			Check: resource.ComposeAggregateTestCheckFunc(
				acctest.CheckPathDisappears(srv.URL, "/api/v1/teams/current"),
			),
			ExpectNonEmptyPlan: true,
		}},
	})
}
