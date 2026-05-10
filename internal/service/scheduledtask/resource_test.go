package scheduledtask_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"testing"

	"github.com/SebTardif/terraform-provider-coolify/internal/acctest"
	"github.com/SebTardif/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// ---------------------------------------------------------------------------
// TestScheduledTaskResource_Create
// ---------------------------------------------------------------------------

func TestScheduledTaskResource_Create(t *testing.T) {
	t.Parallel()

	task := client.ScheduledTask{
		UUID:      "task-create-uuid",
		Name:      "backup-db",
		Command:   "pg_dump mydb > /backups/mydb.sql",
		Frequency: "*/5 * * * *",
		Enabled:   true,
	}

	mu := sync.Mutex{}
	deleted := false

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/{appUUID}/scheduled-tasks", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": task.UUID})
	})
	mux.HandleFunc("GET /api/v1/applications/{appUUID}/scheduled-tasks/{taskUUID}", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if deleted {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(task)
	})
	mux.HandleFunc("GET /api/v1/applications/{appUUID}/scheduled-tasks", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if deleted {
			json.NewEncoder(w).Encode([]client.ScheduledTask{})
		} else {
			json.NewEncoder(w).Encode([]client.ScheduledTask{task})
		}
	})
	mux.HandleFunc("DELETE /api/v1/applications/{appUUID}/scheduled-tasks/{taskUUID}", func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		deleted = true
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.CheckDestroy(srv.URL, "coolify_scheduled_task", "/api/v1/applications/cccc0001-0001-4000-8000-000000000001/scheduled-tasks/"),
		Steps: []resource.TestStep{
			{
				Config: testScheduledTaskResourceConfig(srv.URL, `
					application_uuid = "cccc0001-0001-4000-8000-000000000001"
					name             = "backup-db"
					command          = "pg_dump mydb > /backups/mydb.sql"
					frequency        = "*/5 * * * *"
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_scheduled_task.test", "uuid", "task-create-uuid"),
					resource.TestCheckResourceAttr("coolify_scheduled_task.test", "name", "backup-db"),
					resource.TestCheckResourceAttr("coolify_scheduled_task.test", "command", "pg_dump mydb > /backups/mydb.sql"),
					resource.TestCheckResourceAttr("coolify_scheduled_task.test", "frequency", "*/5 * * * *"),
					resource.TestCheckResourceAttr("coolify_scheduled_task.test", "enabled", "true"),
				),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestScheduledTaskResource_Update
// ---------------------------------------------------------------------------

func TestScheduledTaskResource_Update(t *testing.T) {
	t.Parallel()

	mu := sync.Mutex{}
	currentTask := client.ScheduledTask{
		UUID:      "task-update-uuid",
		Name:      "cleanup-logs",
		Command:   "rm -rf /tmp/logs/*",
		Frequency: "0 * * * *",
		Enabled:   true,
	}
	deleted := false

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/{appUUID}/scheduled-tasks", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": currentTask.UUID})
	})
	mux.HandleFunc("GET /api/v1/applications/{appUUID}/scheduled-tasks/{taskUUID}", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if deleted {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(currentTask)
	})
	mux.HandleFunc("GET /api/v1/applications/{appUUID}/scheduled-tasks", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if deleted {
			json.NewEncoder(w).Encode([]client.ScheduledTask{})
		} else {
			json.NewEncoder(w).Encode([]client.ScheduledTask{currentTask})
		}
	})
	mux.HandleFunc("PATCH /api/v1/applications/{appUUID}/scheduled-tasks/{taskUUID}", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if v, ok := body["name"].(string); ok {
			currentTask.Name = v
		}
		if v, ok := body["command"].(string); ok {
			currentTask.Command = v
		}
		if v, ok := body["frequency"].(string); ok {
			currentTask.Frequency = v
		}
		if v, ok := body["enabled"].(bool); ok {
			currentTask.Enabled = v
		}
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("DELETE /api/v1/applications/{appUUID}/scheduled-tasks/{taskUUID}", func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		deleted = true
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.CheckDestroy(srv.URL, "coolify_scheduled_task", "/api/v1/applications/cccc0001-0001-4000-8000-000000000001/scheduled-tasks/"),
		Steps: []resource.TestStep{
			{
				Config: testScheduledTaskResourceConfig(srv.URL, `
					application_uuid = "cccc0001-0001-4000-8000-000000000001"
					name             = "cleanup-logs"
					command          = "rm -rf /tmp/logs/*"
					frequency        = "0 * * * *"
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_scheduled_task.test", "name", "cleanup-logs"),
					resource.TestCheckResourceAttr("coolify_scheduled_task.test", "command", "rm -rf /tmp/logs/*"),
					resource.TestCheckResourceAttr("coolify_scheduled_task.test", "enabled", "true"),
				),
			},
			{
				Config: testScheduledTaskResourceConfig(srv.URL, `
					application_uuid = "cccc0001-0001-4000-8000-000000000001"
					name             = "cleanup-old-logs"
					command          = "find /tmp/logs -mtime +7 -delete"
					frequency        = "0 0 * * *"
					enabled          = false
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_scheduled_task.test", "name", "cleanup-old-logs"),
					resource.TestCheckResourceAttr("coolify_scheduled_task.test", "command", "find /tmp/logs -mtime +7 -delete"),
					resource.TestCheckResourceAttr("coolify_scheduled_task.test", "frequency", "0 0 * * *"),
					resource.TestCheckResourceAttr("coolify_scheduled_task.test", "enabled", "false"),
				),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestScheduledTaskResource_Import
// ---------------------------------------------------------------------------

func TestScheduledTaskResource_Import(t *testing.T) {
	t.Parallel()

	task := client.ScheduledTask{
		UUID:      "task-import-uuid",
		Name:      "import-task",
		Command:   "echo hello",
		Frequency: "0 0 * * *",
		Enabled:   true,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/{appUUID}/scheduled-tasks", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": task.UUID})
	})
	mux.HandleFunc("GET /api/v1/applications/{appUUID}/scheduled-tasks", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]client.ScheduledTask{task})
	})
	mux.HandleFunc("DELETE /api/v1/applications/{appUUID}/scheduled-tasks/{taskUUID}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			// Step 1: create so the resource exists in state.
			{
				Config: testScheduledTaskResourceConfig(srv.URL, `
					application_uuid = "cccc0001-0001-4000-8000-000000000001"
					name             = "import-task"
					command          = "echo hello"
					frequency        = "0 0 * * *"
				`),
			},
			// Step 2: import and verify.
			{
				ResourceName:                         "coolify_scheduled_task.test",
				ImportState:                          true,
				ImportStateId:                        "application:cccc0001-0001-4000-8000-000000000001:task-import-uuid",
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestScheduledTaskResource_ImportService
// ---------------------------------------------------------------------------

func TestScheduledTaskResource_ImportService(t *testing.T) {
	t.Parallel()

	task := client.ScheduledTask{
		UUID:      "task-svc-imp-uuid",
		Name:      "svc-import-task",
		Command:   "echo service",
		Frequency: "0 0 * * *",
		Enabled:   true,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/services/{svcUUID}/scheduled-tasks", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": task.UUID})
	})
	mux.HandleFunc("GET /api/v1/services/{svcUUID}/scheduled-tasks", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]client.ScheduledTask{task})
	})
	mux.HandleFunc("DELETE /api/v1/services/{svcUUID}/scheduled-tasks/{taskUUID}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testScheduledTaskResourceConfig(srv.URL, `
					service_uuid = "ffff0001-0001-4000-8000-000000000001"
					name         = "svc-import-task"
					command      = "echo service"
					frequency    = "0 0 * * *"
				`),
			},
			{
				ResourceName:                         "coolify_scheduled_task.test",
				ImportState:                          true,
				ImportStateId:                        "service:ffff0001-0001-4000-8000-000000000001:task-svc-imp-uuid",
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestScheduledTaskResource_ImportBadFormat
// ---------------------------------------------------------------------------

func TestScheduledTaskResource_ImportBadFormat(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/{appUUID}/scheduled-tasks", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": "task-err-uuid"})
	})
	mux.HandleFunc("GET /api/v1/applications/{appUUID}/scheduled-tasks", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]client.ScheduledTask{{UUID: "task-err-uuid", Name: "t", Command: "c", Frequency: "* * * * *", Enabled: true}})
	})
	mux.HandleFunc("DELETE /api/v1/applications/{appUUID}/scheduled-tasks/{taskUUID}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testScheduledTaskResourceConfig(srv.URL, `
					application_uuid = "cccc0001-0001-4000-8000-000000000001"
					name             = "t"
					command          = "c"
					frequency        = "* * * * *"
				`),
			},
			{
				ResourceName:  "coolify_scheduled_task.test",
				ImportState:   true,
				ImportStateId: "bad-format",
				ExpectError:   regexp.MustCompile(`Invalid import ID format`),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestScheduledTaskResource_ImportBadType
// ---------------------------------------------------------------------------

func TestScheduledTaskResource_ImportBadType(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/{appUUID}/scheduled-tasks", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": "task-err2-uuid"})
	})
	mux.HandleFunc("GET /api/v1/applications/{appUUID}/scheduled-tasks", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]client.ScheduledTask{{UUID: "task-err2-uuid", Name: "t", Command: "c", Frequency: "* * * * *", Enabled: true}})
	})
	mux.HandleFunc("DELETE /api/v1/applications/{appUUID}/scheduled-tasks/{taskUUID}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testScheduledTaskResourceConfig(srv.URL, `
					application_uuid = "cccc0001-0001-4000-8000-000000000001"
					name             = "t"
					command          = "c"
					frequency        = "* * * * *"
				`),
			},
			{
				ResourceName:  "coolify_scheduled_task.test",
				ImportState:   true,
				ImportStateId: "unknown:uuid:task-uuid",
				ExpectError:   regexp.MustCompile(`Invalid import ID type`),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestScheduledTaskResource_Disappears
// ---------------------------------------------------------------------------

func TestScheduledTaskResource_Disappears(t *testing.T) {
	t.Parallel()

	task := client.ScheduledTask{
		UUID:      "task-disappear-uuid",
		Name:      "disappear-task",
		Command:   "echo gone",
		Frequency: "* * * * *",
		Enabled:   true,
	}

	mu := sync.Mutex{}
	deleted := false

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/{appUUID}/scheduled-tasks", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": task.UUID})
	})
	mux.HandleFunc("GET /api/v1/applications/{appUUID}/scheduled-tasks", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if deleted {
			json.NewEncoder(w).Encode([]client.ScheduledTask{})
		} else {
			json.NewEncoder(w).Encode([]client.ScheduledTask{task})
		}
	})
	mux.HandleFunc("DELETE /api/v1/applications/{appUUID}/scheduled-tasks/{taskUUID}", func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		deleted = true
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testScheduledTaskResourceConfig(srv.URL, `
					application_uuid = "cccc0001-0001-4000-8000-000000000001"
					name             = "disappear-task"
					command          = "echo gone"
					frequency        = "* * * * *"
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_scheduled_task.test", "uuid"),
					acctest.CheckResourceDisappears(srv.URL, "coolify_scheduled_task.test", "/api/v1/applications/cccc0001-0001-4000-8000-000000000001/scheduled-tasks/"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestScheduledTaskResource_CreateWithServiceUUID
// ---------------------------------------------------------------------------

func TestScheduledTaskResource_CreateWithServiceUUID(t *testing.T) {
	t.Parallel()

	task := client.ScheduledTask{
		UUID:      "task-svc-uuid",
		Name:      "service-task",
		Command:   "curl http://localhost/health",
		Frequency: "*/10 * * * *",
		Enabled:   true,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/services/{svcUUID}/scheduled-tasks", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": task.UUID})
	})
	mux.HandleFunc("GET /api/v1/services/{svcUUID}/scheduled-tasks", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]client.ScheduledTask{task})
	})
	mux.HandleFunc("DELETE /api/v1/services/{svcUUID}/scheduled-tasks/{taskUUID}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testScheduledTaskResourceConfig(srv.URL, `
					service_uuid = "ffff0001-0001-4000-8000-000000000001"
					name         = "service-task"
					command      = "curl http://localhost/health"
					frequency    = "*/10 * * * *"
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_scheduled_task.test", "uuid", "task-svc-uuid"),
					resource.TestCheckResourceAttr("coolify_scheduled_task.test", "name", "service-task"),
					resource.TestCheckResourceAttr("coolify_scheduled_task.test", "command", "curl http://localhost/health"),
					resource.TestCheckResourceAttr("coolify_scheduled_task.test", "frequency", "*/10 * * * *"),
					resource.TestCheckResourceAttr("coolify_scheduled_task.test", "enabled", "true"),
				),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestScheduledTaskResource_InvalidFrequency
