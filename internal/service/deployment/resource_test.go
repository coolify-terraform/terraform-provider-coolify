package deployment_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"testing"

	"github.com/SebTardif/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestDeploymentResource_Create(t *testing.T) {
	t.Parallel()
	deploymentUUID := "dep-test-uuid-001"
	appUUID := "cccc0002-0002-4000-8000-000000000002"

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
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
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
  application_uuid = "cccc0001-0001-4000-8000-000000000001"
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
  application_uuid = "cccc0001-0001-4000-8000-000000000001"
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

func TestDeploymentResource_Import(t *testing.T) {
	t.Parallel()
	deploymentUUID := "dep-import-uuid-001"
	appUUID := "cccc0001-0001-4000-8000-000000000001"

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
			"status": "finished",
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
}
`, srv.URL, appUUID),
				Check: resource.TestCheckResourceAttr("coolify_deployment.test", "uuid", deploymentUUID),
			},
			{
				ResourceName:                         "coolify_deployment.test",
				ImportState:                          true,
				ImportStateId:                        deploymentUUID,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateVerifyIgnore:              []string{"application_uuid", "triggers"},
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
  application_uuid = "cccc0001-0001-4000-8000-000000000001"
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

func TestDeploymentResource_InvalidUUID(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.NotFoundHandler()))
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
  application_uuid = "not-a-valid-uuid"
}
`, srv.URL),
				ExpectError: regexp.MustCompile(`must be a valid UUID`),
			},
		},
	})
}
