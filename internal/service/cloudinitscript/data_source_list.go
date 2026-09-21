package cloudinitscript

import (
	"context"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = (*cloudInitListDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*cloudInitListDataSource)(nil)

type cloudInitListDataSource struct{ client *client.Client }

type cloudInitListModel struct {
	Scripts []cloudInitScriptModel `tfsdk:"scripts"`
}

func NewListDataSource() datasource.DataSource { return &cloudInitListDataSource{} }

func (d *cloudInitListDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cloud_init_scripts"
}

func (d *cloudInitListDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists Coolify team cloud-init scripts. Requires Coolify >= v4.3.0. The script body is present only when the token can read sensitive values.",
		Attributes: map[string]schema.Attribute{
			"scripts": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"uuid": schema.StringAttribute{Computed: true, MarkdownDescription: "Script UUID."},
						"name": schema.StringAttribute{Computed: true, MarkdownDescription: "Script name."},
						"script": schema.StringAttribute{
							Computed:            true,
							Sensitive:           true,
							MarkdownDescription: "Script body when the token can read it.",
						},
					},
				},
			},
		},
	}
}

func (d *cloudInitListDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = flex.ConfigureDataSourceClient(req, &resp.Diagnostics)
}

func (d *cloudInitListDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	got, err := d.client.ListCloudInitScripts(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error listing cloud-init scripts", err.Error())
		return
	}
	data := cloudInitListModel{Scripts: make([]cloudInitScriptModel, 0, len(got))}
	for i := range got {
		item := cloudInitScriptModel{
			UUID: types.StringValue(got[i].UUID),
			Name: types.StringValue(got[i].Name),
		}
		if got[i].Script != "" {
			item.Script = types.StringValue(got[i].Script)
		}
		data.Scripts = append(data.Scripts, item)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