// ---------------------------------------------------------------------------

func TestScheduledTaskResource_InvalidFrequency(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.NotFoundHandler()))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testScheduledTaskResourceConfig(srv.URL, `
					application_uuid = "cccc0001-0001-4000-8000-000000000001"
					name             = "bad-freq"
					command          = "echo test"
					frequency        = "not-a-cron"
				`),
				ExpectError: regexp.MustCompile(`must be a valid cron expression`),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestScheduledTaskResource_ServiceDisappears
// ---------------------------------------------------------------------------

func TestScheduledTaskResource_ServiceDisappears(t *testing.T) {
	t.Parallel()

	task := client.ScheduledTask{
		UUID: "task-svc-disappear-uuid", Name: "svc-gone", Command: "echo bye", Frequency: "* * * * *", Enabled: true,
	}

	mu := sync.Mutex{}
	deleted := false

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/services/{svcUUID}/scheduled-tasks", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": task.UUID})
	})
	mux.HandleFunc("GET /api/v1/services/{svcUUID}/scheduled-tasks", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if deleted {
			json.NewEncoder(w).Encode([]client.ScheduledTask{})
		} else {
			json.NewEncoder(w).Encode([]client.ScheduledTask{task})
		}
	})
	mux.HandleFunc("DELETE /api/v1/services/{svcUUID}/scheduled-tasks/{taskUUID}", func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		deleted = true
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testScheduledTaskResourceConfig(srv.URL, `
					service_uuid = "ffff0001-0001-4000-8000-000000000001"
					name         = "svc-gone"
					command      = "echo bye"
					frequency    = "* * * * *"
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_scheduled_task.test", "uuid"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["coolify_scheduled_task.test"]
						if !ok {
							return fmt.Errorf("resource not found in state")
						}
						uuid := rs.Primary.Attributes["uuid"]
						req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/api/v1/services/ffff0001-0001-4000-8000-000000000001/scheduled-tasks/"+uuid, nil)
						resp, err := http.DefaultClient.Do(req)
						if err != nil {
							return err
						}
						resp.Body.Close()
						return nil
					},
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func testScheduledTaskResourceConfig(endpoint, attrs string) string {
	return fmt.Sprintf(`
provider "coolify" {
  endpoint = %q
  token    = "test-token"
}

resource "coolify_scheduled_task" "test" {
  %s
}
`, endpoint, attrs)
}
