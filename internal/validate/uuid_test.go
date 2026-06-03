package validate_test

import (
	"context"
	"strings"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestUUID_Valid(t *testing.T) {
	t.Parallel()
	valid := []string{
		"550e8400-e29b-41d4-a716-446655440000",
		"aaaa0001-0001-4000-8000-000000000001",
		"00000000-0000-0000-0000-000000000000",
		"ABCDEF12-3456-7890-ABCD-EF1234567890",
		// Coolify NanoID format
		"deey8xhb2bm3fxpobcxyddfv",             // real NanoID (24 chars)
		"abcdefghij0123456789",                 // boundary: exactly 20 chars
		"abcdefghij0123456789ABCDEFGHIJ012345", // boundary: exactly 36 chars
		"ABCDEFghij0123456789abcdef",           // mixed case (26 chars)
	}
	v := validate.UUID()
	for _, s := range valid {
		resp := validator.StringResponse{}
		v.ValidateString(context.Background(), validator.StringRequest{
			ConfigValue: types.StringValue(s),
		}, &resp)
		if resp.Diagnostics.HasError() {
			t.Errorf("UUID(%q) should be valid, got error: %s", s, resp.Diagnostics.Errors()[0].Detail())
		}
	}
}

func TestUUID_Invalid(t *testing.T) {
	t.Parallel()
	invalid := []string{
		"not-a-uuid",
		"proj-uuid-1",
		"12345",
		"550e8400-e29b-41d4-a716",
		"550e8400-e29b-41d4-a716-44665544000g",
		"",
		// NanoID boundary violations
		"abcdefghij012345678",                   // 19 chars (min - 1)
		"abcdefghij0123456789ABCDEFGHIJ0123456", // 37 chars (max + 1)
		"abcdef_ghij01234567890",                // underscore not in [a-zA-Z0-9]
		"abcdef ghij01234567890",                // space not in [a-zA-Z0-9]
	}
	v := validate.UUID()
	for _, s := range invalid {
		resp := validator.StringResponse{}
		v.ValidateString(context.Background(), validator.StringRequest{
			ConfigValue: types.StringValue(s),
		}, &resp)
		if !resp.Diagnostics.HasError() {
			t.Errorf("UUID(%q) should be invalid, got no error", s)
		}
	}
}

func TestUUID_NullAndUnknown(t *testing.T) {
	t.Parallel()
	v := validate.UUID()

	// Null values should pass (optional field not set)
	nullResp := validator.StringResponse{}
	v.ValidateString(context.Background(), validator.StringRequest{
		ConfigValue: types.StringNull(),
	}, &nullResp)
	if nullResp.Diagnostics.HasError() {
		t.Error("UUID(null) should pass, got error")
	}

	// Unknown values should pass (computed at apply time)
	unknownResp := validator.StringResponse{}
	v.ValidateString(context.Background(), validator.StringRequest{
		ConfigValue: types.StringUnknown(),
	}, &unknownResp)
	if unknownResp.Diagnostics.HasError() {
		t.Error("UUID(unknown) should pass, got error")
	}
}

func TestIsUUID(t *testing.T) {
	t.Parallel()
	if !validate.IsUUID("550e8400-e29b-41d4-a716-446655440000") {
		t.Error("expected valid UUID to return true")
	}
	if !validate.IsUUID("deey8xhb2bm3fxpobcxyddfv") {
		t.Error("expected valid NanoID to return true")
	}
	if validate.IsUUID("not-a-uuid") {
		t.Error("expected invalid string to return false")
	}
	if validate.IsUUID("../../admin") {
		t.Error("expected path traversal to return false")
	}
	if validate.IsUUID("") {
		t.Error("expected empty string to return false")
	}
}

func TestImportUUID_Valid(t *testing.T) {
	t.Parallel()
	if err := validate.ImportUUID("550e8400-e29b-41d4-a716-446655440000"); err != nil {
		t.Errorf("expected nil error for valid UUID, got: %v", err)
	}
}

func TestImportUUID_Invalid(t *testing.T) {
	t.Parallel()
	err := validate.ImportUUID("../../admin")
	if err == nil {
		t.Fatal("expected error for path traversal ID")
	}
	if !strings.Contains(err.Error(), "../../admin") {
		t.Errorf("error should contain the bad ID, got: %v", err)
	}
}

func TestParseCompoundImportID_SimpleUUID(t *testing.T) {
	t.Parallel()
	parsed, compound, err := validate.ParseCompoundImportID("deey8xhb2bm3fxpobcxyddfv")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if compound {
		t.Error("expected compound=false for simple UUID")
	}
	if parsed.UUID != "deey8xhb2bm3fxpobcxyddfv" {
		t.Errorf("expected UUID=%q, got %q", "deey8xhb2bm3fxpobcxyddfv", parsed.UUID)
	}
}

func TestParseCompoundImportID_CompoundFormat(t *testing.T) {
	t.Parallel()
	id := "550e8400-e29b-41d4-a716-446655440000:aaaa0001-0001-4000-8000-000000000001:production:deey8xhb2bm3fxpobcxyddfv"
	parsed, compound, err := validate.ParseCompoundImportID(id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !compound {
		t.Error("expected compound=true for 4-part ID")
	}
	if parsed.ProjectUUID != "550e8400-e29b-41d4-a716-446655440000" {
		t.Errorf("wrong ProjectUUID: %s", parsed.ProjectUUID)
	}
	if parsed.ServerUUID != "aaaa0001-0001-4000-8000-000000000001" {
		t.Errorf("wrong ServerUUID: %s", parsed.ServerUUID)
	}
	if parsed.EnvironmentName != "production" {
		t.Errorf("wrong EnvironmentName: %s", parsed.EnvironmentName)
	}
	if parsed.UUID != "deey8xhb2bm3fxpobcxyddfv" {
		t.Errorf("wrong UUID: %s", parsed.UUID)
	}
}

func TestParseCompoundImportID_InvalidFormats(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		id   string
		want string
	}{
		{"two parts", "abc:def", "expected UUID or"},
		{"three parts", "a:b:c", "expected UUID or"},
		{"invalid simple UUID", "../../admin", "not a valid UUID"},
		{"empty env name", "deey8xhb2bm3fxpobcxyddfv:deey8xhb2bm3fxpobcxyddfv::deey8xhb2bm3fxpobcxyddfv", "environment_name must not be empty"},
		{"invalid project UUID", "bad:deey8xhb2bm3fxpobcxyddfv:prod:deey8xhb2bm3fxpobcxyddfv", "project_uuid"},
		{"invalid server UUID", "deey8xhb2bm3fxpobcxyddfv:bad:prod:deey8xhb2bm3fxpobcxyddfv", "server_uuid"},
		{"invalid resource UUID", "deey8xhb2bm3fxpobcxyddfv:deey8xhb2bm3fxpobcxyddfv:prod:bad", "uuid"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, _, err := validate.ParseCompoundImportID(tc.id)
			if err == nil {
				t.Fatalf("expected error for %q", tc.id)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error %q should contain %q", err.Error(), tc.want)
			}
		})
	}
}

