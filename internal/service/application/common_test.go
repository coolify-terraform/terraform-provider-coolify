package application

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestCanonicalGitRepo(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"strips https", "https://github.com/org/repo", "github.com/org/repo"},
		{"strips http", "http://github.com/org/repo", "github.com/org/repo"},
		{"no protocol unchanged", "github.com/org/repo", "github.com/org/repo"},
		{"ssh unchanged", "git@github.com:org/repo.git", "git@github.com:org/repo.git"},
		{"empty string", "", ""},
		{"bare slug", "org/repo", "org/repo"},
		{"trailing slash", "https://github.com/org/repo/", "github.com/org/repo/"},
		{"double protocol", "https://https://github.com/org/repo", "https://github.com/org/repo"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := canonicalGitRepo(tt.in)
			if got != tt.want {
				t.Errorf("canonicalGitRepo(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestNormalizeGitRepository(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"full https URL", "https://github.com/org/repo", "https://github.com/org/repo"},
		{"full http URL", "http://github.com/org/repo", "http://github.com/org/repo"},
		{"SSH URL", "git@github.com:org/repo.git", "git@github.com:org/repo.git"},
		{"domain-prefixed", "github.com/org/repo", "github.com/org/repo"},
		{"gitlab domain", "gitlab.com/org/repo", "gitlab.com/org/repo"},
		{"bare slug", "org/repo", "https://github.com/org/repo"},
		{"bare slug with .git", "org/repo.git", "https://github.com/org/repo.git"},
		{"bare slug nested", "org/repo/subdir", "https://github.com/org/repo/subdir"},
		{"empty string", "", ""},
		{"single word", "myrepo", "myrepo"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := normalizeGitRepository(tt.in)
			if got != tt.want {
				t.Errorf("normalizeGitRepository(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestResolveGitRepository_ProtocolNormalization(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		state    string
		apiValue string
		want     string
	}{
		{
			"user bare domain, API adds https",
			"github.com/org/repo",
			"https://github.com/org/repo",
			"github.com/org/repo", // preserve user's value
		},
		{
			"user full URL, API returns same",
			"https://github.com/org/repo",
			"https://github.com/org/repo",
			"https://github.com/org/repo",
		},
		{
			"user full URL, API strips protocol",
			"https://github.com/org/repo",
			"org/repo",
			"https://github.com/org/repo", // preserve user's value (matches normalized)
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			state := types.StringValue(tt.state)
			got := resolveGitRepository(state, tt.apiValue)
			if got.ValueString() != tt.want {
				t.Errorf("resolveGitRepository(%q, %q) = %q, want %q", tt.state, tt.apiValue, got.ValueString(), tt.want)
			}
		})
	}
}

func TestExtendedBuildDeployAttrsPreviewURLTemplateIsReadOnly(t *testing.T) {
	t.Parallel()

	attr, ok := extendedBuildDeployAttrs()["preview_url_template"]
	if !ok {
		t.Fatal("preview_url_template attribute missing")
	}

	stringAttr, ok := attr.(schema.StringAttribute)
	if !ok {
		t.Fatalf("preview_url_template has type %T, want schema.StringAttribute", attr)
	}

	if stringAttr.Optional {
		t.Fatal("preview_url_template should be read-only")
	}
	if !stringAttr.Computed {
		t.Fatal("preview_url_template should remain computed")
	}
}

func FuzzNormalizeGitRepository(f *testing.F) {
	// Seed with representative inputs from each branch.
	for _, s := range []string{
		"", "myrepo", "org/repo", "org/repo.git",
		"https://github.com/org/repo", "http://gitlab.com/a/b",
		"git@github.com:org/repo.git", "github.com/org/repo",
		"gitlab.com/org/repo", "org/repo/sub",
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, input string) {
		result := normalizeGitRepository(input)

		// Invariant 1: must never panic (implicit).

		// Invariant 2: idempotent.
		if second := normalizeGitRepository(result); second != result {
			t.Errorf("not idempotent: f(%q)=%q, f(f(%q))=%q", input, result, input, second)
		}

		// Invariant 3: if the input already had a scheme, it is preserved.
		if strings.Contains(input, "://") && result != input {
			t.Errorf("scheme-bearing input mutated: %q -> %q", input, result)
		}

		// Invariant 4: if the input starts with git@, it is preserved.
		if strings.HasPrefix(input, "git@") && result != input {
			t.Errorf("SSH input mutated: %q -> %q", input, result)
		}
	})
}

// --- hasNonDefaultAppExtendedFields tests ---

func strPtr(v string) *types.String { s := types.StringValue(v); return &s }

func int64Ptr(v int64) *types.Int64 { i := types.Int64Value(v); return &i }

func boolPtr(v bool) *types.Bool { b := types.BoolValue(v); return &b }

func TestHasNonDefaultAppExtendedFields_AllDefaults(t *testing.T) {
	t.Parallel()
	// All fields nil -> returns false (no PATCH needed).
	f := commonAppFields{}
	if hasNonDefaultAppExtendedFields(f) {
		t.Error("expected false for zero-value struct (all nils)")
	}
}

func TestHasNonDefaultAppExtendedFields_EachField(t *testing.T) {
	t.Parallel()
	// Each entry sets exactly one field to a non-default value.
	// The function must return true for every entry.
	tests := []struct {
		name  string
		setup func(*commonAppFields)
	}{
		{"LimitsMemory", func(f *commonAppFields) { f.LimitsMemory = strPtr("512M") }},
		{"LimitsMemorySwap", func(f *commonAppFields) { f.LimitsMemorySwap = strPtr("1G") }},
		{"LimitsMemoryReservation", func(f *commonAppFields) { f.LimitsMemoryReservation = strPtr("256M") }},
		{"LimitsCPUs", func(f *commonAppFields) { f.LimitsCPUs = strPtr("2") }},
		{"LimitsCPUSet", func(f *commonAppFields) { f.LimitsCPUSet = strPtr("0-3") }},
		{"LimitsMemorySwappiness", func(f *commonAppFields) { f.LimitsMemorySwappiness = int64Ptr(10) }},
		{"LimitsCPUShares", func(f *commonAppFields) { f.LimitsCPUShares = int64Ptr(512) }},
		{"HealthCheckEnabled", func(f *commonAppFields) { f.HealthCheckEnabled = boolPtr(true) }},
		{"HealthCheckPath", func(f *commonAppFields) { f.HealthCheckPath = strPtr("/health") }},
		{"HealthCheckPort", func(f *commonAppFields) { f.HealthCheckPort = strPtr("8080") }},
		{"HealthCheckInterval", func(f *commonAppFields) { f.HealthCheckInterval = int64Ptr(30) }},
		{"HealthCheckTimeout", func(f *commonAppFields) { f.HealthCheckTimeout = int64Ptr(10) }},
		{"HealthCheckRetries", func(f *commonAppFields) { f.HealthCheckRetries = int64Ptr(3) }},
		{"HealthCheckStartPeriod", func(f *commonAppFields) { f.HealthCheckStartPeriod = int64Ptr(15) }},
		{"HealthCheckCommand", func(f *commonAppFields) { f.HealthCheckCommand = strPtr("curl localhost") }},
		{"HealthCheckHost", func(f *commonAppFields) { f.HealthCheckHost = strPtr("0.0.0.0") }},
		{"HealthCheckMethod", func(f *commonAppFields) { f.HealthCheckMethod = strPtr("POST") }},
		{"HealthCheckResponseText", func(f *commonAppFields) { f.HealthCheckResponseText = strPtr("OK") }},
		{"HealthCheckReturnCode", func(f *commonAppFields) { f.HealthCheckReturnCode = int64Ptr(204) }},
		{"HealthCheckScheme", func(f *commonAppFields) { f.HealthCheckScheme = strPtr("https") }},
		{"HealthCheckType", func(f *commonAppFields) { f.HealthCheckType = strPtr("tcp") }},
		{"IsAutoDeployEnabled", func(f *commonAppFields) { f.IsAutoDeployEnabled = boolPtr(false) }},
		{"BaseDirectory", func(f *commonAppFields) { f.BaseDirectory = strPtr("/app") }},
		{"PublishDirectory", func(f *commonAppFields) { f.PublishDirectory = strPtr("/dist") }},
		{"DockerRegistryImageTag", func(f *commonAppFields) { f.DockerRegistryImageTag = strPtr("v1") }},
		{"DockerComposeDomains", func(f *commonAppFields) { f.DockerComposeDomains = strPtr("foo.com") }},
		{"GitCommitSha", func(f *commonAppFields) { f.GitCommitSha = strPtr("abc123") }},
		{"WatchPaths", func(f *commonAppFields) { f.WatchPaths = strPtr("/src") }},
		{"CustomDockerRunOptions", func(f *commonAppFields) { f.CustomDockerRunOptions = strPtr("--cap-add=SYS_PTRACE") }},
		{"CustomLabels", func(f *commonAppFields) { f.CustomLabels = strPtr("env=prod") }},
		{"CustomNetworkAliases", func(f *commonAppFields) { f.CustomNetworkAliases = strPtr("myapp") }},
		{"CustomNginxConfiguration", func(f *commonAppFields) { f.CustomNginxConfiguration = strPtr("server {}") }},
		{"PortsMappings", func(f *commonAppFields) { f.PortsMappings = strPtr("8080:80") }},
		{"IsHTTPBasicAuthEnabled", func(f *commonAppFields) { f.IsHTTPBasicAuthEnabled = boolPtr(true) }},
		{"HTTPBasicAuthUsername", func(f *commonAppFields) { f.HTTPBasicAuthUsername = strPtr("admin") }},
		{"HTTPBasicAuthPassword", func(f *commonAppFields) { f.HTTPBasicAuthPassword = strPtr("secret") }},
		{"PreDeploymentCommand", func(f *commonAppFields) { f.PreDeploymentCommand = strPtr("npm run pre") }},
		{"PreDeploymentCommandContainer", func(f *commonAppFields) { f.PreDeploymentCommandContainer = strPtr("web") }},
		{"PostDeploymentCommand", func(f *commonAppFields) { f.PostDeploymentCommand = strPtr("npm run post") }},
		{"PostDeploymentCommandContainer", func(f *commonAppFields) { f.PostDeploymentCommandContainer = strPtr("web") }},
		{"ConnectToDockerNetwork", func(f *commonAppFields) { f.ConnectToDockerNetwork = boolPtr(true) }},
		{"IsForceHTTPSEnabled", func(f *commonAppFields) { f.IsForceHTTPSEnabled = boolPtr(false) }},
		{"IsStatic", func(f *commonAppFields) { f.IsStatic = boolPtr(true) }},
		{"IsSPA", func(f *commonAppFields) { f.IsSPA = boolPtr(true) }},
		{"IsContainerLabelEscapeEnabled", func(f *commonAppFields) { f.IsContainerLabelEscapeEnabled = boolPtr(false) }},
		{"IsPreserveRepositoryEnabled", func(f *commonAppFields) { f.IsPreserveRepositoryEnabled = boolPtr(true) }},
		{"UseBuildServer", func(f *commonAppFields) { f.UseBuildServer = boolPtr(true) }},
		{"ForceDomainOverride", func(f *commonAppFields) { f.ForceDomainOverride = boolPtr(true) }},
		{"Redirect", func(f *commonAppFields) { f.Redirect = strPtr("www") }},
		{"StaticImage", func(f *commonAppFields) { f.StaticImage = strPtr("caddy:latest") }},
		{"ManualWebhookSecretGitHub", func(f *commonAppFields) { f.ManualWebhookSecretGitHub = strPtr("gh-secret") }},
		{"ManualWebhookSecretGitLab", func(f *commonAppFields) { f.ManualWebhookSecretGitLab = strPtr("gl-secret") }},
		{"ManualWebhookSecretBitbucket", func(f *commonAppFields) { f.ManualWebhookSecretBitbucket = strPtr("bb-secret") }},
		{"ManualWebhookSecretGitea", func(f *commonAppFields) { f.ManualWebhookSecretGitea = strPtr("gitea-secret") }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := commonAppFields{}
			tt.setup(&f)
			if !hasNonDefaultAppExtendedFields(f) {
				t.Errorf("expected true when %s is non-default", tt.name)
			}
		})
	}
}

func TestHasNonDefaultAppExtendedFields_DefaultValues(t *testing.T) {
	t.Parallel()
	// Setting a field to its default value should NOT trigger a PATCH.
	tests := []struct {
		name  string
		setup func(*commonAppFields)
	}{
		{"LimitsMemory=0", func(f *commonAppFields) { f.LimitsMemory = strPtr("0") }},
		{"LimitsCPUs=0", func(f *commonAppFields) { f.LimitsCPUs = strPtr("0") }},
		{"LimitsMemorySwappiness=60", func(f *commonAppFields) { f.LimitsMemorySwappiness = int64Ptr(60) }},
		{"LimitsCPUShares=1024", func(f *commonAppFields) { f.LimitsCPUShares = int64Ptr(1024) }},
		{"HealthCheckEnabled=false", func(f *commonAppFields) { f.HealthCheckEnabled = boolPtr(false) }},
		{"HealthCheckPath=/", func(f *commonAppFields) { f.HealthCheckPath = strPtr("/") }},
		{"HealthCheckInterval=5", func(f *commonAppFields) { f.HealthCheckInterval = int64Ptr(5) }},
		{"IsAutoDeployEnabled=true", func(f *commonAppFields) { f.IsAutoDeployEnabled = boolPtr(true) }},
		{"IsForceHTTPSEnabled=true", func(f *commonAppFields) { f.IsForceHTTPSEnabled = boolPtr(true) }},
		{"Redirect=both", func(f *commonAppFields) { f.Redirect = strPtr(defaultRedirect) }},
		{"StaticImage=nginx:alpine", func(f *commonAppFields) { f.StaticImage = strPtr(defaultStaticImage) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := commonAppFields{}
			tt.setup(&f)
			if hasNonDefaultAppExtendedFields(f) {
				t.Errorf("expected false when %s is set to its default", tt.name)
			}
		})
	}
}

func TestDeleteApplication_AddsWarningWhenPollingTimesOut(t *testing.T) {
	t.Parallel()

	const uuid = "app-delete-timeout-uuid"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodDelete && r.URL.Path == fmt.Sprintf("/api/v1/applications/%s", uuid):
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodGet && r.URL.Path == fmt.Sprintf("/api/v1/applications/%s", uuid):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"uuid":"` + uuid + `"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/version":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"version":"test"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := client.New(srv.URL, "test-token")
	ctx, cancel := context.WithTimeout(context.Background(), 800*time.Millisecond)
	defer cancel()
	resp := &resource.DeleteResponse{}

	deleteApplication(ctx, c, "coolify_application", uuid, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics.Errors())
	}
	if resp.Diagnostics.WarningsCount() != 1 {
		t.Fatalf("expected 1 warning, got %d", resp.Diagnostics.WarningsCount())
	}
	warning := resp.Diagnostics.Warnings()[0]
	if warning.Summary() != deletePollingTimeoutWarningSummary {
		t.Fatalf("warning summary = %q, want %q", warning.Summary(), deletePollingTimeoutWarningSummary)
	}
	if !strings.Contains(warning.Detail(), uuid) {
		t.Fatalf("warning detail %q does not mention uuid %s", warning.Detail(), uuid)
	}
	if !strings.Contains(warning.Detail(), "may still exist temporarily") {
		t.Fatalf("warning detail %q does not explain the temporary remote state", warning.Detail())
	}
}
