package serverdockercleanup_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestServerDockerCleanupResource_CRUD(t *testing.T) {
	t.Parallel()
	const serverUUID = "aaaa0001-0001-4000-8000-000000000001"
	store := map[string]any{
		"docker_cleanup_frequency":            "0 0 * * *",
		"docker_cleanup_threshold":            float64(80),
		"force_docker_cleanup":                false,
		"delete_unused_volumes":               false,
		"delete_unused_networks":              false,
		"disable_application_image_retention": false,
	}
	var mu sync.Mutex
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if !strings.HasSuffix(r.URL.Path, "/docker-cleanup") {
			http.Error(w, r.URL.Path, http.StatusNotFound)
			return
		}
		switch r.Method {
		case http.MethodPatch:
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode docker cleanup patch: %v", err)
			}
			if body["docker_cleanup_frequency"] == nil {
				t.Errorf("expected docker_cleanup_frequency in PATCH body, got %v", body)
			}
			for k, v := range body {
				store[k] = v
			}
			_ = json.NewEncoder(w).Encode(store)
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(store)
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
resource "coolify_server_docker_cleanup" "test" {
  server_uuid              = "` + serverUUID + `"
  docker_cleanup_frequency = "@daily"
  docker_cleanup_threshold = 70
  force_docker_cleanup     = true
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_server_docker_cleanup.test", "docker_cleanup_frequency", "@daily"),
					resource.TestCheckResourceAttr("coolify_server_docker_cleanup.test", "docker_cleanup_threshold", "70"),
					resource.TestCheckResourceAttr("coolify_server_docker_cleanup.test", "force_docker_cleanup", "true"),
				),
			},
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_server_docker_cleanup" "test" {
  server_uuid              = "` + serverUUID + `"
  docker_cleanup_frequency = "daily"
  docker_cleanup_threshold = 80
  force_docker_cleanup     = true
}`,
				Check: resource.TestCheckResourceAttr("coolify_server_docker_cleanup.test", "docker_cleanup_frequency", "daily"),
			},
			{
				ResourceName:                         "coolify_server_docker_cleanup.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateId:                        serverUUID,
				ImportStateVerifyIdentifierAttribute: "server_uuid",
			},
		},
	})
}

// Destroy leaves the remote Docker cleanup schedule in place (no DELETE API).
func TestServerDockerCleanupResource_DestroyLeavesRemote(t *testing.T) {
	t.Parallel()
	const serverUUID = "aaaa0001-0001-4000-8000-000000000001"
	store := map[string]any{
		"docker_cleanup_frequency": "0 0 * * *",
		"docker_cleanup_threshold": float64(80),
	}
	var mu sync.Mutex
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if !strings.HasSuffix(r.URL.Path, "/docker-cleanup") {
			http.Error(w, r.URL.Path, http.StatusNotFound)
			return
		}
		switch r.Method {
		case http.MethodPatch:
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			for k, v := range body {
				store[k] = v
			}
			_ = json.NewEncoder(w).Encode(store)
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(store)
		default:
			http.Error(w, r.URL.Path, http.StatusNotFound)
		}
	})))
	defer srv.Close()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy: func(_ *terraform.State) error {
			mu.Lock()
			defer mu.Unlock()
			if store["docker_cleanup_threshold"] != float64(70) {
				return fmt.Errorf("destroy must leave remote cleanup threshold, got %v", store["docker_cleanup_threshold"])
			}
			return nil
		},
		Steps: []resource.TestStep{{
			Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_server_docker_cleanup" "test" {
  server_uuid              = "` + serverUUID + `"
  docker_cleanup_frequency = "@daily"
  docker_cleanup_threshold = 70
}`,
			Check: resource.TestCheckResourceAttr("coolify_server_docker_cleanup.test", "docker_cleanup_threshold", "70"),
		}},
	})
}

func TestServerDockerCleanupResource_CreateAPIError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/docker-cleanup") && r.Method == http.MethodPatch {
			http.Error(w, `{"message":"Validation failed."}`, http.StatusUnprocessableEntity)
			return
		}
		http.Error(w, r.URL.Path, http.StatusNotFound)
	})))
	defer srv.Close()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_server_docker_cleanup" "test" {
  server_uuid              = "aaaa0001-0001-4000-8000-000000000001"
  docker_cleanup_frequency = "@daily"
  docker_cleanup_threshold = 70
}`,
			ExpectError: regexp.MustCompile(`Error applying Docker cleanup schedule`),
		}},
	})
}
