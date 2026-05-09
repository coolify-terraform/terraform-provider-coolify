package deployment_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/SebTardif/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestDeploymentResource_Create(t *testing.T) {
	t.Parallel()
	deploymentUUID := "dep-test-uuid-001"
	appUUID := "app-deploy-uuid"

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/{uuid}/restart", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"deployment_uuid": deploymentUUID,
			"message":         "Restart request queued.",
		})
	})
	mux.HandleFunc("GET /api/v1/deployments/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		uuid := r.PathValue("uuid")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"uuid":   uuid,
			"status": "queued",
		})
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
provider "coolify" {
  endpoint = %q
  token    = "test-token"
}

resource "coolify_deployment" "test" {
  application_uuid = %q
  triggers = {
    version = "1"
  }
}
`, srv.URL, appUUID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_deployment.test", "application_uuid", appUUID),
					resource.TestCheckResourceAttr("coolify_deployment.test", "uuid", deploymentUUID),
					resource.TestCheckResourceAttr("coolify_deployment.test", "status", "queued"),
					resource.TestCheckResourceAttr("coolify_deployment.test", "triggers.version", "1"),
				),
			},
		},
	})
}

func TestDeploymentResource_TriggersForceNew(t *testing.T) {
	t.Parallel()
	mu := sync.Mutex{}
	deploymentCount := 0

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/{uuid}/restart", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		deploymentCount++
		uuid := fmt.Sprintf("dep-uuid-%d", deploymentCount)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"deployment_uuid": uuid,
			"message":         "Restart request queued.",
		})
	})
	mux.HandleFunc("GET /api/v1/deployments/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		uuid := r.PathValue("uuid")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"uuid":   uuid,
			"status": "queued",
		})
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
provider "coolify" {
  endpoint = %q
  token    = "test-token"
}

resource "coolify_deployment" "test" {
  application_uuid = "app-uuid-1"
  triggers = {
    version = "1"
  }
}
`, srv.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_deployment.test", "uuid", "dep-uuid-1"),
					resource.TestCheckResourceAttr("coolify_deployment.test", "triggers.version", "1"),
				),
			},
			{
				Config: fmt.Sprintf(`
provider "coolify" {
  endpoint = %q
  token    = "test-token"
}

resource "coolify_deployment" "test" {
  application_uuid = "app-uuid-1"
  triggers = {
    version = "2"
  }
}
`, srv.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_deployment.test", "uuid", "dep-uuid-2"),
					resource.TestCheckResourceAttr("coolify_deployment.test", "triggers.version", "2"),
				),
			},
		},
	})
}

func TestDeploymentResource_Disappears(t *testing.T) {
	t.Parallel()
	deploymentUUID := "dep-disappear-uuid"

	mu := sync.Mutex{}
	deleted := false

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/{uuid}/restart", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"deployment_uuid": deploymentUUID,
			"message":         "Restart request queued.",
		})
	})
	mux.HandleFunc("GET /api/v1/deployments/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if deleted {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		uuid := r.PathValue("uuid")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"uuid":   uuid,
			"status": "queued",
		})
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
provider "coolify" {
  endpoint = %q
  token    = "test-token"
}

resource "coolify_deployment" "test" {
  application_uuid = "app-uuid-1"
}
`, srv.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_deployment.test", "uuid", deploymentUUID),
					func(s *terraform.State) error {
						mu.Lock()
						deleted = true
						mu.Unlock()
						return nil
					},
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
