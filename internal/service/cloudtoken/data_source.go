package cloudtoken

import (
	"context"
	"fmt"

	"github.com/SebTardif/terraform-provider-coolify/internal/client"
	"github.com/SebTardif/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = (*cloudTokenDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*cloudTokenDataSource)(nil)
)

// cloudTokenDataSource is the data source implementation for a single Coolify cloud token.
type cloudTokenDataSource struct {
	client *client.Client
}

// cloudTokenDataSourceModel maps the data source schema data.
type cloudTokenDataSourceModel struct {
	UUID          types.String `tfsdk:"uuid"`
	Name          types.String `tfsdk:"name"`
	CloudProvider types.String `tfsdk:"cloud_provider"`
}

// NewDataSource returns a new cloud token data source instance.
func NewDataSource() datasource.DataSource {
	return &cloudTokenDataSource{}
}

func (d *cloudTokenDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cloud_token"
}

func (d *cloudTokenDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Use this data source to retrieve information about a single Coolify cloud token by its UUID.",
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the cloud token to look up.",
				Required:            true,
				Validators:          []validator.String{validate.UUID()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the cloud token.",
				Computed:            true,
			},
			"cloud_provider": schema.StringAttribute{
				MarkdownDescription: "The cloud provider type.",
				Computed:            true,
			},
		},
	}
}

func (d *cloudTokenDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Provider Data",
			"Expected *client.Client, got an unexpected type. Please report this issue to the provider developers.",
		)
		return
	}
	d.client = c
}

func (d *cloudTokenDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config cloudTokenDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ct, err := d.client.GetCloudToken(ctx, config.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Cloud Token", fmt.Sprintf("Could not read cloud token: %s", err))
		return
	}

	config.UUID = types.StringValue(ct.UUID)
	config.Name = types.StringValue(ct.Name)
	config.CloudProvider = types.StringValue(ct.Provider)

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
