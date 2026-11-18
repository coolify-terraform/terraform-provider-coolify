package acctest

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccCheckDestroy_Retries429ThenNotFound(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("Authorization = %q", r.Header.Get("Authorization"))
		}
		if hits.Add(1) == 1 {
			w.Header().Set("Retry-After", "30")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("COOLIFY_ENDPOINT", srv.URL)
	t.Setenv("COOLIFY_TOKEN", "test-token")
	restore := stubDestroyRateLimitWait(t)
	defer restore()

	err := AccCheckDestroy("coolify_database_redis", "/api/v1/databases/")(destroyState("coolify_database_redis", "db-1"))
	if err != nil {
		t.Fatal(err)
	}
	if hits.Load() != 2 {
		t.Fatalf("GET count = %d, want 2", hits.Load())
	}
}

func TestAccCheckDestroy_RateLimitDoesNotClaimResourceRemains(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("COOLIFY_ENDPOINT", srv.URL)
	t.Setenv("COOLIFY_TOKEN", "test-token")
	restore := stubDestroyRateLimitWait(t)
	defer restore()

	err := AccCheckDestroy("coolify_database_redis", "/api/v1/databases/")(destroyState("coolify_database_redis", "db-1"))
	if err == nil {
		t.Fatal("expected rate limit error")
	}
	if got := err.Error(); !strings.Contains(got, "status 429") || !strings.Contains(got, "rate limited") || strings.Contains(got, "still exists") {
		t.Fatalf("error = %q", got)
	}
}

func TestAccCheckDestroy_OtherStatusStillExists(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("COOLIFY_ENDPOINT", srv.URL)
	t.Setenv("COOLIFY_TOKEN", "test-token")

	err := AccCheckDestroy("coolify_database_redis", "/api/v1/databases/")(destroyState("coolify_database_redis", "db-1"))
	if err == nil || !strings.Contains(err.Error(), "still exists") || !strings.Contains(err.Error(), "status 200") {
		t.Fatalf("error = %v", err)
	}
}

func TestDestroy429Wait_CapsRetryAfter(t *testing.T) {
	if got := destroy429Wait(0, 30*time.Second); got != accDestroy429MaxWait {
		t.Fatalf("wait = %s, want cap %s", got, accDestroy429MaxWait)
	}
	if got := destroy429Wait(0, 0); got != 200*time.Millisecond {
		t.Fatalf("default wait = %s", got)
	}
}

func destroyState(resourceType, uuid string) *terraform.State {
	return &terraform.State{
		Modules: []*terraform.ModuleState{{
			Path: []string{"root"},
			Resources: map[string]*terraform.ResourceState{
				resourceType + ".test": {
					Type: resourceType,
					Primary: &terraform.InstanceState{
						ID:         uuid,
						Attributes: map[string]string{"uuid": uuid},
					},
				},
			},
		}},
	}
}

func stubDestroyRateLimitWait(t *testing.T) func() {
	t.Helper()
	prev := waitAfterRateLimit
	waitAfterRateLimit = func(time.Duration) {}
	return func() { waitAfterRateLimit = prev }
}
