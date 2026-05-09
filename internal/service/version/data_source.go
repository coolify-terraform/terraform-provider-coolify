package version

import (
	"context"
	"fmt"

	"github.com/SebTardif/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = (*versionDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*versionDataSource)(nil)
)

type versionDataSource struct {
	client *client.Client
}

type versionDataSourceModel struct {
	Version types.String `tfsdk:"version"`
}

func NewDataSource() datasource.DataSource { return &versionDataSource{} }

func (d *versionDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_version"
}

func (d *versionDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves the version of the connected Coolify instance.",
		Attributes: map[string]schema.Attribute{
			"version": schema.StringAttribute{
				MarkdownDescription: "The Coolify instance version string.",
				Computed:            true,
			},
		},
	}
}

func (d *versionDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected type", fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData))
		return
	}
	d.client = c
}

func (d *versionDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	v, err := d.client.GetVersion(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error reading Coolify version", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &versionDataSourceModel{
		Version: types.StringValue(v),
	})...)
}
