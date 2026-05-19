package resourceaction_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestResourceActionResource_StartDatabase(t *testing.T) {
	t.Parallel()
	dbUUID := "aaaa0001-0001-4000-8000-000000000001"

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/databases/{uuid}/start", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != dbUUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"message":"Database starting request queued."}`)
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

resource "coolify_resource_action" "start_db" {
  resource_uuid = %q
  resource_type = "database"
  action        = "start"
}
`, srv.URL, dbUUID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_resource_action.start_db", "resource_uuid", dbUUID),
					resource.TestCheckResourceAttr("coolify_resource_action.start_db", "resource_type", "database"),
					resource.TestCheckResourceAttr("coolify_resource_action.start_db", "action", "start"),
				),
			},
		},
	})
}

func TestResourceActionResource_StopService(t *testing.T) {
	t.Parallel()
	svcUUID := "bbbb0001-0001-4000-8000-000000000001"

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/services/{uuid}/stop", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != svcUUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"message":"Service stopping request queued."}`)
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

resource "coolify_resource_action" "stop_svc" {
  resource_uuid = %q
  resource_type = "service"
  action        = "stop"
}
`, srv.URL, svcUUID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_resource_action.stop_svc", "resource_uuid", svcUUID),
					resource.TestCheckResourceAttr("coolify_resource_action.stop_svc", "action", "stop"),
				),
			},
		},
	})
}

func TestResourceActionResource_RestartApplication(t *testing.T) {
	t.Parallel()
	appUUID := "cccc0001-0001-4000-8000-000000000001"

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/{uuid}/restart", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != appUUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"deployment_uuid":"deploy-001","message":"Restart queued."}`)
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

resource "coolify_resource_action" "restart_app" {
  resource_uuid = %q
  resource_type = "application"
  action        = "restart"
}
`, srv.URL, appUUID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_resource_action.restart_app", "resource_uuid", appUUID),
					resource.TestCheckResourceAttr("coolify_resource_action.restart_app", "action", "restart"),
				),
			},
		},
	})
}

func TestResourceActionResource_TriggersForceReplace(t *testing.T) {
	t.Parallel()
	dbUUID := "dddd0002-0002-4000-8000-000000000002"
	callCount := 0

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/databases/{uuid}/restart", func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"message":"Database restarting."}`)
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

resource "coolify_resource_action" "restart_db" {
  resource_uuid = %q
  resource_type = "database"
  action        = "restart"
  triggers = {
    version = "1"
  }
}
`, srv.URL, dbUUID),
			},
			{
				Config: fmt.Sprintf(`
provider "coolify" {
  endpoint = %q
  token    = "test-token"
}

resource "coolify_resource_action" "restart_db" {
  resource_uuid = %q
  resource_type = "database"
  action        = "restart"
  triggers = {
    version = "2"
  }
}
`, srv.URL, dbUUID),
			},
		},
	})
}

func TestResourceActionResource_InvalidAction(t *testing.T) {
	t.Parallel()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: `
provider "coolify" {
  endpoint = "http://localhost:8000"
  token    = "test-token"
}

resource "coolify_resource_action" "bad" {
  resource_uuid = "aaaa0001-0001-4000-8000-000000000001"
  resource_type = "database"
  action        = "delete"
}
`,
				ExpectError: regexp.MustCompile("delete"),
			},
		},
	})
}

func TestResourceActionResource_APIError(t *testing.T) {
	t.Parallel()
	dbUUID := "aaaa0001-0001-4000-8000-000000000001"

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/databases/{uuid}/start", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"error":"Server is not reachable."}`, http.StatusServiceUnavailable)
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

resource "coolify_resource_action" "start_db" {
  resource_uuid = %q
  resource_type = "database"
  action        = "start"
}
`, srv.URL, dbUUID),
				ExpectError: regexp.MustCompile(`Could not start database`),
			},
		},
	})
}

func TestResourceActionResource_AlreadyStopped(t *testing.T) {
	t.Parallel()
	dbUUID := "aaaa0003-0003-4000-8000-000000000003"

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/databases/{uuid}/stop", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"message":"Database is already stopped."}`)
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

resource "coolify_resource_action" "stop_db" {
  resource_uuid = %q
  resource_type = "database"
  action        = "stop"
}
`, srv.URL, dbUUID),
				Check: resource.TestCheckResourceAttr("coolify_resource_action.stop_db", "action", "stop"),
			},
		},
	})
}

func TestResourceActionResource_AlreadyRunning(t *testing.T) {
	t.Parallel()
	svcUUID := "aaaa0004-0004-4000-8000-000000000004"

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/services/{uuid}/start", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"message":"Service is already running."}`)
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

resource "coolify_resource_action" "start_svc" {
  resource_uuid = %q
  resource_type = "service"
  action        = "start"
}
`, srv.URL, svcUUID),
				Check: resource.TestCheckResourceAttr("coolify_resource_action.start_svc", "action", "start"),
			},
		},
	})
}

func TestResourceActionResource_InvalidResourceType(t *testing.T) {
	t.Parallel()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: `
provider "coolify" {
  endpoint = "http://localhost:8000"
  token    = "test-token"
}

resource "coolify_resource_action" "bad" {
  resource_uuid = "aaaa0001-0001-4000-8000-000000000001"
  resource_type = "container"
  action        = "start"
}
`,
				ExpectError: regexp.MustCompile("container"),
			},
		},
	})
}
