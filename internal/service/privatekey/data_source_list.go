package privatekey

import (
	"context"
	"fmt"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/filter"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
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
	Filters     []filter.Config       `tfsdk:"filter"`
}

// privateKeyItemModel maps a single private key in the list.
type privateKeyItemModel struct {
	UUID         types.String `tfsdk:"uuid"`
	Name         types.String `tfsdk:"name"`
	Description  types.String `tfsdk:"description"`
	PublicKey    types.String `tfsdk:"public_key"`
	Fingerprint  types.String `tfsdk:"fingerprint"`
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
						"public_key": schema.StringAttribute{
							MarkdownDescription: "The public key derived from the private key.",
							Computed:            true,
						},
						"fingerprint": schema.StringAttribute{
							MarkdownDescription: "The fingerprint of the private key.",
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
		Blocks: map[string]schema.Block{
			"filter": filter.Block(),
		},
	}
}

func (d *privateKeyListDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = flex.ConfigureDataSourceClient(req, &resp.Diagnostics)
}

func (d *privateKeyListDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config privateKeyListDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "reading data source", map[string]interface{}{"data_source_type": "coolify_private_keys"})

	keys, err := d.client.ListPrivateKeys(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error listing private keys", fmt.Sprintf("Could not list private keys: %s", err))
		return
	}

	keys = filter.Apply(ctx, keys, config.Filters, func(k client.PrivateKey, field string) (string, bool) {
		switch field {
		case "uuid":
			return k.UUID, true
		case "name":
			return k.Name, true
		case "description":
			return k.Description, true
		case "fingerprint":
			return k.Fingerprint, true
		case "is_git_related":
			return filter.BoolToString(k.IsGitRelated), true
		default:
			return "", false
		}
	})

	var state privateKeyListDataSourceModel
	state.Filters = config.Filters
	for _, k := range keys {
		item := privateKeyItemModel{
			UUID:         types.StringValue(k.UUID),
			Name:         types.StringValue(k.Name),
			IsGitRelated: types.BoolValue(k.IsGitRelated),
		}
		item.Description = flex.StringToFramework(k.Description)
		item.PublicKey = flex.StringToFramework(k.PublicKey)
		item.Fingerprint = flex.StringToFramework(k.Fingerprint)
		state.PrivateKeys = append(state.PrivateKeys, item)
	}

	if state.PrivateKeys == nil {
		state.PrivateKeys = []privateKeyItemModel{}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
