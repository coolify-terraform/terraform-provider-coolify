package application

import (
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestRejectZeroPortsExposesUnknownPlan(t *testing.T) {
	t.Parallel()

	unknown := types.StringUnknown()
	stored := types.StringValue("80")
	// Create on a release that rejects a real zero-port app, and an update
	// on releases that reject a clear. An unknown port is neither.
	cases := []struct {
		version  string
		isCreate bool
	}{
		{version: "v4.1.1", isCreate: true},
		{version: "v4.2.0-test", isCreate: false},
		{version: "4.3.14", isCreate: false},
	}
	for _, tc := range cases {
		t.Run(tc.version, func(t *testing.T) {
			t.Parallel()
			c := &client.Client{CoolifyVersion: tc.version}
			var diags diag.Diagnostics
			rejectZeroPortsExposes(c, &unknown, &stored, tc.isCreate, &diags)
			if diags.HasError() {
				t.Fatalf("unknown ports_exposes on %s: %s", tc.version, diags)
			}
		})
	}
}

func TestRejectZeroPortsExposesKnownZeroStillGated(t *testing.T) {
	t.Parallel()

	none := types.StringNull()
	stored := types.StringValue("80")

	t.Run("create below v4.1.2", func(t *testing.T) {
		t.Parallel()
		c := &client.Client{CoolifyVersion: "v4.1.1"}
		var diags diag.Diagnostics
		rejectZeroPortsExposes(c, &none, &stored, true, &diags)
		if !diags.HasError() {
			t.Fatal("create with no ports on v4.1.1 should fail the plan")
		}
	})

	t.Run("clear below v4.3.15", func(t *testing.T) {
		t.Parallel()
		for _, ver := range []string{"4.3.0", "v4.3.0-edge", "4.3.14"} {
			c := &client.Client{CoolifyVersion: ver}
			var diags diag.Diagnostics
			rejectZeroPortsExposes(c, &none, &stored, false, &diags)
			if !diags.HasError() {
				t.Fatalf("clear on %s should fail the plan", ver)
			}
		}
	})

	t.Run("clear on v4.3.15", func(t *testing.T) {
		t.Parallel()
		c := &client.Client{CoolifyVersion: "4.3.15"}
		var diags diag.Diagnostics
		rejectZeroPortsExposes(c, &none, &stored, false, &diags)
		if diags.HasError() {
			t.Fatalf("clear on v4.3.15: %s", diags)
		}
	})
}
