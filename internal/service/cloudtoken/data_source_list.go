package cloudtoken

import (
	"context"
	"fmt"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = (*cloudTokenListDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*cloudTokenListDataSource)(nil)
)

// cloudTokenListDataSource is the data source implementation for listing all Coolify cloud tokens.
type cloudTokenListDataSource struct {
	client *client.Client
}

// cloudTokenListDataSourceModel maps the data source schema data.
type cloudTokenListDataSourceModel struct {
	CloudTokens []cloudTokenItemModel `tfsdk:"cloud_tokens"`
}

// cloudTokenItemModel maps a single cloud token in the list.
type cloudTokenItemModel struct {
	UUID          types.String `tfsdk:"uuid"`
	Name          types.String `tfsdk:"name"`
	CloudProvider types.String `tfsdk:"cloud_provider"`
}

// NewListDataSource returns a new cloud tokens list data source instance.
func NewListDataSource() datasource.DataSource {
	return &cloudTokenListDataSource{}
}

func (d *cloudTokenListDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cloud_tokens"
}

func (d *cloudTokenListDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Use this data source to list all Coolify cloud tokens.",
		Attributes: map[string]schema.Attribute{
			"cloud_tokens": schema.ListNestedAttribute{
				MarkdownDescription: "The list of cloud tokens.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"uuid": schema.StringAttribute{
							MarkdownDescription: "The unique identifier of the cloud token.",
							Computed:            true,
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
				},
			},
		},
	}
}

func (d *cloudTokenListDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *cloudTokenListDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	tokens, err := d.client.ListCloudTokens(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error listing cloud tokens", fmt.Sprintf("Could not list cloud tokens: %s", err))
		return
	}

	var state cloudTokenListDataSourceModel
	for _, t := range tokens {
		item := cloudTokenItemModel{
			UUID:          types.StringValue(t.UUID),
			Name:          types.StringValue(t.Name),
			CloudProvider: types.StringValue(t.Provider),
		}
		state.CloudTokens = append(state.CloudTokens, item)
	}

	if state.CloudTokens == nil {
		state.CloudTokens = []cloudTokenItemModel{}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
