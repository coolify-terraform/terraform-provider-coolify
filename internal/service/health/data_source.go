package health

import (
	"context"
	"fmt"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = (*healthDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*healthDataSource)(nil)
)

type healthDataSource struct {
	client *client.Client
}

type healthDataSourceModel struct {
	Status types.String `tfsdk:"status"`
}

// NewDataSource returns a new health data source.
func NewDataSource() datasource.DataSource { return &healthDataSource{} }

func (d *healthDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_health"
}

func (d *healthDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves the health status of the connected Coolify instance.",
		Attributes: map[string]schema.Attribute{
			"status": schema.StringAttribute{
				MarkdownDescription: "The health status of the Coolify instance.",
				Computed:            true,
			},
		},
	}
}

func (d *healthDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Data Source Configure Type", fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData))
		return
	}
	d.client = c
}

func (d *healthDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	status, err := d.client.GetHealth(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error reading Coolify health", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &healthDataSourceModel{
		Status: types.StringValue(status),
	})...)
}
