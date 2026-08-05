package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// fullSettingsInput sets every ApplicationSetting field plus one ordinary field,
// so a stripped payload is still a valid, non-empty request.
func fullSettingsInput() UpdateApplicationInput {
	b := true
	n := int64(5)
	name := "gate-probe"
	return UpdateApplicationInput{
		Name:                          &name,
		IsPreviewDeploymentsEnabled:   &b,
		UseBuildSecrets:               &b,
		IsGitSubmodulesEnabled:        &b,
		IsGitLfsEnabled:               &b,
		IsGitShallowCloneEnabled:      &b,
		DisableBuildCache:             &b,
		InjectBuildArgsToDockerfile:   &b,
		IncludeSourceCommitInBuild:    &b,
		IsEnvSortingEnabled:           &b,
		IsPrDeploymentsPublicEnabled:  &b,
		DockerImagesToKeep:            &n,
		IsGzipEnabled:                 &b,
		IsStripprefixEnabled:          &b,
		IsRawComposeDeploymentEnabled: &b,
		StopGracePeriod:               &n,
	}
}

// capturingServer records the body of the next PATCH.
func capturingServer(t *testing.T, body *map[string]json.RawMessage) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("PATCH /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(body); err != nil {
			t.Errorf("decoding PATCH body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]string{"uuid": "app-1"}); err != nil {
			t.Errorf("encoding response: %v", err)
		}
	})
	return httptest.NewServer(mux)
}

// TestUpdateApplication_VersionGate is the regression guard for the Create-time
// 422 on Coolify 4.1.x. Those releases have no APPLICATION_SETTING_FIELDS
// constant, so every settings field falls outside the endpoint's allow list and
// is rejected — and one rejected field aborts the whole request, which fails
// Create and leaves the resource tainted.
func TestUpdateApplication_VersionGate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		version      string
		wantSettings bool
	}{
		{"4.1.2 withholds settings", "4.1.2", false},
		{"4.1.0 withholds settings", "4.1.0", false},
		{"4.2.0 sends settings", "4.2.0", true},
		{"4.3.0 sends settings", "4.3.0", true},
		{"v-prefixed 4.2.0 sends settings", "v4.2.0", true},
		{"unknown version assumes newest", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var body map[string]json.RawMessage
			srv := capturingServer(t, &body)
			defer srv.Close()

			c := New(srv.URL, "token")
			c.CoolifyVersion = tt.version

			if _, err := c.UpdateApplication(context.Background(), "app-1", fullSettingsInput()); err != nil {
				t.Fatalf("UpdateApplication: %v", err)
			}

			if _, ok := body["name"]; !ok {
				t.Error("the gate dropped a non-settings field; only settings may be withheld")
			}
			for _, key := range ApplicationSettingsWriteJSONKeys {
				_, sent := body[key]
				if sent != tt.wantSettings {
					t.Errorf("Coolify %q: %s sent=%v, want %v", tt.version, key, sent, tt.wantSettings)
				}
			}
		})
	}
}

func TestSupportsApplicationSettings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		version string
		want    bool
	}{
		{"4.1.0", false},
		{"4.1.2", false},
		{"4.2.0", true},
		{"4.2.1", true},
		{"5.0.0", true},
		{"", true},              // unknown: assume newest
		{"not-a-version", true}, // unparseable: assume newest, matching IsVersionAtLeast
	}
	for _, tt := range tests {
		c := &Client{CoolifyVersion: tt.version}
		if got := c.SupportsApplicationSettings(); got != tt.want {
			t.Errorf("SupportsApplicationSettings(%q) = %v, want %v", tt.version, got, tt.want)
		}
	}
	var nilClient *Client
	if !nilClient.SupportsApplicationSettings() {
		t.Error("nil client should assume newest behaviour rather than panic")
	}
}

func TestHasOnlyApplicationSettings(t *testing.T) {
	t.Parallel()

	b := true
	name := "x"

	if (UpdateApplicationInput{}).HasOnlyApplicationSettings() {
		t.Error("an empty input has nothing to strip; want false")
	}
	if !(UpdateApplicationInput{IsGzipEnabled: &b}).HasOnlyApplicationSettings() {
		t.Error("settings-only input should report true so callers can skip the request")
	}
	if !(UpdateApplicationInput{IsPreviewDeploymentsEnabled: &b}).HasOnlyApplicationSettings() {
		t.Error("preview-only input is version-gated and should report true so callers can skip the request")
	}
	if !(UpdateApplicationInput{UseBuildSecrets: &b}).HasOnlyApplicationSettings() {
		t.Error("build-secrets-only input is version-gated and should report true so callers can skip the request")
	}
	if (UpdateApplicationInput{Name: &name, IsGzipEnabled: &b}).HasOnlyApplicationSettings() {
		t.Error("input with a non-settings field still has work to do; want false")
	}
}
