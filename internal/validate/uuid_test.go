package validate_test

import (
	"context"
	"testing"

	"github.com/SebTardif/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestUUID_Valid(t *testing.T) {
	t.Parallel()
	valid := []string{
		"550e8400-e29b-41d4-a716-446655440000",
		"aaaa0001-0001-4000-8000-000000000001",
		"00000000-0000-0000-0000-000000000000",
		"ABCDEF12-3456-7890-ABCD-EF1234567890",
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