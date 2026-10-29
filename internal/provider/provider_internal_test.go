package provider

import (
	"context"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestConfigureHeaders(t *testing.T) {
	// t.Parallel() intentionally omitted: t.Setenv is incompatible with parallel tests.
	headersMap := func(m map[string]string) types.Map {
		elems := make(map[string]attr.Value, len(m))
		for k, v := range m {
			elems[k] = types.StringValue(v)
		}
		return types.MapValueMust(types.StringType, elems)
	}
	tests := []struct {
		name        string
		headers     types.Map
		env         string
		want        map[string]string
		wantErr     string
		wantNoValue string
	}{
		{name: "neither set", headers: types.MapNull(types.StringType)},
		{name: "empty map", headers: headersMap(map[string]string{})},
		{
			name:    "attribute only",
			headers: headersMap(map[string]string{"Proxy-Authorization": "Bearer jwt"}),
			want:    map[string]string{"Proxy-Authorization": "Bearer jwt"},
		},
		{
			name:    "env only",
			headers: types.MapNull(types.StringType),
			env:     `{"Proxy-Authorization":"Bearer env-jwt"}`,
			want:    map[string]string{"Proxy-Authorization": "Bearer env-jwt"},
		},
		{
			name:    "attribute overrides env",
			headers: headersMap(map[string]string{"X-From": "attribute"}),
			env:     `{"Proxy-Authorization":"Bearer env-jwt"}`,
			want:    map[string]string{"X-From": "attribute"},
		},
		{
			name:    "empty attribute overrides env",
			headers: headersMap(map[string]string{}),
			env:     `{"Proxy-Authorization":"Bearer env-jwt"}`,
			want:    map[string]string{},
		},
		{
			name:    "unknown attribute falls back to env",
			headers: types.MapUnknown(types.StringType),
			env:     `{"X-From":"env"}`,
			want:    map[string]string{"X-From": "env"},
		},
		{name: "blank env", headers: types.MapNull(types.StringType), env: "  "},
		{
			name:        "env not a JSON object of strings",
			headers:     types.MapNull(types.StringType),
			env:         `{"X-Secret":"s3cr3t-value","X-Number":1}`,
			wantErr:     "COOLIFY_HEADERS must be a JSON object of strings",
			wantNoValue: "s3cr3t-value",
		},
		{
			name:        "reserved header in attribute",
			headers:     headersMap(map[string]string{"Authorization": "Bearer s3cr3t-value"}),
			wantErr:     `The headers attribute is not valid:` + "\n" + `"Authorization" cannot be set`,
			wantNoValue: "s3cr3t-value",
		},
		{
			name:        "reserved header in env",
			headers:     types.MapNull(types.StringType),
			env:         `{"Content-Type":"text/s3cr3t-value"}`,
			wantErr:     `The COOLIFY_HEADERS environment variable is not valid:` + "\n" + `"Content-Type" cannot be set`,
			wantNoValue: "s3cr3t-value",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("COOLIFY_HEADERS", tt.env)
			cfg := client.RetryConfig{}
			diags := configureHeaders(context.Background(), coolifyProviderModel{Headers: tt.headers}, &cfg)
			if tt.wantErr != "" {
				require.True(t, diags.HasError())
				detail := diags.Errors()[0].Detail()
				assert.Contains(t, detail, tt.wantErr)
				assert.NotContains(t, detail, tt.wantNoValue)
				return
			}
			require.False(t, diags.HasError(), "%v", diags)
			if len(tt.want) == 0 {
				assert.Empty(t, cfg.Headers)
				return
			}
			assert.Equal(t, tt.want, cfg.Headers)
		})
	}
}
