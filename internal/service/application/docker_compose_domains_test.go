package application

import (
	"errors"
	"strings"
	"testing"
)

// TestDockerComposeDomainsDescription_DocumentsOrderingAndVersions locks the
// operator-facing docs for issue #653. Coolify rejects docker_compose_domains
// until docker_compose_raw is set; that guard is present on every Coolify
// version this provider supports (minCoolifyVersion 4.1.0 through 4.2.0+).
func TestDockerComposeDomainsDescription_DocumentsOrderingAndVersions(t *testing.T) {
	t.Parallel()

	required := []string{
		"dockercompose",
		"jsonencode",
		"name",
		"domain",
		"docker_compose_raw",
		"deployment",
		"two-stage",
		"instant_deploy",
		"v4.1.0",
		"coolify_service",
		"load-compose",
	}
	for _, needle := range required {
		if !strings.Contains(dockerComposeDomainsDescription, needle) {
			t.Errorf("dockerComposeDomainsDescription missing %q", needle)
		}
	}

	// Must not claim a version floor higher than minCoolifyVersion for this
	// constraint: the empty-raw guard exists on 4.1.0, 4.1.1, 4.1.2, and 4.2.0.
	if strings.Contains(dockerComposeDomainsDescription, "Requires Coolify >= v4.2") {
		t.Error("ordering constraint applies on all supported versions; do not mark it v4.2.0-only")
	}
}

func TestAnnotateDockerComposeDomainsError(t *testing.T) {
	t.Parallel()

	t.Run("nil", func(t *testing.T) {
		t.Parallel()
		if got := annotateDockerComposeDomainsError(nil); got != "" {
			t.Errorf("got %q, want empty", got)
		}
	})

	t.Run("unrelated", func(t *testing.T) {
		t.Parallel()
		err := errors.New(`unexpected status 422: {"message":"Validation failed.","errors":{"name":"required"}}`)
		if got := annotateDockerComposeDomainsError(err); got != "" {
			t.Errorf("got %q, want empty", got)
		}
	})

	t.Run("coolify empty raw", func(t *testing.T) {
		t.Parallel()
		err := errors.New(`unexpected status 422: {"message":"Validation failed.","errors":{"docker_compose_domains":"` +
			coolifyDockerComposeDomainsNeedRaw + `. Reload the compose file from the git repository first."}}`)
		got := annotateDockerComposeDomainsError(err)
		if got == "" {
			t.Fatal("expected annotation, got empty")
		}
		for _, needle := range []string{
			"docker_compose_raw",
			"docker_compose_domains",
			"v4.1.0",
			"second apply",
			"deploy once",
		} {
			if !strings.Contains(got, needle) {
				t.Errorf("annotation missing %q: %s", needle, got)
			}
		}
	})
}

// TestCoolifyDockerComposeDomainsNeedRawMatchesUpstream locks the substring we
// match against Coolify controller error text so a typo does not silently drop
// the operator guidance.
func TestCoolifyDockerComposeDomainsNeedRawMatchesUpstream(t *testing.T) {
	t.Parallel()
	// Exact text from ApplicationsController on Coolify v4.1.0–v4.2.0.
	const upstream = "Cannot set docker_compose_domains without docker_compose_raw"
	if coolifyDockerComposeDomainsNeedRaw != upstream {
		t.Errorf("coolifyDockerComposeDomainsNeedRaw = %q, want %q", coolifyDockerComposeDomainsNeedRaw, upstream)
	}
}
