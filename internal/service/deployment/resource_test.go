package deployment_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/service/deployment"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestMain(m *testing.M) {
	deployment.SetPollIntervalForTest(100 * time.Millisecond)
	os.Exit(m.Run())
}

func requireDeployQueryUUID(w http.ResponseWriter, r *http.Request, expectedAppUUID string) bool {
	if r.URL.Query().Get("uuid") == expectedAppUUID {
		return true
	}

	http.Error(w, `{"message":"No resources found."}`, http.StatusNotFound)
	return false
}

func writeDeployQueued(w http.ResponseWriter, appUUID, deploymentUUID string) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"deployments": []map[string]string{{
			"message":         "Application deployment queued.",
			"resource_uuid":   appUUID,
			"deployment_uuid": deploymentUUID,
		}},
	})
}

func TestDeploymentResource_Create(t *testing.T) {
	t.Parallel()
	deploymentUUID := "aaaa0001-0001-4000-8000-000000000001"
	appUUID := "cccc0002-0002-4000-8000-000000000002"

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/deploy", func(w http.ResponseWriter, r *http.Request) {
		if !requireDeployQueryUUID(w, r, appUUID) {
			return
		}
		writeDeployQueued(w, appUUID, deploymentUUID)
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
	mux.HandleFunc("POST /api/v1/deploy", func(w http.ResponseWriter, r *http.Request) {
		if !requireDeployQueryUUID(w, r, appUUID) {
			return
		}
		mu.Lock()
		deploymentCount++
		uuid := fmt.Sprintf("dep-uuid-%d", deploymentCount)
		mu.Unlock()
		writeDeployQueued(w, appUUID, uuid)
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
	mux.HandleFunc("POST /api/v1/deploy", func(w http.ResponseWriter, r *http.Request) {
		if !requireDeployQueryUUID(w, r, appUUID) {
			return
		}
		writeDeployQueued(w, appUUID, deploymentUUID)
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
	mux.HandleFunc("POST /api/v1/deploy", func(w http.ResponseWriter, r *http.Request) {
		if !requireDeployQueryUUID(w, r, appUUID) {
			return
		}
		writeDeployQueued(w, appUUID, deploymentUUID)
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
	mux.HandleFunc("POST /api/v1/deploy", func(w http.ResponseWriter, r *http.Request) {
		if !requireDeployQueryUUID(w, r, appUUID) {
			return
		}
		writeDeployQueued(w, appUUID, deploymentUUID)
	})
	mux.HandleFunc("GET /api/v1/deployments/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != appUUID {
			http.Error(w, `{"message":"Application not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"count": 0, "deployments": []any{}})
	})
	mux.HandleFunc("GET /api/v1/deployments/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		count := readBackCalls.Add(1)
		// Fail resolve GET (4 attempts) plus Create status read-back
		// (4 more) so the "default to queued" fallback still runs.
		// Succeed afterward so the post-apply refresh Read works.
		if count <= 8 {
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
	mux.HandleFunc("POST /api/v1/deploy", func(w http.ResponseWriter, r *http.Request) {
		if !requireDeployQueryUUID(w, r, appUUID) {
			return
		}
		writeDeployQueued(w, appUUID, deploymentUUID)
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

func TestDeploymentResource_WaitForCompletionFailed(t *testing.T) {
	t.Parallel()

	for _, failureStatus := range []string{"failed", "error"} {
		t.Run(failureStatus, func(t *testing.T) {
			t.Parallel()
			deploymentUUID := "wait-err-0001-4000-8000-000000000001"
			appUUID := "cccc0004-0004-4000-8000-000000000004"

			mu := sync.Mutex{}
			getCount := 0

			mux := http.NewServeMux()
			mux.HandleFunc("POST /api/v1/deploy", func(w http.ResponseWriter, r *http.Request) {
				if !requireDeployQueryUUID(w, r, appUUID) {
					return
				}
				writeDeployQueued(w, appUUID, deploymentUUID)
			})
			mux.HandleFunc("GET /api/v1/deployments/{uuid}", func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				getCount++
				n := getCount
				mu.Unlock()
				uuid := r.PathValue("uuid")
				// Coolify uses ApplicationDeploymentStatus::FAILED ("failed")
				// for deployment failures. The provider also handles "error"
				// (ProcessStatus) for safety. This test exercises both.
				status := "in_progress"
				if n >= 3 {
					status = failureStatus
				}
				w.Header().Set("Content-Type", "application/json")
				body := map[string]interface{}{
					"deployment_uuid": uuid,
					"status":          status,
				}
				if status == failureStatus {
					body["logs"] = []map[string]interface{}{
						{"output": "cloning repository", "hidden": false},
						{"output": "nixpacks build failed: no start command", "hidden": false},
					}
				}
				json.NewEncoder(w).Encode(body)
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
						ExpectError: regexp.MustCompile(`nixpacks build failed: no start command`),
					},
				},
			})
		})
	}
}

func TestDeploymentResource_WaitForCompletionTimeout(t *testing.T) {
	t.Parallel()
	deploymentUUID := "wait-timeout-0001-4000-8000-000000000001"
	appUUID := "cccc0005-0005-4000-8000-000000000005"

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/deploy", func(w http.ResponseWriter, r *http.Request) {
		if !requireDeployQueryUUID(w, r, appUUID) {
			return
		}
		writeDeployQueued(w, appUUID, deploymentUUID)
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

func TestDeploymentResource_ImportBadFormat(t *testing.T) {
	t.Parallel()
	deploymentUUID := "bbbb0002-0002-4000-8000-000000000002"
	appUUID := "cccc0002-0002-4000-8000-000000000002"

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/deploy", func(w http.ResponseWriter, r *http.Request) {
		if !requireDeployQueryUUID(w, r, appUUID) {
			return
		}
		writeDeployQueued(w, appUUID, deploymentUUID)
	})
	mux.HandleFunc("GET /api/v1/deployments/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"deployment_uuid": r.PathValue("uuid"),
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
			},
			{
				ResourceName:  "coolify_deployment.test",
				ImportState:   true,
				ImportStateId: "just-a-single-uuid",
				ExpectError:   regexp.MustCompile(`Invalid import ID format`),
			},
		},
	})
}

func TestDeploymentResource_ImportBadUUID(t *testing.T) {
	t.Parallel()
	deploymentUUID := "bbbb0003-0003-4000-8000-000000000003"
	appUUID := "cccc0003-0003-4000-8000-000000000003"

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/deploy", func(w http.ResponseWriter, r *http.Request) {
		if !requireDeployQueryUUID(w, r, appUUID) {
			return
		}
		writeDeployQueued(w, appUUID, deploymentUUID)
	})
	mux.HandleFunc("GET /api/v1/deployments/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"deployment_uuid": r.PathValue("uuid"),
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
			},
			{
				ResourceName:  "coolify_deployment.test",
				ImportState:   true,
				ImportStateId: "not-valid:also-not-valid",
				ExpectError:   regexp.MustCompile(`Invalid Import ID`),
			},
		},
	})
}

func TestDeploymentResource_ImportBadDeploymentUUID(t *testing.T) {
	t.Parallel()
	deploymentUUID := "bbbb0004-0004-4000-8000-000000000004"
	appUUID := "cccc0004-0004-4000-8000-000000000004"

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/deploy", func(w http.ResponseWriter, r *http.Request) {
		if !requireDeployQueryUUID(w, r, appUUID) {
			return
		}
		writeDeployQueued(w, appUUID, deploymentUUID)
	})
	mux.HandleFunc("GET /api/v1/deployments/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"deployment_uuid": r.PathValue("uuid"),
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
			},
			{
				ResourceName:  "coolify_deployment.test",
				ImportState:   true,
				ImportStateId: appUUID + ":not-a-uuid",
				ExpectError:   regexp.MustCompile(`(?s)Invalid Import ID.*deployment UUID segment`),
			},
		},
	})
}

func TestDeploymentResource_UpdateWaitForCompletion(t *testing.T) {
	t.Parallel()
	deploymentUUID := "aaaa0006-0006-4000-8000-000000000006"
	appUUID := "cccc0006-0006-4000-8000-000000000006"

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/deploy", func(w http.ResponseWriter, r *http.Request) {
		if !requireDeployQueryUUID(w, r, appUUID) {
			return
		}
		writeDeployQueued(w, appUUID, deploymentUUID)
	})
	mux.HandleFunc("GET /api/v1/deployments/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"deployment_uuid": r.PathValue("uuid"),
			"status":          "finished",
		})
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			// Create with wait_for_completion = false (default)
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
				Check: resource.TestCheckResourceAttr("coolify_deployment.test", "wait_for_completion", "false"),
			},
			// Update wait_for_completion without changing triggers (should not error)
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
				Check: resource.TestCheckResourceAttr("coolify_deployment.test", "wait_for_completion", "true"),
			},
			// Idempotency
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
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestDeploymentResource_EmptyUUID(t *testing.T) {
	t.Parallel()
	appUUID := "cccc0007-0007-4000-8000-000000000007"

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/deploy", func(w http.ResponseWriter, r *http.Request) {
		if !requireDeployQueryUUID(w, r, appUUID) {
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"deployments": []map[string]string{{
				"message":       "Deployment already queued for this commit.",
				"resource_uuid": appUUID,
			}},
		})
	})
	mux.HandleFunc("GET /api/v1/deployments/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != appUUID {
			http.Error(w, `{"message":"Application not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"count": 0, "deployments": []any{}})
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
				ExpectError: regexp.MustCompile("Deployment triggered but no UUID returned"),
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

// ---------------------------------------------------------------------------
// TestDeploymentResource_CreateAPIError
// ---------------------------------------------------------------------------

func TestDeploymentResource_DestroyNoOp(t *testing.T) {
	t.Parallel()
	deploymentUUID := "destroy-nop-0001-4000-8000-000000000001"
	appUUID := "cccc0008-0008-4000-8000-000000000008"
	var restartCalls atomic.Int32

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/deploy", func(w http.ResponseWriter, r *http.Request) {
		if !requireDeployQueryUUID(w, r, appUUID) {
			return
		}
		restartCalls.Add(1)
		writeDeployQueued(w, appUUID, deploymentUUID)
	})
	mux.HandleFunc("GET /api/v1/deployments/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"deployment_uuid": r.PathValue("uuid"),
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
			},
			acctest.DestroyRemoveResourceStep(srv.URL),
		},
	})
	if restartCalls.Load() != 1 {
		t.Fatalf("expected deploy only on create, got %d calls", restartCalls.Load())
	}
}

func TestDeploymentResource_AdoptsQueuedDeploy(t *testing.T) {
	t.Parallel()
	appUUID := "cccc0009-0009-4000-8000-000000000009"
	existingUUID := "existing-queued-0001-4000-8000-000000000001"

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/deploy", func(w http.ResponseWriter, r *http.Request) {
		if !requireDeployQueryUUID(w, r, appUUID) {
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"deployments": []map[string]string{{
				"message":       "Deployment already queued for this commit.",
				"resource_uuid": appUUID,
			}},
		})
	})
	mux.HandleFunc("GET /api/v1/deployments/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != appUUID {
			http.Error(w, `{"message":"Application not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"count": 1,
			"deployments": []map[string]any{{
				"deployment_uuid": existingUUID,
				"status":          "in_progress",
			}},
		})
	})
	mux.HandleFunc("GET /api/v1/deployments/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != existingUUID {
			http.Error(w, `{"message":"Deployment not found."}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"deployment_uuid": existingUUID,
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
  application_uuid = %q
}
`, srv.URL, appUUID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_deployment.test", "uuid", existingUUID),
					resource.TestCheckResourceAttr("coolify_deployment.test", "status", "in_progress"),
				),
			},
		},
	})
}

func TestDeploymentResource_PhantomDeployUUID(t *testing.T) {
	t.Parallel()
	appUUID := "cccc0010-0010-4000-8000-000000000010"
	phantomUUID := "phantom-never-persisted-0001-4000-8000-0001"
	existingUUID := "existing-queued-0002-4000-8000-000000000002"

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/deploy", func(w http.ResponseWriter, r *http.Request) {
		if !requireDeployQueryUUID(w, r, appUUID) {
			return
		}
		writeDeployQueued(w, appUUID, phantomUUID)
	})
	mux.HandleFunc("GET /api/v1/deployments/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != appUUID {
			http.Error(w, `{"message":"Application not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"count": 1,
			"deployments": []map[string]any{{
				"deployment_uuid": existingUUID,
				"status":          "queued",
			}},
		})
	})
	mux.HandleFunc("GET /api/v1/deployments/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		uuid := r.PathValue("uuid")
		if uuid != existingUUID {
			http.Error(w, `{"message":"Deployment not found."}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
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
					resource.TestCheckResourceAttr("coolify_deployment.test", "uuid", existingUUID),
				),
			},
		},
	})
}

func TestDeploymentResource_TransientGetAdoptsQueued(t *testing.T) {
	t.Parallel()
	appUUID := "cccc0011-0011-4000-8000-000000000011"
	phantomUUID := "phantom-never-persisted-0002-4000-8000-0002"
	existingUUID := "existing-queued-0003-4000-8000-000000000003"

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/deploy", func(w http.ResponseWriter, r *http.Request) {
		if !requireDeployQueryUUID(w, r, appUUID) {
			return
		}
		writeDeployQueued(w, appUUID, phantomUUID)
	})
	mux.HandleFunc("GET /api/v1/deployments/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != appUUID {
			http.Error(w, `{"message":"Application not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"count": 1,
			"deployments": []map[string]any{{
				"deployment_uuid": existingUUID,
				"status":          "queued",
			}},
		})
	})
	mux.HandleFunc("GET /api/v1/deployments/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		uuid := r.PathValue("uuid")
		if uuid == phantomUUID {
			http.Error(w, `{"message":"temporary failure"}`, http.StatusUnprocessableEntity)
			return
		}
		if uuid != existingUUID {
			http.Error(w, `{"message":"Deployment not found."}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
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
					resource.TestCheckResourceAttr("coolify_deployment.test", "uuid", existingUUID),
				),
			},
		},
	})
}

func TestDeploymentResource_CreateAPIError(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/deploy", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"message":"validation failed"}`, http.StatusUnprocessableEntity)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_deployment" "test" {
  application_uuid = "550e8400-e29b-41d4-a716-446655440001"
}
`,
				ExpectError: regexp.MustCompile(`Error triggering deployment`),
			},
		},
	})
}
