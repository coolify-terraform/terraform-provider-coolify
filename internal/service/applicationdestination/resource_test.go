package applicationdestination_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestApplicationDestinationResource_Attach(t *testing.T) {
	t.Parallel()
	const (
		appUUID  = "aaaa0001-0001-4000-8000-000000000001"
		destUUID = "dddd0001-0001-4000-8000-000000000001"
	)
	attached := false
	var mu sync.Mutex
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/destinations"):
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode attach body: %v", err)
			}
			if body["destination_uuid"] != destUUID {
				t.Errorf("expected destination_uuid %s, got %v", destUUID, body)
			}
			attached = true
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "ok"})
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/destinations"):
			out := []any{
				map[string]any{"uuid": "primary-dest", "is_primary": true},
			}
			if attached {
				out = append(out, map[string]any{"uuid": destUUID, "is_primary": false})
			}
			_ = json.NewEncoder(w).Encode(out)
		case r.Method == http.MethodDelete:
			attached = false
			w.WriteHeader(http.StatusOK)
		default:
			http.Error(w, r.URL.Path, http.StatusNotFound)
		}
	})))
	defer srv.Close()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_application_destination" "test" {
  application_uuid = "` + appUUID + `"
  destination_uuid = "` + destUUID + `"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_application_destination.test", "application_uuid", appUUID),
					resource.TestCheckResourceAttr("coolify_application_destination.test", "destination_uuid", destUUID),
				),
			},
			{
				ResourceName:  "coolify_application_destination.test",
				ImportState:   true,
				ImportStateId: appUUID + ":" + destUUID,
			},
		},
	})
}

func TestApplicationDestinationResource_Disappears(t *testing.T) {
	t.Parallel()
	const (
		appUUID  = "aaaa0001-0001-4000-8000-000000000001"
		destUUID = "dddd0001-0001-4000-8000-000000000001"
	)
	attached := false
	var mu sync.Mutex
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/destinations"):
			attached = true
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "ok"})
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/destinations"):
			out := []any{
				map[string]any{"uuid": "primary-dest", "is_primary": true},
			}
			if attached {
				out = append(out, map[string]any{"uuid": destUUID, "is_primary": false})
			}
			_ = json.NewEncoder(w).Encode(out)
		case r.Method == http.MethodDelete:
			attached = false
			w.WriteHeader(http.StatusOK)
		default:
			http.Error(w, r.URL.Path, http.StatusNotFound)
		}
	})))
	defer srv.Close()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_application_destination" "test" {
  application_uuid = "` + appUUID + `"
  destination_uuid = "` + destUUID + `"
}`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("coolify_application_destination.test", "destination_uuid", destUUID),
				acctest.CheckPathDisappears(srv.URL, "/api/v1/applications/"+appUUID+"/destinations/"+destUUID),
			),
			ExpectNonEmptyPlan: true,
		}},
	})
}
