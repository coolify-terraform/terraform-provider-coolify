package hetzner

import (
	"fmt"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

func cloudProviderTokenUUIDAttribute(resourceLabel string) schema.StringAttribute {
	return schema.StringAttribute{
		MarkdownDescription: fmt.Sprintf("The UUID of the cloud provider token to use for listing Hetzner %s.", resourceLabel),
		Required:            true,
		Validators:          []validator.String{validate.UUID()},
	}
}

func configureHetznerDataSourceClient(req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) *client.Client {
	return flex.ConfigureDataSourceClient(req, &resp.Diagnostics)
}
