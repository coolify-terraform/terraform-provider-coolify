package hetzner

import (
	"context"
	"fmt"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/filter"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-log/tflog"
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

func readFilteredTokenList[T any](
	ctx context.Context,
	tokenUUID string,
	filters []filter.Config,
	dataSourceType string,
	listErrorSummary string,
	resp *datasource.ReadResponse,
	c *client.Client,
	listFn func(context.Context, string) ([]T, error),
	accessor func(T, string) (string, bool),
) ([]T, bool) {
	tflog.Debug(ctx, "reading data source", map[string]interface{}{"data_source_type": dataSourceType})

	items, err := listFn(ctx, tokenUUID)
	if err != nil {
		resp.Diagnostics.AddError(listErrorSummary, err.Error())
		return nil, false
	}

	return filter.Apply(ctx, items, filters, accessor), true
}
