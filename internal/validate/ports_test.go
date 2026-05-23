package validate_test

import (
	"context"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestPortMappings(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"empty string", "", true},
		{"valid single", "8080:5432", false},
		{"valid multiple", "8080:5432,8443:5433", false},
		{"valid edge low", "1:1", false},
		{"valid edge high", "65535:65535", false},
		{"zero host port", "0:5432", true},
		{"zero container port", "8080:0", true},
		{"host port too high", "65536:5432", true},
		{"container port too high", "8080:65536", true},
		{"negative port", "-1:5432", true},
		{"missing colon", "8080", true},
		{"non-numeric", "abc:5432", true},
		{"empty pair", "8080:5432,,8443:5433", true},
		{"whitespace after comma", "8080:5432, 8443:5433", false},
		{"whitespace around colon", " 8080 : 5432 ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validator.StringRequest{
				ConfigValue: types.StringValue(tt.value),
			}
			resp := &validator.StringResponse{}
			validate.PortMappings().ValidateString(context.Background(), req, resp)

			if tt.wantErr && !resp.Diagnostics.HasError() {
				t.Errorf("expected error for %q but got none", tt.value)
			}
			if !tt.wantErr && resp.Diagnostics.HasError() {
				t.Errorf("unexpected error for %q: %s", tt.value, resp.Diagnostics.Errors()[0].Detail())
			}
		})
	}
}

func TestPortMappings_NullAndUnknown(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value types.String
	}{
		{"null", types.StringNull()},
		{"unknown", types.StringUnknown()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := validator.StringRequest{ConfigValue: tc.value}
			resp := &validator.StringResponse{}
			validate.PortMappings().ValidateString(context.Background(), req, resp)
			if resp.Diagnostics.HasError() {
				t.Errorf("expected no error for %s value, got: %s", tc.name, resp.Diagnostics.Errors()[0].Detail())
			}
		})
	}
}
