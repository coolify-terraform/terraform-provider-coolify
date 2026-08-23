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
	gpuDriver := "nvidia"
	gpuCount := "1"
	noindex := []string{"https://staging.example.com"}
	return UpdateApplicationInput{
		Name:                             &name,
		IsPreviewDeploymentsEnabled:      &b,
		UseBuildSecrets:                  &b,
		IsGitSubmodulesEnabled:           &b,
		IsGitLfsEnabled:                  &b,
		IsGitShallowCloneEnabled:         &b,
		DisableBuildCache:                &b,
		InjectBuildArgsToDockerfile:      &b,
		IncludeSourceCommitInBuild:       &b,
		IsEnvSortingEnabled:              &b,
		IsPrDeploymentsPublicEnabled:     &b,
		DockerImagesToKeep:               &n,
		IsGzipEnabled:                    &b,
		IsStripprefixEnabled:             &b,
		IsRawComposeDeploymentEnabled:    &b,
		StopGracePeriod:                  &n,
		IsLogDrainEnabled:                &b,
		IsGpuEnabled:                     &b,
		GpuDriver:                        &gpuDriver,
		GpuCount:                         &gpuCount,
		GpuDeviceIds:                     &gpuCount,
		GpuOptions:                       &gpuDriver,
		IsConsistentContainerNameEnabled: &b,
		CustomInternalName:               &name,
		NoindexDomains:                   &noindex,
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
		name            string
		version         string
		wantSettingsV42 bool
		wantSettingsV43 bool
	}{
		{"4.1.2 withholds all settings", "4.1.2", false, false},
		{"4.1.0 withholds all settings", "4.1.0", false, false},
		{"4.2.0 sends v4.2 settings only", "4.2.0", true, false},
		{"4.2.1 sends v4.2 settings only", "4.2.1", true, false},
		{"4.3.0 sends all settings", "4.3.0", true, true},
		{"v-prefixed 4.2.0 sends v4.2 only", "v4.2.0", true, false},
		{"v-prefixed 4.3.0 sends all", "v4.3.0", true, true},
		{"unknown version assumes newest", "", true, true},
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
				if sent != tt.wantSettingsV42 {
					t.Errorf("Coolify %q: %s sent=%v, want %v", tt.version, key, sent, tt.wantSettingsV42)
				}
			}
			for _, key := range ApplicationSettingsV43WriteJSONKeys {
				_, sent := body[key]
				if sent != tt.wantSettingsV43 {
					t.Errorf("Coolify %q: %s sent=%v, want %v", tt.version, key, sent, tt.wantSettingsV43)
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
		{"4.3.0", true},
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

func TestSupportsApplicationSettingsV43(t *testing.T) {
	t.Parallel()

	tests := []struct {
		version string
		want    bool
	}{
		{"4.1.0", false},
		{"4.2.0", false},
		{"4.2.9", false},
		{"4.3.0", true},
		{"4.3.1", true},
		{"5.0.0", true},
		{"", true},
		{"not-a-version", true},
	}
	for _, tt := range tests {
		c := &Client{CoolifyVersion: tt.version}
		if got := c.SupportsApplicationSettingsV43(); got != tt.want {
			t.Errorf("SupportsApplicationSettingsV43(%q) = %v, want %v", tt.version, got, tt.want)
		}
	}
	var nilClient *Client
	if !nilClient.SupportsApplicationSettingsV43() {
		t.Error("nil client should assume newest behaviour rather than panic")
	}
}

func TestSupportsSMTPEhloDomain(t *testing.T) {
	t.Parallel()

	tests := []struct {
		version string
		want    bool
	}{
		{"4.3.9", false},
		{"v4.3.9", false},
		{"4.3.0", true}, // CI edge version lie
		{"v4.3.0-edge", true},
		{"4.3.10", true},
		{"v4.3.10", true},
		{"4.4.0", true},
		{"4.4-rc.1", true}, // unparseable minor; IsVersionAtLeast fail-opens
		{"", true},
		{"not-a-version", true},
	}
	for _, tt := range tests {
		c := &Client{CoolifyVersion: tt.version}
		if got := c.SupportsSMTPEhloDomain(); got != tt.want {
			t.Errorf("SupportsSMTPEhloDomain(%q) = %v, want %v", tt.version, got, tt.want)
		}
	}
	var nilClient *Client
	if !nilClient.SupportsSMTPEhloDomain() {
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
	if !(UpdateApplicationInput{IsLogDrainEnabled: &b}).HasOnlyApplicationSettings() {
		t.Error("v4.3 settings-only input should report true so callers can skip the request")
	}
	if (UpdateApplicationInput{Name: &name, IsGzipEnabled: &b}).HasOnlyApplicationSettings() {
		t.Error("input with a non-settings field still has work to do; want false")
	}
}

func TestHasOnlyApplicationSettingsV43(t *testing.T) {
	t.Parallel()

	b := true
	name := "x"
	noindex := []string{"https://a.example.com"}

	if (UpdateApplicationInput{}).HasOnlyApplicationSettingsV43() {
		t.Error("empty input: want false")
	}
	if !(UpdateApplicationInput{IsLogDrainEnabled: &b}).HasOnlyApplicationSettingsV43() {
		t.Error("v4.3-only input should report true")
	}
	if !(UpdateApplicationInput{NoindexDomains: &noindex}).HasOnlyApplicationSettingsV43() {
		t.Error("noindex-only input should report true")
	}
	if (UpdateApplicationInput{IsGzipEnabled: &b}).HasOnlyApplicationSettingsV43() {
		t.Error("v4.2-only field must not count as v4.3-only")
	}
	if (UpdateApplicationInput{Name: &name, IsLogDrainEnabled: &b}).HasOnlyApplicationSettingsV43() {
		t.Error("input with a non-settings field still has work to do; want false")
	}
}

func TestVersionGatedWriteKeysPresent(t *testing.T) {
	t.Parallel()

	if got := (UpdateApplicationInput{}).versionGatedWriteKeysPresent(); len(got) != 0 {
		t.Fatalf("empty input: got %v", got)
	}
	got := fullSettingsInput().versionGatedWriteKeysPresent()
	want := len(ApplicationSettingsWriteJSONKeys) + len(ApplicationSettingsV43WriteJSONKeys)
	if len(got) != want {
		t.Fatalf("got %d keys %v, want %d", len(got), got, want)
	}
	gotV43 := fullSettingsInput().versionGatedV43WriteKeysPresent()
	if len(gotV43) != len(ApplicationSettingsV43WriteJSONKeys) {
		t.Fatalf("v43: got %d keys %v, want %d", len(gotV43), gotV43, len(ApplicationSettingsV43WriteJSONKeys))
	}
}

// TestFullSettingsInput_CoversExportedKeys fails if ApplicationSettingsWriteJSONKeys
// or ApplicationSettingsV43WriteJSONKeys grow without fullSettingsInput (and thus
// clearApplicationSettings) following.
func TestFullSettingsInput_CoversExportedKeys(t *testing.T) {
	t.Parallel()

	raw, err := json.Marshal(fullSettingsInput())
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var body map[string]json.RawMessage
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	allKeys := append(append([]string{}, ApplicationSettingsWriteJSONKeys...), ApplicationSettingsV43WriteJSONKeys...)
	for _, key := range allKeys {
		if _, ok := body[key]; !ok {
			t.Errorf("fullSettingsInput omits %q; update the fixture and clearApplicationSettings", key)
		}
	}

	cleared := fullSettingsInput()
	cleared.clearApplicationSettings()
	clearedRaw, err := json.Marshal(cleared)
	if err != nil {
		t.Fatalf("marshal cleared: %v", err)
	}
	var clearedBody map[string]json.RawMessage
	if err := json.Unmarshal(clearedRaw, &clearedBody); err != nil {
		t.Fatalf("unmarshal cleared: %v", err)
	}
	for _, key := range allKeys {
		if _, ok := clearedBody[key]; ok {
			t.Errorf("clearApplicationSettings left %q in the payload", key)
		}
	}
	if _, ok := clearedBody["name"]; !ok {
		t.Error("clearApplicationSettings must not drop non-settings fields")
	}

	// v4.3-only clear must leave v4.2 fields and strip v4.3 fields.
	v43Only := fullSettingsInput()
	v43Only.clearApplicationSettingsV43()
	v43Raw, err := json.Marshal(v43Only)
	if err != nil {
		t.Fatalf("marshal v43 clear: %v", err)
	}
	var v43Body map[string]json.RawMessage
	if err := json.Unmarshal(v43Raw, &v43Body); err != nil {
		t.Fatalf("unmarshal v43 clear: %v", err)
	}
	for _, key := range ApplicationSettingsV43WriteJSONKeys {
		if _, ok := v43Body[key]; ok {
			t.Errorf("clearApplicationSettingsV43 left %q", key)
		}
	}
	for _, key := range ApplicationSettingsWriteJSONKeys {
		if _, ok := v43Body[key]; !ok {
			t.Errorf("clearApplicationSettingsV43 dropped v4.2 key %q", key)
		}
	}
}
