package validate_test

import (
	"context"
	"testing"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestFQDN_Valid(t *testing.T) {
	t.Parallel()
	valid := []string{
		"https://app.example.com",
		"http://localhost:8080",
		"https://coolify.io/api",
		"http://192.168.1.1:3000",
	}
	v := validate.FQDN()
	for _, s := range valid {
		resp := validator.StringResponse{}
		v.ValidateString(context.Background(), validator.StringRequest{
			ConfigValue: types.StringValue(s),
		}, &resp)
		if resp.Diagnostics.HasError() {
			t.Errorf("FQDN(%q) should be valid, got error: %s", s, resp.Diagnostics.Errors()[0].Detail())
		}
	}
}

func TestFQDN_Invalid(t *testing.T) {
	t.Parallel()
	invalid := []string{
		"app.example.com",
		"ftp://files.example.com",
		"https://",
		"not-a-url",
		"",
	}
	v := validate.FQDN()
	for _, s := range invalid {
		resp := validator.StringResponse{}
		v.ValidateString(context.Background(), validator.StringRequest{
			ConfigValue: types.StringValue(s),
		}, &resp)
		if !resp.Diagnostics.HasError() {
			t.Errorf("FQDN(%q) should be invalid, got no error", s)
		}
	}
}

func TestFQDN_NullAndUnknown(t *testing.T) {
	t.Parallel()
	v := validate.FQDN()

	nullResp := validator.StringResponse{}
	v.ValidateString(context.Background(), validator.StringRequest{
		ConfigValue: types.StringNull(),
	}, &nullResp)
	if nullResp.Diagnostics.HasError() {
		t.Error("FQDN(null) should pass, got error")
	}

	unknownResp := validator.StringResponse{}
	v.ValidateString(context.Background(), validator.StringRequest{
		ConfigValue: types.StringUnknown(),
	}, &unknownResp)
	if unknownResp.Diagnostics.HasError() {
		t.Error("FQDN(unknown) should pass, got error")
	}
}
