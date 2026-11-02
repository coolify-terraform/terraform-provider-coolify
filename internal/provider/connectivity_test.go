package provider

import (
	"context"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProviderSchema_ConnectivitySecretsAreSensitive(t *testing.T) {
	t.Parallel()
	var resp provider.SchemaResponse
	New("test")().Schema(context.Background(), provider.SchemaRequest{}, &resp)

	key, ok := resp.Schema.Attributes["client_key"].(schema.StringAttribute)
	require.True(t, ok)
	assert.True(t, key.Sensitive)

	proxy, ok := resp.Schema.Blocks["proxy"].(schema.SingleNestedBlock)
	require.True(t, ok)
	password, ok := proxy.Attributes["password"].(schema.StringAttribute)
	require.True(t, ok)
	assert.True(t, password.Sensitive)
}

func TestApplyClientCert_UnknownIsAnError(t *testing.T) {
	t.Parallel()
	var cfg client.RetryConfig
	diags := applyClientCert(coolifyProviderModel{
		ClientCert: types.StringValue("pem"),
		ClientKey:  types.StringUnknown(),
	}, &cfg)
	require.True(t, diags.HasError())
	assert.Empty(t, cfg.ClientKey)
}

func TestApplyRequestTimeout_BadEnv(t *testing.T) {
	t.Setenv("COOLIFY_REQUEST_TIMEOUT", "nope")
	var cfg client.RetryConfig
	diags := applyRequestTimeout(coolifyProviderModel{RequestTimeout: types.Int64Null()}, &cfg)
	require.True(t, diags.HasError())
}

func TestApplyProxy_HTTPSDoesNotEchoPassword(t *testing.T) {
	t.Parallel()
	var cfg client.RetryConfig
	diags := applyProxy(coolifyProviderModel{
		Proxy: &proxyModel{
			URL:      types.StringValue("https://user:s3cr3t-value@proxy.example:8443"),
			Username: types.StringNull(),
			Password: types.StringNull(),
			NoProxy:  types.StringNull(),
		},
	}, &cfg)
	require.True(t, diags.HasError())
	assert.NotContains(t, diags[0].Detail(), "s3cr3t-value")
}
