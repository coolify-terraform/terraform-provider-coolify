package privatekey

import (
	"context"
	"fmt"

	"github.com/SebTardif/terraform-provider-coolify/internal/client"
	"github.com/SebTardif/terraform-provider-coolify/internal/flex"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = (*privateKeyListDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*privateKeyListDataSource)(nil)
)

// privateKeyListDataSource is the data source implementation for listing all Coolify private keys.
type privateKeyListDataSource struct {
	client *client.Client
}

// privateKeyListDataSourceModel maps the data source schema data.
type privateKeyListDataSourceModel struct {
	PrivateKeys []privateKeyItemModel `tfsdk:"private_keys"`
}

// privateKeyItemModel maps a single private key in the list.
type privateKeyItemModel struct {
	UUID         types.String `tfsdk:"uuid"`
	Name         types.String `tfsdk:"name"`
	Description  types.String `tfsdk:"description"`
	IsGitRelated types.Bool   `tfsdk:"is_git_related"`
}

// NewListDataSource returns a new private keys list data source instance.
func NewListDataSource() datasource.DataSource {
	return &privateKeyListDataSource{}
}

func (d *privateKeyListDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_private_keys"
}

func (d *privateKeyListDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Use this data source to list all Coolify private keys.",
		Attributes: map[string]schema.Attribute{
			"private_keys": schema.ListNestedAttribute{
				MarkdownDescription: "The list of private keys.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"uuid": schema.StringAttribute{
							MarkdownDescription: "The unique identifier of the private key.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "The name of the private key.",
							Computed:            true,
						},
						"description": schema.StringAttribute{
							MarkdownDescription: "A description of the private key.",
							Computed:            true,
						},
						"is_git_related": schema.BoolAttribute{
							MarkdownDescription: "Whether this key is used for Git operations.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *privateKeyListDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData),
		)
		return
	}
	d.client = c
}

func (d *privateKeyListDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	keys, err := d.client.ListPrivateKeys(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error Listing Private Keys", fmt.Sprintf("Could not list private keys: %s", err))
		return
	}

	var state privateKeyListDataSourceModel
	for _, k := range keys {
		item := privateKeyItemModel{
			UUID:         types.StringValue(k.UUID),
			Name:         types.StringValue(k.Name),
			IsGitRelated: types.BoolValue(k.IsGitRelated),
		}
		item.Description = flex.StringToFramework(k.Description)
		state.PrivateKeys = append(state.PrivateKeys, item)
	}

	if state.PrivateKeys == nil {
		state.PrivateKeys = []privateKeyItemModel{}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
