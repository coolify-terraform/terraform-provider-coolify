package applicationpreview

import (
	"errors"
	"strings"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestPreviewDomainInput(t *testing.T) {
	t.Parallel()
	url := "https://pr.example.com"
	composeJSON := `[{"name":"web","domain":"https://pr.example.com"}]`
	tests := []struct {
		name                 string
		domains              types.String
		dockerComposeDomains types.String
		force                types.Bool
		wantOK               bool
		wantErr              bool
		wantDomains          *string
		wantCompose          string
		wantForce            bool
	}{
		{
			name:                 "empty domains",
			domains:              types.StringValue(""),
			dockerComposeDomains: types.StringNull(),
			force:                types.BoolNull(),
		},
		{
			name:                 "whitespace domains",
			domains:              types.StringValue(" "),
			dockerComposeDomains: types.StringNull(),
			force:                types.BoolNull(),
		},
		{
			name:                 "comma-only domains",
			domains:              types.StringValue(","),
			dockerComposeDomains: types.StringNull(),
			force:                types.BoolNull(),
		},
		{
			name:                 "real URL",
			domains:              types.StringValue("https://pr.example.com"),
			dockerComposeDomains: types.StringNull(),
			force:                types.BoolNull(),
			wantOK:               true,
			wantDomains:          &url,
		},
		{
			name:                 "empty compose array",
			domains:              types.StringNull(),
			dockerComposeDomains: types.StringValue("[]"),
			force:                types.BoolNull(),
		},
		{
			name:                 "empty compose",
			domains:              types.StringNull(),
			dockerComposeDomains: types.StringValue(""),
			force:                types.BoolNull(),
		},
		{
			name:                 "valid compose JSON",
			domains:              types.StringNull(),
			dockerComposeDomains: types.StringValue(composeJSON),
			force:                types.BoolNull(),
			wantOK:               true,
			wantCompose:          composeJSON,
		},
		{
			name:                 "invalid compose JSON",
			domains:              types.StringNull(),
			dockerComposeDomains: types.StringValue("not-json"),
			force:                types.BoolNull(),
			wantErr:              true,
		},
		{
			name:                 "compose object is invalid JSON array",
			domains:              types.StringNull(),
			dockerComposeDomains: types.StringValue(`{"web":{"domain":"https://pr.example.com"}}`),
			force:                types.BoolNull(),
			wantErr:              true,
		},
		{
			name:                 "force only is not a domain write",
			domains:              types.StringNull(),
			dockerComposeDomains: types.StringNull(),
			force:                types.BoolValue(true),
		},
		{
			name:                 "real URL with force true",
			domains:              types.StringValue("https://pr.example.com"),
			dockerComposeDomains: types.StringNull(),
			force:                types.BoolValue(true),
			wantOK:               true,
			wantDomains:          &url,
			wantForce:            true,
		},
		{
			name:                 "real URL with force false",
			domains:              types.StringValue("https://pr.example.com"),
			dockerComposeDomains: types.StringNull(),
			force:                types.BoolValue(false),
			wantOK:               true,
			wantDomains:          &url,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := applicationPreviewModel{
				Domains:              tt.domains,
				DockerComposeDomains: tt.dockerComposeDomains,
				ForceDomainOverride:  tt.force,
			}
			got, ok, err := m.previewDomainInput()
			if tt.wantErr {
				if err == nil {
					t.Fatal("previewDomainInput() error = nil, want error")
				}
				if !strings.Contains(err.Error(), "docker_compose_domains must be a JSON array") {
					t.Errorf("previewDomainInput() error = %v, want JSON array error", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("previewDomainInput() unexpected error: %v", err)
			}
			if ok != tt.wantOK {
				t.Errorf("previewDomainInput() ok = %v, want %v", ok, tt.wantOK)
			}
			if tt.wantDomains == nil {
				if got.Domains != nil {
					t.Errorf("Domains = %v, want nil", *got.Domains)
				}
			} else if got.Domains == nil || *got.Domains != *tt.wantDomains {
				t.Errorf("Domains = %v, want %q", got.Domains, *tt.wantDomains)
			}
			if string(got.DockerComposeDomains) == "[]" {
				t.Fatal("DockerComposeDomains must not be []")
			}
			if tt.wantCompose == "" {
				if len(got.DockerComposeDomains) != 0 {
					t.Errorf("DockerComposeDomains = %s, want empty (not sent)", got.DockerComposeDomains)
				}
			} else if string(got.DockerComposeDomains) != tt.wantCompose {
				t.Errorf("DockerComposeDomains = %s, want %s", got.DockerComposeDomains, tt.wantCompose)
			}
			if tt.wantForce {
				if got.ForceDomainOverride == nil || !*got.ForceDomainOverride {
					t.Errorf("ForceDomainOverride = %v, want true", got.ForceDomainOverride)
				}
			} else if got.ForceDomainOverride != nil {
				t.Errorf("ForceDomainOverride = %v, want nil", *got.ForceDomainOverride)
			}
		})
	}
}

func TestAnnotatePreviewDomainError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "not found",
			err:  &client.NotFoundError{Message: "resource not found (PATCH /previews/1): Preview not found."},
			want: "Coolify has no preview for this PR yet",
		},
		{
			name: "status 404 text",
			err:  errors.New("api error (status 404) for PATCH /previews/1: Preview not found."),
			want: "Coolify has no preview for this PR yet",
		},
		{
			name: "conflict",
			err:  &client.APIStatusError{Status: 409, Message: "Domain conflict."},
			want: "force_domain_override = true",
		},
		{
			name: "status 409 text",
			err:  errors.New("api error (status 409) for PATCH /previews/1: Domain conflict."),
			want: "force_domain_override = true",
		},
		{
			name: "other",
			err:  errors.New("internal server error"),
			want: "internal server error",
		},
		{
			name: "nil",
			err:  nil,
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := annotatePreviewDomainError(tt.err)
			if !strings.Contains(got, tt.want) {
				t.Errorf("annotatePreviewDomainError() = %q, want substring %q", got, tt.want)
			}
		})
	}
}
