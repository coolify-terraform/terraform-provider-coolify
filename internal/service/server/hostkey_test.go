package server_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestServerResource_SSHHostKeyAppliedWhenUnpinned(t *testing.T) {
	t.Parallel()
	const hostKey = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITestKeyForCoolify47"
	var mu sync.Mutex
	version := "4.6.0"
	resets := 0
	srvState := &client.Server{
		UUID:           "bbbb0001-0001-4000-8000-000000000001",
		Name:           "my-server",
		IP:             "10.0.0.1",
		Port:           22,
		User:           "root",
		PrivateKeyUUID: "dddd0002-0002-4000-8000-000000000002",
		IsReachable:    true,
		IsUsable:       true,
		Settings: &client.ServerSettings{
			ConcurrentBuilds:                     2,
			DynamicTimeout:                       3600,
			DeploymentQueueLimit:                 25,
			ConnectionTimeout:                    10,
			ServerDiskUsageNotificationThreshold: 80,
			ServerDiskUsageCheckFrequency:        "*/5 * * * *",
			IsTerminalEnabled:                    true,
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		v := version
		mu.Unlock()
		_, _ = w.Write([]byte(v))
	})
	mux.HandleFunc("POST /api/v1/servers", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		mu.Lock()
		defer mu.Unlock()
		_ = json.NewEncoder(w).Encode(srvState)
	})
	mux.HandleFunc("POST /api/v1/servers/{uuid}/host-key/reset", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		resets++
		srvState.Settings.SSHHostKeyFingerprints = []string{"SHA256:abc"}
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"message":"Host key reset.","ssh_host_key_fingerprints":["SHA256:abc"]}`))
	})
	mux.HandleFunc("GET /api/v1/servers/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		mu.Lock()
		defer mu.Unlock()
		_ = json.NewEncoder(w).Encode(srvState)
	})
	mux.HandleFunc("PATCH /api/v1/servers/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		mu.Lock()
		defer mu.Unlock()
		_ = json.NewEncoder(w).Encode(srvState)
	})
	mux.HandleFunc("DELETE /api/v1/servers/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	config := acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_server" "test" {
  name             = "my-server"
  ip               = "10.0.0.1"
  private_key_uuid = "dddd0002-0002-4000-8000-000000000002"
  ssh_host_key     = "` + hostKey + `"
}
`
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{Config: config},
			{
				PreConfig: func() {
					mu.Lock()
					version = "4.7.1"
					mu.Unlock()
				},
				Config: config,
			},
		},
	})
	mu.Lock()
	defer mu.Unlock()
	if resets != 1 {
		t.Fatalf("host-key reset calls = %d, want 1", resets)
	}
}
