package environmentvariable

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func previewValidateConfig(t *testing.T, attrs map[string]any) resource.ValidateConfigResponse {
	t.Helper()
	ctx := context.Background()
	r := &environmentVariableResource{}
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema: %v", schemaResp.Diagnostics.Errors())
	}
	state := tfsdk.State{
		Schema: schemaResp.Schema,
		Raw:    tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil),
	}
	for name, val := range attrs {
		diags := state.SetAttribute(ctx, path.Root(name), val)
		if diags.HasError() {
			t.Fatalf("set %s: %v", name, diags.Errors())
		}
	}
	resp := resource.ValidateConfigResponse{}
	r.ValidateConfig(ctx, resource.ValidateConfigRequest{Config: tfsdk.Config(state)}, &resp)
	return resp
}

func TestRejectPreview_UnknownApplicationUUID(t *testing.T) {
	t.Parallel()
	resp := previewValidateConfig(t, map[string]any{
		"application_uuid": types.StringUnknown(),
		"key":              types.StringValue("TOKEN"),
		"value":            types.StringValue("secret"),
		"is_preview":       types.BoolValue(true),
	})
	if resp.Diagnostics.HasError() {
		t.Fatalf("unknown application_uuid with is_preview=true: %v", resp.Diagnostics.Errors())
	}
}

func TestRejectPreview_KnownApplicationUUID(t *testing.T) {
	t.Parallel()
	resp := previewValidateConfig(t, map[string]any{
		"application_uuid": types.StringValue("aaaa0001-0001-4000-8000-000000000001"),
		"key":              types.StringValue("TOKEN"),
		"value":            types.StringValue("secret"),
		"is_preview":       types.BoolValue(true),
	})
	if resp.Diagnostics.HasError() {
		t.Fatalf("known application uuid: %v", resp.Diagnostics.Errors())
	}
}

func TestRejectPreview_UnknownServiceUUID(t *testing.T) {
	t.Parallel()
	resp := previewValidateConfig(t, map[string]any{
		"service_uuid": types.StringUnknown(),
		"key":          types.StringValue("TOKEN"),
		"value":        types.StringValue("secret"),
		"is_preview":   types.BoolValue(true),
	})
	if !resp.Diagnostics.HasError() {
		t.Fatal("unknown service uuid with is_preview=true produced no error")
	}
}

func TestRejectPreview_UnknownDatabaseUUID(t *testing.T) {
	t.Parallel()
	resp := previewValidateConfig(t, map[string]any{
		"database_uuid": types.StringUnknown(),
		"key":           types.StringValue("TOKEN"),
		"value":         types.StringValue("secret"),
		"is_preview":    types.BoolValue(true),
	})
	if !resp.Diagnostics.HasError() {
		t.Fatal("unknown database uuid with is_preview=true produced no error")
	}
}

func TestRejectPreview_KnownServiceUUID(t *testing.T) {
	t.Parallel()
	resp := previewValidateConfig(t, map[string]any{
		"service_uuid": types.StringValue("dddd0001-0001-4000-8000-000000000001"),
		"key":          types.StringValue("TOKEN"),
		"value":        types.StringValue("secret"),
		"is_preview":   types.BoolValue(true),
	})
	if !resp.Diagnostics.HasError() {
		t.Fatal("known service uuid with is_preview=true produced no error")
	}
}
