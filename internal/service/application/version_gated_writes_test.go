package application

import (
	"sort"
	"strings"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestConfiguredVersionGatedWriteAttrs(t *testing.T) {
	t.Parallel()

	t.Run("empty when unset", func(t *testing.T) {
		t.Parallel()
		if got := configuredVersionGatedWriteAttrs(commonAppFields{}); len(got) != 0 {
			t.Fatalf("got %v, want empty", got)
		}
	})

	t.Run("ignores null and unknown", func(t *testing.T) {
		t.Parallel()
		n := types.BoolNull()
		u := types.BoolUnknown()
		f := commonAppFields{IsGzipEnabled: &n, UseBuildSecrets: &u}
		if got := configuredVersionGatedWriteAttrs(f); len(got) != 0 {
			t.Fatalf("got %v, want empty", got)
		}
	})

	t.Run("lists set bools and ints sorted", func(t *testing.T) {
		t.Parallel()
		gz := types.BoolValue(false)
		prev := types.BoolValue(true)
		grace := types.Int64Value(30)
		f := commonAppFields{
			IsGzipEnabled:               &gz,
			IsPreviewDeploymentsEnabled: &prev,
			StopGracePeriod:             &grace,
		}
		got := configuredVersionGatedWriteAttrs(f)
		want := []string{"is_gzip_enabled", "is_preview_deployments_enabled", "stop_grace_period"}
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Fatalf("got %v, want %v", got, want)
		}
	})

	// Drift guard: every JSON key the client strips on Coolify < 4.2.0 must
	// surface in the plan warning list when configured (and no extra keys).
	t.Run("matches client ApplicationSettingsWriteJSONKeys", func(t *testing.T) {
		t.Parallel()
		tb := types.BoolValue(true)
		ti := types.Int64Value(1)
		f := commonAppFields{
			IsPreviewDeploymentsEnabled:   &tb,
			UseBuildSecrets:               &tb,
			IsGitSubmodulesEnabled:        &tb,
			IsGitLfsEnabled:               &tb,
			IsGitShallowCloneEnabled:      &tb,
			DisableBuildCache:             &tb,
			InjectBuildArgsToDockerfile:   &tb,
			IncludeSourceCommitInBuild:    &tb,
			IsEnvSortingEnabled:           &tb,
			IsPrDeploymentsPublicEnabled:  &tb,
			StopGracePeriod:               &ti,
			DockerImagesToKeep:            &ti,
			IsGzipEnabled:                 &tb,
			IsStripprefixEnabled:          &tb,
			IsRawComposeDeploymentEnabled: &tb,
		}
		got := configuredVersionGatedWriteAttrs(f)
		want := append([]string(nil), client.ApplicationSettingsWriteJSONKeys...)
		sort.Strings(want)
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Fatalf("warn attrs drifted from client strip list\ngot:  %v\nwant: %v", got, want)
		}
	})
}

func TestWarnUnsupportedApplicationSettingsWrites(t *testing.T) {
	t.Parallel()

	setGzip := types.BoolValue(false)
	fields := commonAppFields{IsGzipEnabled: &setGzip}

	t.Run("no warn on 4.2.0", func(t *testing.T) {
		t.Parallel()
		var diags diag.Diagnostics
		c := &client.Client{CoolifyVersion: "4.2.0"}
		warnUnsupportedApplicationSettingsWrites(c, fields, &diags)
		if diags.WarningsCount() != 0 {
			t.Fatalf("warnings=%d body=%v", diags.WarningsCount(), diags.Warnings())
		}
	})

	t.Run("no warn when fields unset on 4.1.2", func(t *testing.T) {
		t.Parallel()
		var diags diag.Diagnostics
		c := &client.Client{CoolifyVersion: "4.1.2"}
		warnUnsupportedApplicationSettingsWrites(c, commonAppFields{}, &diags)
		if diags.WarningsCount() != 0 {
			t.Fatalf("warnings=%d", diags.WarningsCount())
		}
	})

	t.Run("warn on 4.1.2 when field set", func(t *testing.T) {
		t.Parallel()
		var diags diag.Diagnostics
		c := &client.Client{CoolifyVersion: "4.1.2"}
		warnUnsupportedApplicationSettingsWrites(c, fields, &diags)
		if diags.WarningsCount() != 1 {
			t.Fatalf("warnings=%d, want 1: %v", diags.WarningsCount(), diags.Warnings())
		}
		w := diags.Warnings()[0]
		if w.Summary() != versionGatedWriteAttrSummary {
			t.Errorf("summary=%q", w.Summary())
		}
		detail := w.Detail()
		for _, needle := range []string{
			"4.1.2",
			"v4.2.0",
			"is_gzip_enabled",
			"state",
			"API",
			"Upgrade",
		} {
			if !strings.Contains(detail, needle) {
				t.Errorf("detail missing %q: %s", needle, detail)
			}
		}
	})

	t.Run("no warn when SupportsApplicationSettings true via empty version", func(t *testing.T) {
		t.Parallel()
		var diags diag.Diagnostics
		// Empty version assumes newest (unit tests); no warning.
		c := &client.Client{CoolifyVersion: ""}
		warnUnsupportedApplicationSettingsWrites(c, fields, &diags)
		if diags.WarningsCount() != 0 {
			t.Fatalf("warnings=%d", diags.WarningsCount())
		}
	})

	t.Run("nil client no panic", func(t *testing.T) {
		t.Parallel()
		var diags diag.Diagnostics
		warnUnsupportedApplicationSettingsWrites(nil, fields, &diags)
		if diags.WarningsCount() != 0 {
			t.Fatalf("warnings=%d", diags.WarningsCount())
		}
	})

	// Regression: after apply, state/plan carry Coolify defaults via flatten +
	// UseStateForUnknown. ModifyPlan must inspect config, so null config must
	// not warn even when plan-shaped values would look "set".
	t.Run("no warn for state defaults when config omitted on 4.1.2", func(t *testing.T) {
		t.Parallel()
		var diags diag.Diagnostics
		c := &client.Client{CoolifyVersion: "4.1.2"}
		// Values as they appear after flatten defaults (plan/state), but config
		// omits the attrs so ModifyPlan receives nulls.
		nullBool := types.BoolNull()
		nullInt := types.Int64Null()
		cfgLike := commonAppFields{
			IsGzipEnabled:               &nullBool,
			IsPreviewDeploymentsEnabled: &nullBool,
			UseBuildSecrets:             &nullBool,
			StopGracePeriod:             &nullInt,
			DockerImagesToKeep:          &nullInt,
		}
		warnUnsupportedApplicationSettingsWrites(c, cfgLike, &diags)
		if diags.WarningsCount() != 0 {
			t.Fatalf("config-omitted attrs must not warn; warnings=%d %v", diags.WarningsCount(), diags.Warnings())
		}
	})
}
