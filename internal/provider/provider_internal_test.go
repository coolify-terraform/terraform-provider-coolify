package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

func TestBuildClientConfig(t *testing.T) {
	// t.Parallel() intentionally omitted: t.Setenv is incompatible with parallel tests.
	tests := []struct {
		name         string
		config       coolifyProviderModel
		envCACert    string
		envInsecure  string
		wantCACert   string
		wantInsecure bool
	}{
		{
			name:   "neither set",
			config: coolifyProviderModel{CACert: types.StringNull(), Insecure: types.BoolNull()},
		},
		{
			name:         "env var only",
			config:       coolifyProviderModel{CACert: types.StringNull(), Insecure: types.BoolNull()},
			envCACert:    "env-cert-pem",
			envInsecure:  "true",
			wantCACert:   "env-cert-pem",
			wantInsecure: true,
		},
		{
			name:         "schema overrides env",
			config:       coolifyProviderModel{CACert: types.StringValue("schema-cert"), Insecure: types.BoolValue(false)},
			envCACert:    "env-cert",
			envInsecure:  "true",
			wantCACert:   "schema-cert",
			wantInsecure: false,
		},
		{
			name:         "schema only",
			config:       coolifyProviderModel{CACert: types.StringValue("my-ca"), Insecure: types.BoolValue(true)},
			wantCACert:   "my-ca",
			wantInsecure: true,
		},
		{
			name:         "insecure env case insensitive",
			config:       coolifyProviderModel{CACert: types.StringNull(), Insecure: types.BoolNull()},
			envInsecure:  "TRUE",
			wantInsecure: true,
		},
		{
			name:         "insecure env false",
			config:       coolifyProviderModel{CACert: types.StringNull(), Insecure: types.BoolNull()},
			envInsecure:  "false",
			wantInsecure: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Always set both env vars so each subtest starts from a known state.
			t.Setenv("COOLIFY_CA_CERT", tt.envCACert)
			t.Setenv("COOLIFY_INSECURE", tt.envInsecure)
			cfg := buildClientConfig(tt.config)
			assert.Equal(t, tt.wantCACert, cfg.CACert)
			assert.Equal(t, tt.wantInsecure, cfg.Insecure)
		})
	}
}
