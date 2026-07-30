package service

import (
	"context"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// serviceValidateConfig runs ValidateConfig with the given type and
// docker_compose_raw values (other attributes left null).
func serviceValidateConfig(t *testing.T, typeVal, composeVal types.String) resource.ValidateConfigResponse {
	t.Helper()
	ctx := context.Background()
	r := &serviceResource{}

	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema: %v", schemaResp.Diagnostics.Errors())
	}

	state := tfsdk.State{
		Schema: schemaResp.Schema,
		Raw:    tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil),
	}
	// Required-looking ids so Config.Get is realistic; validation only
	// inspects type and docker_compose_raw.
	for _, set := range []struct {
		p path.Path
		v types.String
	}{
		{path.Root("project_uuid"), types.StringValue("aaaa0001-0001-4000-8000-000000000001")},
		{path.Root("server_uuid"), types.StringValue("bbbb0001-0001-4000-8000-000000000001")},
		{path.Root("type"), typeVal},
		{path.Root("docker_compose_raw"), composeVal},
	} {
		diags := state.SetAttribute(ctx, set.p, set.v)
		if diags.HasError() {
			t.Fatalf("set %s: %v", set.p, diags.Errors())
		}
	}

	resp := resource.ValidateConfigResponse{}
	r.ValidateConfig(ctx, resource.ValidateConfigRequest{
		Config: tfsdk.Config(state),
	}, &resp)
	return resp
}

func TestServiceResource_ValidateConfig_UnknownComposeOrType(t *testing.T) {
	t.Parallel()

	compose := types.StringValue("version: '3'\nservices:\n  web:\n    image: nginx\n")
	catalog := types.StringValue("plausible")

	cases := []struct {
		name    string
		typ     types.String
		compose types.String
		wantErr *regexp.Regexp
	}{
		{
			name:    "unknown compose only (var/data source pattern)",
			typ:     types.StringNull(),
			compose: types.StringUnknown(),
			wantErr: nil,
		},
		{
			name:    "unknown type only",
			typ:     types.StringUnknown(),
			compose: types.StringNull(),
			wantErr: nil,
		},
		{
			name:    "both unknown",
			typ:     types.StringUnknown(),
			compose: types.StringUnknown(),
			wantErr: nil,
		},
		{
			name:    "known compose only",
			typ:     types.StringNull(),
			compose: compose,
			wantErr: nil,
		},
		{
			name:    "known type only",
			typ:     catalog,
			compose: types.StringNull(),
			wantErr: nil,
		},
		{
			name:    "neither set",
			typ:     types.StringNull(),
			compose: types.StringNull(),
			wantErr: regexp.MustCompile(`(?i)must be set|missing required`),
		},
		{
			name:    "both known set",
			typ:     catalog,
			compose: compose,
			wantErr: regexp.MustCompile(`(?i)mutually exclusive|conflicting`),
		},
		// Conflict is deferred while either side is still unknown.
		{
			name:    "known type and unknown compose",
			typ:     catalog,
			compose: types.StringUnknown(),
			wantErr: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			resp := serviceValidateConfig(t, tc.typ, tc.compose)
			if tc.wantErr == nil {
				if resp.Diagnostics.HasError() {
					t.Fatalf("unexpected errors: %v", resp.Diagnostics.Errors())
				}
				return
			}
			if !resp.Diagnostics.HasError() {
				t.Fatal("expected validation error, got none")
			}
			var details string
			for _, d := range resp.Diagnostics.Errors() {
				details += d.Summary() + " " + d.Detail() + "\n"
			}
			if !tc.wantErr.MatchString(details) {
				t.Fatalf("error %q does not match %s", details, tc.wantErr)
			}
		})
	}
}
