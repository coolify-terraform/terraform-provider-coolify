package flex

import (
	"fmt"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// ConfigureClient extracts the *client.Client from provider data during
// resource Configure. Returns nil when ProviderData is nil (early call
// before provider configuration). Adds an error diagnostic if the type
// assertion fails.
func ConfigureClient(req resource.ConfigureRequest, diags *diag.Diagnostics) *client.Client {
	if req.ProviderData == nil {
		return nil
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		diags.AddError(
			"Unexpected Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData),
		)
		return nil
	}
	return c
}

// ConfigureDataSourceClient extracts the *client.Client from provider data
// during data source Configure. Returns nil when ProviderData is nil (early
// call before provider configuration). Adds an error diagnostic if the type
// assertion fails.
func ConfigureDataSourceClient(req datasource.ConfigureRequest, diags *diag.Diagnostics) *client.Client {
	if req.ProviderData == nil {
		return nil
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		diags.AddError(
			"Unexpected Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData),
		)
		return nil
	}
	return c
}
