package privatekey

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
	_ datasource.DataSource              = &privateKeyDataSource{}
	_ datasource.DataSourceWithConfigure = &privateKeyDataSource{}
)

type privateKeyDataSource struct {
	client *client.Client
}

type privateKeyDataSourceModel struct {
	UUID         types.String `tfsdk:"uuid"`
	Name         types.String `tfsdk:"name"`
	Description  types.String `tfsdk:"description"`
	PrivateKey   types.String `tfsdk:"private_key"`
	IsGitRelated types.Bool   `tfsdk:"is_git_related"`
}

// NewDataSource returns a new private key data source.
func NewDataSource() datasource.DataSource {
	return &privateKeyDataSource{}
}

func (d *privateKeyDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_private_key"
}

func (d *privateKeyDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves information about a Coolify private key.",
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the private key.",
				Required:            true,
				Validators:          []validator.String{validate.UUID()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the private key.",
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "A description of the private key.",
				Computed:            true,
			},
			"private_key": schema.StringAttribute{
				MarkdownDescription: "The PEM-encoded private key content.",
				Computed:            true,
				Sensitive:           true,
			},
			"is_git_related": schema.BoolAttribute{
				MarkdownDescription: "Whether this key is used for Git operations.",
				Computed:            true,
			},
		},
	}
}

func (d *privateKeyDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *privateKeyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config privateKeyDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	key, err := d.client.GetPrivateKey(ctx, config.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading private key", err.Error())
		return
	}

	config.UUID = types.StringValue(key.UUID)
	config.Name = types.StringValue(key.Name)
	config.Description = types.StringValue(key.Description)
	config.PrivateKey = types.StringValue(key.PrivateKey)
	config.IsGitRelated = types.BoolValue(key.IsGitRelated)

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
