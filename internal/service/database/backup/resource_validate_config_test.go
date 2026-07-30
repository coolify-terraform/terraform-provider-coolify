package backup

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

func databaseBackupValidateConfig(t *testing.T, saveS3 types.Bool, s3UUID types.String) resource.ValidateConfigResponse {
	t.Helper()
	ctx := context.Background()
	r := &databaseBackupResource{}

	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema: %v", schemaResp.Diagnostics.Errors())
	}

	state := tfsdk.State{
		Schema: schemaResp.Schema,
		Raw:    tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil),
	}
	for _, set := range []struct {
		p path.Path
		v any
	}{
		{path.Root("database_uuid"), types.StringValue("eeee0001-0001-4000-8000-000000000001")},
		{path.Root("frequency"), types.StringValue("0 2 * * *")},
		{path.Root("save_s3"), saveS3},
		{path.Root("s3_storage_uuid"), s3UUID},
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

func TestDatabaseBackupResource_ValidateConfig_UnknownS3UUID(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		saveS3  types.Bool
		s3UUID  types.String
		wantErr *regexp.Regexp
	}{
		{
			name:    "unknown s3 uuid with save_s3 true must not error",
			saveS3:  types.BoolValue(true),
			s3UUID:  types.StringUnknown(),
			wantErr: nil,
		},
		{
			name:    "unknown save_s3 must not error",
			saveS3:  types.BoolUnknown(),
			s3UUID:  types.StringNull(),
			wantErr: nil,
		},
		{
			name:    "known save_s3 true without uuid still errors",
			saveS3:  types.BoolValue(true),
			s3UUID:  types.StringNull(),
			wantErr: regexp.MustCompile(`Missing S3 Storage UUID`),
		},
		{
			name:    "valid pair ok",
			saveS3:  types.BoolValue(true),
			s3UUID:  types.StringValue("cccc0003-0003-4000-8000-000000000003"),
			wantErr: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			resp := databaseBackupValidateConfig(t, tc.saveS3, tc.s3UUID)
			hasErr := resp.Diagnostics.HasError()
			if tc.wantErr == nil {
				if hasErr {
					t.Fatalf("unexpected errors: %v", resp.Diagnostics.Errors())
				}
				return
			}
			if !hasErr {
				t.Fatal("expected validation error, got none")
			}
			var joined string
			for _, d := range resp.Diagnostics.Errors() {
				joined += d.Detail() + " " + d.Summary()
			}
			if !tc.wantErr.MatchString(joined) {
				t.Fatalf("error %q does not match %s", joined, tc.wantErr)
			}
		})
	}
}
