package cloudinitscript

import (
	"context"
	"fmt"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = (*cloudInitDS)(nil)
var _ datasource.DataSourceWithConfigure = (*cloudInitDS)(nil)

type cloudInitDS struct{ client *client.Client }

func NewDataSource() datasource.DataSource { return &cloudInitDS{} }

func (d *cloudInitDS) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cloud_init_script"
}

func (d *cloudInitDS) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads a Coolify cloud-init script by UUID. Requires Coolify >= v4.3.0.",
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{Required: true, MarkdownDescription: "Script UUID."},
			"name": schema.StringAttribute{Computed: true, MarkdownDescription: "Script name."},
			"script": schema.StringAttribute{
				Computed:            true,
				Sensitive:           true,
				MarkdownDescription: "Script body when the token can read it.",
			},
		},
	}
}

func (d *cloudInitDS) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = flex.ConfigureDataSourceClient(req, &resp.Diagnostics)
}

func (d *cloudInitDS) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data cloudInitScriptModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	got, err := d.client.GetCloudInitScript(ctx, data.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading cloud-init script", fmt.Sprintf("%s: %s", data.UUID.ValueString(), err))
		return
	}
	data.Name = types.StringValue(got.Name)
	if got.Script != "" {
		data.Script = types.StringValue(got.Script)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
