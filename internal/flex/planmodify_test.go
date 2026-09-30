package flex

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func resourceRaw() tftypes.Value {
	typ := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"id": tftypes.String,
	}}
	return tftypes.NewValue(typ, map[string]tftypes.Value{
		"id": tftypes.NewValue(tftypes.String, "res"),
	})
}

func TestRequiresReplaceIfKnown(t *testing.T) {
	t.Parallel()
	mod := RequiresReplaceIfKnown()
	raw := resourceRaw()
	state := tfsdk.State{Raw: raw}
	plan := tfsdk.Plan{Raw: raw}

	cases := []struct {
		name         string
		state, plan  types.String
		wantReplace  bool
		nullResource bool
	}{
		{name: "null state adopts", state: types.StringNull(), plan: types.StringValue("aaa"), wantReplace: false},
		{name: "unknown state adopts", state: types.StringUnknown(), plan: types.StringValue("aaa"), wantReplace: false},
		{name: "known change replaces", state: types.StringValue("aaa"), plan: types.StringValue("bbb"), wantReplace: true},
		{name: "equal known does not replace", state: types.StringValue("aaa"), plan: types.StringValue("aaa"), wantReplace: false},
		{name: "create skips", state: types.StringNull(), plan: types.StringValue("aaa"), wantReplace: false, nullResource: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := planmodifier.StringRequest{
				StateValue: tc.state,
				PlanValue:  tc.plan,
				State:      state,
				Plan:       plan,
			}
			if tc.nullResource {
				req.State.Raw = tftypes.Value{}
			}
			resp := &planmodifier.StringResponse{}
			mod.PlanModifyString(context.Background(), req, resp)
			if resp.RequiresReplace != tc.wantReplace {
				t.Fatalf("RequiresReplace = %v, want %v", resp.RequiresReplace, tc.wantReplace)
			}
		})
	}
}
