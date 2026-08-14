package tag

import (
	"context"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = (*tagListDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*tagListDataSource)(nil)

type tagListDataSource struct{ client *client.Client }

type tagListModel struct {
	Tags []tagModel `tfsdk:"tags"`
}

func NewListDataSource() datasource.DataSource { return &tagListDataSource{} }

func (d *tagListDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tags"
}

func (d *tagListDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists Coolify team tags. Requires Coolify >= v4.2.0.",
		Attributes: map[string]schema.Attribute{
			"tags": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"uuid": schema.StringAttribute{Computed: true},
						"name": schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *tagListDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = flex.ConfigureDataSourceClient(req, &resp.Diagnostics)
}

func (d *tagListDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data tagListModel
	got, err := d.client.ListTags(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error listing tags", err.Error())
		return
	}
	data.Tags = make([]tagModel, 0, len(got))
	for _, t := range got {
		data.Tags = append(data.Tags, tagModel{UUID: types.StringValue(t.UUID), Name: types.StringValue(t.Name)})
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
