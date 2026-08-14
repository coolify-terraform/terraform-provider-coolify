package tag

import (
	"context"
	"fmt"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = (*tagDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*tagDataSource)(nil)

type tagDataSource struct{ client *client.Client }

func NewDataSource() datasource.DataSource { return &tagDataSource{} }

func (d *tagDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tag"
}

func (d *tagDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads a Coolify team tag by UUID. Requires Coolify >= v4.2.0.",
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{Required: true, MarkdownDescription: "Tag UUID."},
			"name": schema.StringAttribute{Computed: true, MarkdownDescription: "Tag name."},
		},
	}
}

func (d *tagDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = flex.ConfigureDataSourceClient(req, &resp.Diagnostics)
}

func (d *tagDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data tagModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	got, err := d.client.GetTag(ctx, data.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading tag", fmt.Sprintf("tag %s: %s", data.UUID.ValueString(), err))
		return
	}
	data.Name = types.StringValue(got.Name)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
