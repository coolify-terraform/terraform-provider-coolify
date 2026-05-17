package deployment_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func requireRestartApplicationUUID(w http.ResponseWriter, r *http.Request, expectedAppUUID string) bool {
	if r.PathValue("uuid") == expectedAppUUID {
		return true
	}

	http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
	return false
}

func TestDeploymentResource_Create(t *testing.T) {
	t.Parallel()
	deploymentUUID := "aaaa0001-0001-4000-8000-000000000001"
	appUUID := "cccc0002-0002-4000-8000-000000000002"

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/{uuid}/restart", func(w http.ResponseWriter, r *http.Request) {
		if !requireRestartApplicationUUID(w, r, appUUID) {
			return
		}
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
			"deployment_uuid": uuid,
			"status":          "queued",
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
	appUUID := "cccc0001-0001-4000-8000-000000000001"
	mu := sync.Mutex{}
	deploymentCount := 0

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/{uuid}/restart", func(w http.ResponseWriter, r *http.Request) {
		if !requireRestartApplicationUUID(w, r, appUUID) {
			return
		}
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
			"deployment_uuid": uuid,
			"status":          "queued",
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
  application_uuid = %q
  triggers = {
    version = "2"
  }
}
`, srv.URL, appUUID),
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
	deploymentUUID := "bbbb0001-0001-4000-8000-000000000001"
	appUUID := "cccc0001-0001-4000-8000-000000000001"

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/{uuid}/restart", func(w http.ResponseWriter, r *http.Request) {
		if !requireRestartApplicationUUID(w, r, appUUID) {
			return
		}
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
			"deployment_uuid": uuid,
			"status":          "finished",
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
				ImportStateId:                        appUUID + ":" + deploymentUUID,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateVerifyIgnore:              []string{"triggers", "wait_for_completion", "status"},
			},
		},
	})
}

func TestDeploymentResource_Disappears(t *testing.T) {
	t.Parallel()
	appUUID := "cccc0001-0001-4000-8000-000000000001"
	deploymentUUID := "dep-disappear-uuid"

	mu := sync.Mutex{}
	deleted := false

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/{uuid}/restart", func(w http.ResponseWriter, r *http.Request) {
		if !requireRestartApplicationUUID(w, r, appUUID) {
			return
		}
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
			"deployment_uuid": uuid,
			"status":          "queued",
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

func TestDeploymentResource_CreateReadBackFailureDefaultsQueued(t *testing.T) {
	t.Parallel()
	deploymentUUID := "readback-fail-0001-4000-8000-000000000001"
	appUUID := "cccc0003-0003-4000-8000-000000000003"
	var readBackCalls atomic.Int32

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/{uuid}/restart", func(w http.ResponseWriter, r *http.Request) {
		if !requireRestartApplicationUUID(w, r, appUUID) {
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"deployment_uuid": deploymentUUID,
			"message":         "Restart request queued.",
		})
	})
	mux.HandleFunc("GET /api/v1/deployments/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		count := readBackCalls.Add(1)
		// Fail enough times to exhaust the client's 3 retries during
		// Create's read-back, exercising the "default to queued" fallback.
		// Succeed afterward so the post-apply refresh Read works.
		if count <= 4 {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"deployment_uuid": r.PathValue("uuid"),
			"status":          "queued",
		})
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	config := fmt.Sprintf(`
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
`, srv.URL, appUUID)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_deployment.test", "uuid", deploymentUUID),
					resource.TestCheckResourceAttr("coolify_deployment.test", "status", "queued"),
					resource.TestCheckResourceAttr("coolify_deployment.test", "triggers.version", "1"),
				),
			},
			{
				Config:             config,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestDeploymentResource_WaitForCompletion(t *testing.T) {
	t.Parallel()
	deploymentUUID := "wait-0001-0001-4000-8000-000000000001"
	appUUID := "cccc0003-0003-4000-8000-000000000003"

	mu := sync.Mutex{}
	getCount := 0

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/{uuid}/restart", func(w http.ResponseWriter, r *http.Request) {
		if !requireRestartApplicationUUID(w, r, appUUID) {
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"deployment_uuid": deploymentUUID,
			"message":         "Restart request queued.",
		})
	})
	mux.HandleFunc("GET /api/v1/deployments/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		getCount++
		n := getCount
		mu.Unlock()
		uuid := r.PathValue("uuid")
		status := "in_progress"
		if n >= 3 {
			status = "finished"
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"deployment_uuid": uuid,
			"status":          status,
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
  application_uuid    = %q
  wait_for_completion = true
  triggers = {
    version = "1"
  }
}
`, srv.URL, appUUID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_deployment.test", "uuid", deploymentUUID),
					resource.TestCheckResourceAttr("coolify_deployment.test", "status", "finished"),
					resource.TestCheckResourceAttr("coolify_deployment.test", "wait_for_completion", "true"),
				),
			},
		},
	})
}

func TestDeploymentResource_WaitForCompletionError(t *testing.T) {
	t.Parallel()
	deploymentUUID := "wait-err-0001-4000-8000-000000000001"
	appUUID := "cccc0004-0004-4000-8000-000000000004"

	mu := sync.Mutex{}
	getCount := 0

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/{uuid}/restart", func(w http.ResponseWriter, r *http.Request) {
		if !requireRestartApplicationUUID(w, r, appUUID) {
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"deployment_uuid": deploymentUUID,
			"message":         "Restart request queued.",
		})
	})
	mux.HandleFunc("GET /api/v1/deployments/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		getCount++
		n := getCount
		mu.Unlock()
		uuid := r.PathValue("uuid")
		status := "in_progress"
		if n >= 3 {
			status = "error"
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"deployment_uuid": uuid,
			"status":          status,
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
  application_uuid    = %q
  wait_for_completion = true
  triggers = {
    version = "1"
  }
}
`, srv.URL, appUUID),
				ExpectError: regexp.MustCompile(`Deployment failed`),
			},
		},
	})
}

func TestDeploymentResource_WaitForCompletionTimeout(t *testing.T) {
	t.Parallel()
	deploymentUUID := "wait-timeout-0001-4000-8000-000000000001"
	appUUID := "cccc0005-0005-4000-8000-000000000005"

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/{uuid}/restart", func(w http.ResponseWriter, r *http.Request) {
		if !requireRestartApplicationUUID(w, r, appUUID) {
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"deployment_uuid": deploymentUUID,
			"message":         "Restart request queued.",
		})
	})
	mux.HandleFunc("GET /api/v1/deployments/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"deployment_uuid": r.PathValue("uuid"),
			"status":          "in_progress",
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
  application_uuid    = %q
  wait_for_completion = true
  timeouts = {
    create = "1s"
  }
}
`, srv.URL, appUUID),
				ExpectError: regexp.MustCompile(`Deployment timed out`),
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
				ExpectError: acctest.UUIDValidationError(),
			},
		},
	})
}
