//go:build !ci_app_a

package application_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// Coolify rejects any create or PATCH whose domains name a hostname another
// resource already carries, unless that same request says
// force_domain_override: true. The flag is evaluated per request and is never
// stored or returned, so it has to travel with every request that writes
// domains. These tests stand up a fake Coolify that enforces exactly that rule,
// with a sibling application already holding sharedDomain — the load-balanced
// hostname several applications legitimately share.

const (
	sharedDomain  = "https://shared.example.com"
	ownDomain     = "https://own.example.com"
	forceAppUUID  = "force-domain-app-uuid"
	forceProject  = "aaaa0002-0002-4000-8000-000000000002"
	forceServer   = "bbbb0002-0002-4000-8000-000000000002"
	conflictError = `{"message":"Domain conflicts detected. Use force_domain_override=true to proceed.","conflicts":[{"domain":"https://shared.example.com","resource_name":"sibling","resource_type":"application"}]}`
)

// forceDomainServer is a minimal Coolify for coolify_application_docker_image
// that applies the domain-conflict rule to POST and PATCH.
type forceDomainServer struct {
	t       *testing.T
	mu      sync.Mutex
	app     client.Application
	created bool
	deleted bool
}

func (s *forceDomainServer) conflicts(body map[string]interface{}) bool {
	domains, _ := body["domains"].(string)
	if !strings.Contains(domains, sharedDomain) {
		return false
	}
	force, _ := body["force_domain_override"].(bool)
	return !force
}

func (s *forceDomainServer) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/dockerimage", func(w http.ResponseWriter, r *http.Request) {
		body, ok := decodeRequestBodyMap(s.t, w, r)
		if !ok {
			return
		}
		if s.conflicts(body) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(conflictError))
			return
		}
		s.mu.Lock()
		s.app = client.Application{
			UUID:                    forceAppUUID,
			Name:                    body["name"].(string),
			DockerRegistryImageName: body["docker_registry_image_name"].(string),
			PortsExposes:            body["ports_exposes"].(string),
			ProjectUUID:             forceProject,
			ServerUUID:              forceServer,
			EnvironmentName:         "production",
		}
		if d, ok := body["domains"].(string); ok {
			s.app.Domains = d
		}
		s.created = true
		s.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"uuid": forceAppUUID})
	})
	mux.HandleFunc("PATCH /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		body, ok := decodeRequestBodyMap(s.t, w, r)
		if !ok {
			return
		}
		if s.conflicts(body) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(conflictError))
			return
		}
		s.mu.Lock()
		if d, ok := body["domains"].(string); ok {
			s.app.Domains = d
		}
		s.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"uuid": forceAppUUID})
	})
	mux.HandleFunc("GET /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		defer s.mu.Unlock()
		if r.PathValue("uuid") != forceAppUUID || !s.created || s.deleted {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(s.app)
	})
	mux.HandleFunc("DELETE /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		s.deleted = true
		s.mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})
	return acctest.WithVersionEndpoint(mux)
}

func forceDomainConfig(endpoint, domains, extra string) string {
	return testDockerImageResourceConfig(endpoint, `
		name          = "force-domain"
		project_uuid  = "`+forceProject+`"
		server_uuid   = "`+forceServer+`"
		docker_image  = "nginx:alpine"
		ports_exposes = "80"
		domains       = "`+domains+`"
		`+extra)
}

// Creating an application on a hostname a sibling already carries: the POST
// must carry force_domain_override, or Coolify refuses it and nothing exists
// for a later PATCH to fix.
func TestForceDomainOverride_CreateOnSharedDomain(t *testing.T) {
	t.Parallel()
	s := &forceDomainServer{t: t}
	srv := httptest.NewServer(s.handler())
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: forceDomainConfig(srv.URL, sharedDomain+","+ownDomain, `force_domain_override = true`),
			Check: resource.TestCheckResourceAttr("coolify_application_docker_image.test", "domains",
				sharedDomain+","+ownDomain),
		}},
	})
}

// Adding the shared hostname to an application created with its own hostname
// only. force_domain_override was already true in state, so it is unchanged
// between plan and state — but the PATCH that writes domains still needs it.
func TestForceDomainOverride_UpdateAddsSharedDomain(t *testing.T) {
	t.Parallel()
	s := &forceDomainServer{t: t}
	srv := httptest.NewServer(s.handler())
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{Config: forceDomainConfig(srv.URL, ownDomain, `force_domain_override = true`)},
			{
				Config: forceDomainConfig(srv.URL, sharedDomain+","+ownDomain, `force_domain_override = true`),
				Check: resource.TestCheckResourceAttr("coolify_application_docker_image.test", "domains",
					sharedDomain+","+ownDomain),
			},
		},
	})
}

// The same change on an application whose state has no value for the flag —
// what an import leaves, since Coolify never returns it — with the flag under
// ignore_changes, the usual way to stop a stateless plan reporting null -> true
// on every run. ignore_changes makes the plan copy the prior state, so only the
// configuration still says the request must force the override.
func TestForceDomainOverride_UpdateAddsSharedDomainUnderIgnoreChanges(t *testing.T) {
	t.Parallel()
	s := &forceDomainServer{t: t}
	srv := httptest.NewServer(s.handler())
	defer srv.Close()

	ignore := `
		force_domain_override = true
		lifecycle {
			ignore_changes = [force_domain_override]
		}`

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			// No flag in state, as after an import.
			{Config: forceDomainConfig(srv.URL, ownDomain, ``)},
			{
				Config: forceDomainConfig(srv.URL, sharedDomain+","+ownDomain, ignore),
				Check: resource.TestCheckResourceAttr("coolify_application_docker_image.test", "domains",
					sharedDomain+","+ownDomain),
			},
		},
	})
}