// importParentChildSchema returns a minimal schema for testing ImportParentChild
// with the given parent type attributes plus "uuid".
func importParentChildSchema(parentTypes []string) schema.Schema {
	attrs := map[string]schema.Attribute{
		"uuid": schema.StringAttribute{Computed: true},
	}
	for _, t := range parentTypes {
		attrs[t+"_uuid"] = schema.StringAttribute{Optional: true}
	}
	return schema.Schema{Attributes: attrs}
}

// newImportStateResponse creates a response with an empty state for the given schema.
func newImportStateResponse(s schema.Schema) *resource.ImportStateResponse {
	ctx := context.Background()
	return &resource.ImportStateResponse{
		State: tfsdk.State{
			Schema: s,
			Raw:    tftypes.NewValue(s.Type().TerraformType(ctx), nil),
		},
	}
}

func TestImportParentChild_Valid(t *testing.T) {
	t.Parallel()
	parentUUID := "550e8400-e29b-41d4-a716-446655440000"
	childUUID := "aaaa0001-0001-4000-8000-000000000001"
	allowedTypes := []string{"application", "service", "database"}

	for _, typ := range allowedTypes {
		t.Run(typ, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			s := importParentChildSchema(allowedTypes)
			resp := newImportStateResponse(s)
			req := resource.ImportStateRequest{ID: typ + ":" + parentUUID + ":" + childUUID}

			validate.ImportParentChild(ctx, req, resp, allowedTypes, "storage")

			if resp.Diagnostics.HasError() {
				t.Fatalf("unexpected error: %s", resp.Diagnostics.Errors()[0].Detail())
			}

			var gotParent, gotChild types.String
			resp.State.GetAttribute(ctx, path.Root(typ+"_uuid"), &gotParent)
			resp.State.GetAttribute(ctx, path.Root("uuid"), &gotChild)

			if gotParent.ValueString() != parentUUID {
				t.Errorf("expected %s_uuid=%s, got %s", typ, parentUUID, gotParent.ValueString())
			}
			if gotChild.ValueString() != childUUID {
				t.Errorf("expected uuid=%s, got %s", childUUID, gotChild.ValueString())
			}
		})
	}
}

