package flex

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestEnvironmentNamePlan(t *testing.T) {
	t.Parallel()
	mod := EnvironmentNamePlan()
	attrSchema := schema.Schema{Attributes: map[string]schema.Attribute{
		"environment_name": schema.StringAttribute{Optional: true, Computed: true},
		"environment_uuid": schema.StringAttribute{Optional: true},
	}}
	objType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"environment_name": tftypes.String,
		"environment_uuid": tftypes.String,
	}}

	newConfig := func(name, uuid tftypes.Value) tfsdk.Config {
		return tfsdk.Config{
			Schema: attrSchema,
			Raw: tftypes.NewValue(objType, map[string]tftypes.Value{
				"environment_name": name,
				"environment_uuid": uuid,
			}),
		}
	}
	nullStr := tftypes.NewValue(tftypes.String, nil)
	unknownStr := tftypes.NewValue(tftypes.String, tftypes.UnknownValue)
	str := func(s string) tftypes.Value { return tftypes.NewValue(tftypes.String, s) }

	stored := tftypes.NewValue(objType, map[string]tftypes.Value{
		"environment_name": str("production"),
		"environment_uuid": nullStr,
	})
	state := tfsdk.State{Schema: attrSchema, Raw: stored}
	plan := tfsdk.Plan{Schema: attrSchema, Raw: stored}

	cases := []struct {
		name            string
		cfgName         tftypes.Value
		cfgUUID         tftypes.Value
		state           tfsdk.State
		stateValue      types.String
		want            types.String
		wantPlanNull    bool
		wantPlanUnknown bool
		nullPlan        bool
		wantReplace     bool
		wantErr         bool
	}{
		{
			name:       "default production on create",
			cfgName:    nullStr,
			cfgUUID:    nullStr,
			state:      tfsdk.State{Raw: tftypes.Value{}},
			stateValue: types.StringNull(),
			want:       types.StringValue("production"),
		},
		{
			name:         "uuid only clears name",
			cfgName:      nullStr,
			cfgUUID:      str("cccc0001-0001-4000-8000-000000000001"),
			state:        tfsdk.State{Raw: tftypes.Value{}},
			stateValue:   types.StringNull(),
			wantPlanNull: true,
		},
		{
			name:         "uuid replaces a stored name",
			cfgName:      nullStr,
			cfgUUID:      str("cccc0001-0001-4000-8000-000000000001"),
			state:        state,
			stateValue:   types.StringValue("production"),
			wantPlanNull: true,
			wantReplace:  true,
		},
		{
			name:       "omitted name keeps state",
			cfgName:    nullStr,
			cfgUUID:    nullStr,
			state:      state,
			stateValue: types.StringValue("production"),
			want:       types.StringValue("production"),
		},
		{
			name:        "known name change replaces",
			cfgName:     str("staging"),
			cfgUUID:     nullStr,
			state:       state,
			stateValue:  types.StringValue("production"),
			want:        types.StringValue("staging"),
			wantReplace: true,
		},
		{
			name:    "both set is an error",
			cfgName: str("production"),
			cfgUUID: str("cccc0001-0001-4000-8000-000000000001"),
			state:   state,
			wantErr: true,
		},
		{
			name:         "unknown uuid does not default the name",
			cfgName:      nullStr,
			cfgUUID:      unknownStr,
			state:        tfsdk.State{Raw: tftypes.Value{}},
			stateValue:   types.StringNull(),
			wantPlanNull: true,
		},
		{
			name:    "unknown uuid with a name is an error",
			cfgName: str("production"),
			cfgUUID: unknownStr,
			state:   tfsdk.State{Raw: tftypes.Value{}},
			wantErr: true,
		},
		{
			name:            "unknown name on create stays unknown",
			cfgName:         unknownStr,
			cfgUUID:         nullStr,
			state:           tfsdk.State{Raw: tftypes.Value{}},
			stateValue:      types.StringNull(),
			wantPlanUnknown: true,
		},
		{
			name:            "unknown name replaces a stored name",
			cfgName:         unknownStr,
			cfgUUID:         nullStr,
			state:           state,
			stateValue:      types.StringValue("production"),
			wantPlanUnknown: true,
			wantReplace:     true,
		},
		{
			name:         "null plan does not fill the name",
			cfgName:      nullStr,
			cfgUUID:      nullStr,
			state:        state,
			stateValue:   types.StringValue("production"),
			nullPlan:     true,
			wantPlanNull: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := planmodifier.StringRequest{
				Config:     newConfig(tc.cfgName, tc.cfgUUID),
				State:      tc.state,
				Plan:       plan,
				StateValue: tc.stateValue,
				PlanValue:  tc.stateValue,
			}
			switch {
			case tc.nullPlan:
				req.Plan = tfsdk.Plan{Schema: attrSchema, Raw: tftypes.NewValue(objType, nil)}
				req.PlanValue = types.StringNull()
			case !tc.cfgName.IsKnown():
				req.PlanValue = types.StringUnknown()
			case !tc.cfgName.IsNull():
				var s string
				_ = tc.cfgName.As(&s)
				req.PlanValue = types.StringValue(s)
			}
			// The framework seeds PlanValue from the proposed plan. A modifier
			// that leaves it alone must not null a configured name.
			resp := &planmodifier.StringResponse{PlanValue: req.PlanValue}
			mod.PlanModifyString(context.Background(), req, resp)
			if tc.wantErr {
				if !resp.Diagnostics.HasError() {
					t.Fatal("expected error when both name and uuid are set")
				}
				return
			}
			if resp.Diagnostics.HasError() {
				t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
			}
			switch {
			case tc.wantPlanNull:
				if !resp.PlanValue.IsNull() {
					t.Fatalf("plan = %s, want null", resp.PlanValue)
				}
			case tc.wantPlanUnknown:
				if !resp.PlanValue.IsUnknown() {
					t.Fatalf("plan = %s, want unknown", resp.PlanValue)
				}
			default:
				if !resp.PlanValue.Equal(tc.want) {
					t.Fatalf("plan = %s, want %s", resp.PlanValue, tc.want)
				}
			}
			if resp.RequiresReplace != tc.wantReplace {
				t.Fatalf("RequiresReplace = %v, want %v", resp.RequiresReplace, tc.wantReplace)
			}
		})
	}
}
