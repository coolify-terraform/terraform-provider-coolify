package validate_test

import (
	"context"
	"testing"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestNoShellMetachars(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid flags", "--memory=512m --cpus=2", false},
		{"valid privileged", "--privileged", false},
		{"valid network", "--network=host", false},
		{"semicolon", "--memory=512m; rm -rf /", true},
		{"pipe", "--memory=512m | cat", true},
		{"ampersand", "--memory=512m & echo pwned", true},
		{"backtick", "--memory=`whoami`", true},
		{"dollar", "--memory=$HOME", true},
		{"parens", "--memory=$(whoami)", true},
		{"braces", "--memory=${HOME}", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validator.StringRequest{
				ConfigValue: types.StringValue(tt.value),
			}
			resp := &validator.StringResponse{}
			validate.NoShellMetachars().ValidateString(context.Background(), req, resp)

			if tt.wantErr && !resp.Diagnostics.HasError() {
				t.Errorf("expected error for %q but got none", tt.value)
			}
			if !tt.wantErr && resp.Diagnostics.HasError() {
				t.Errorf("unexpected error for %q: %s", tt.value, resp.Diagnostics.Errors()[0].Detail())
			}
		})
	}
}

func TestNoShellMetachars_NullAndUnknown(t *testing.T) {
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
			validate.NoShellMetachars().ValidateString(context.Background(), req, resp)
			if resp.Diagnostics.HasError() {
				t.Errorf("expected no error for %s value, got: %s", tc.name, resp.Diagnostics.Errors()[0].Detail())
			}
		})
	}
}
