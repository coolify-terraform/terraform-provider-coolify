package application_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestApplicationLogsDataSource_ContainerNotRunning(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/applications/{uuid}/logs", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"message":"container is not running"}`)
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

data "coolify_application_logs" "test" {
  uuid = "550e8400-e29b-41d4-a716-446655440000"
}
`, srv.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_application_logs.test", "logs.#", "0"),
				),
			},
		},
	})
}

func TestApplicationLogsDataSource_UnexpectedError(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/applications/{uuid}/logs", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		fmt.Fprintf(w, `{"message":"unexpected validation error"}`)
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

data "coolify_application_logs" "test" {
  uuid = "550e8400-e29b-41d4-a716-446655440000"
}
`, srv.URL),
				ExpectError: regexp.MustCompile(`Error reading application logs`),
			},
		},
	})
}

func TestApplicationLogsDataSource_EmptyLogs(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/applications/{uuid}/logs", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]client.ApplicationLog{})
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

data "coolify_application_logs" "test" {
  uuid = "550e8400-e29b-41d4-a716-446655440000"
}
`, srv.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_application_logs.test", "logs.#", "0"),
				),
			},
		},
	})
}

func TestApplicationLogsDataSource(t *testing.T) {
	t.Parallel()
	logs := []client.ApplicationLog{
		{Line: "Starting application...", Timestamp: "2024-01-01T00:00:00Z"},
		{Line: "Application started successfully", Timestamp: "2024-01-01T00:00:01Z"},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/applications/{uuid}/logs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(logs)
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

data "coolify_application_logs" "test" {
  uuid = "550e8400-e29b-41d4-a716-446655440000"
}
`, srv.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_application_logs.test", "logs.#", "2"),
					resource.TestCheckResourceAttr("data.coolify_application_logs.test", "logs.0.line", "Starting application..."),
					resource.TestCheckResourceAttr("data.coolify_application_logs.test", "logs.0.timestamp", "2024-01-01T00:00:00Z"),
					resource.TestCheckResourceAttr("data.coolify_application_logs.test", "logs.1.line", "Application started successfully"),
					resource.TestCheckResourceAttr("data.coolify_application_logs.test", "logs.1.timestamp", "2024-01-01T00:00:01Z"),
				),
			},
		},
	})
}