func TestImportParentChild_WrongFormat(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		id   string
	}{
		{"single value", "just-a-uuid"},
		{"two parts", "application:uuid1"},
		{"four parts", "application:uuid1:uuid2:uuid3"},
		{"empty", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			s := importParentChildSchema([]string{"application"})
			resp := newImportStateResponse(s)
			req := resource.ImportStateRequest{ID: tc.id}

			validate.ImportParentChild(ctx, req, resp, []string{"application"}, "storage")

			if !resp.Diagnostics.HasError() {
				t.Fatal("expected error for wrong format")
			}
		})
	}
}

func TestImportParentChild_InvalidParentUUID(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s := importParentChildSchema([]string{"application"})
	resp := newImportStateResponse(s)
	req := resource.ImportStateRequest{ID: "application:../../admin:550e8400-e29b-41d4-a716-446655440000"}

	validate.ImportParentChild(ctx, req, resp, []string{"application"}, "storage")

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for invalid parent UUID")
	}
	detail := resp.Diagnostics.Errors()[0].Detail()
	if !strings.Contains(detail, "parent UUID") {
		t.Errorf("error should mention parent UUID, got: %s", detail)
	}
}

func TestImportParentChild_InvalidChildUUID(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s := importParentChildSchema([]string{"application"})
	resp := newImportStateResponse(s)
	req := resource.ImportStateRequest{ID: "application:550e8400-e29b-41d4-a716-446655440000:bad!"}

	validate.ImportParentChild(ctx, req, resp, []string{"application"}, "storage")

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for invalid child UUID")
	}
	detail := resp.Diagnostics.Errors()[0].Detail()
	if !strings.Contains(detail, "storage UUID") {
		t.Errorf("error should mention storage UUID, got: %s", detail)
	}
}

func TestImportParentChild_InvalidType(t *testing.T) {
	t.Parallel()
	parentUUID := "550e8400-e29b-41d4-a716-446655440000"
	childUUID := "aaaa0001-0001-4000-8000-000000000001"
	ctx := context.Background()
	s := importParentChildSchema([]string{"application", "service"})
	resp := newImportStateResponse(s)
	req := resource.ImportStateRequest{ID: "badtype:" + parentUUID + ":" + childUUID}

	validate.ImportParentChild(ctx, req, resp, []string{"application", "service"}, "storage")

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for invalid type")
	}
	detail := resp.Diagnostics.Errors()[0].Detail()
	if !strings.Contains(detail, "badtype") {
		t.Errorf("error should mention the bad type, got: %s", detail)
	}
	if !strings.Contains(detail, "application") || !strings.Contains(detail, "service") {
		t.Errorf("error should list allowed types, got: %s", detail)
	}
}

func TestJoinOr(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		id    string
		types []string
		want  string
	}{
		{
			name:  "single type hint",
			id:    "badtype:550e8400-e29b-41d4-a716-446655440000:aaaa0001-0001-4000-8000-000000000001",
			types: []string{"application"},
			want:  `"application"`,
		},
		{
			name:  "two types hint",
			id:    "badtype:550e8400-e29b-41d4-a716-446655440000:aaaa0001-0001-4000-8000-000000000001",
			types: []string{"application", "service"},
			want:  " or ",
		},
		{
			name:  "three types hint",
			id:    "badtype:550e8400-e29b-41d4-a716-446655440000:aaaa0001-0001-4000-8000-000000000001",
			types: []string{"application", "service", "database"},
			want:  ", or ",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			s := importParentChildSchema(tc.types)
			resp := newImportStateResponse(s)
			req := resource.ImportStateRequest{ID: tc.id}

			validate.ImportParentChild(ctx, req, resp, tc.types, "storage")

			if !resp.Diagnostics.HasError() {
				t.Fatal("expected error")
			}
			detail := resp.Diagnostics.Errors()[0].Detail()
			if !strings.Contains(detail, tc.want) {
				t.Errorf("error should contain %q, got: %s", tc.want, detail)
			}
		})
	}
}
