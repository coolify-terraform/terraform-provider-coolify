package destination

import (
	"context"
	"fmt"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = (*destinationDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*destinationDataSource)(nil)
)

type destinationDataSource struct{ client *client.Client }

type destinationDataSourceModel struct {
	UUID       types.String `tfsdk:"uuid"`
	ServerUUID types.String `tfsdk:"server_uuid"`
	Name       types.String `tfsdk:"name"`
	Network    types.String `tfsdk:"network"`
	Type       types.String `tfsdk:"type"`
}

func NewDataSource() datasource.DataSource { return &destinationDataSource{} }

func (d *destinationDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_destination"
}

func (d *destinationDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up a Coolify destination by UUID. Requires Coolify >= v4.2.0.",
		Attributes: map[string]schema.Attribute{
			"uuid":        schema.StringAttribute{Required: true, MarkdownDescription: "Destination UUID.", Validators: []validator.String{validate.UUID()}},
			"server_uuid": schema.StringAttribute{Computed: true, MarkdownDescription: "Owning server UUID."},
			"name":        schema.StringAttribute{Computed: true, MarkdownDescription: "Display name."},
			"network":     schema.StringAttribute{Computed: true, MarkdownDescription: "Docker network name."},
			"type":        schema.StringAttribute{Computed: true, MarkdownDescription: "`standalone` or `swarm`."},
		},
	}
}

func (d *destinationDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = flex.ConfigureDataSourceClient(req, &resp.Diagnostics)
}

func (d *destinationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config destinationDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "reading data source", map[string]interface{}{"data_source_type": "coolify_destination"})
	got, err := d.client.GetDestination(ctx, config.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading destination", fmt.Sprintf("Could not read destination %s: %s", config.UUID.ValueString(), err))
		return
	}
	config.UUID = types.StringValue(got.UUID)
	config.ServerUUID = types.StringValue(got.ServerUUID)
	config.Name = flex.StringToFramework(got.Name)
	config.Network = types.StringValue(got.Network)
	config.Type = types.StringValue(got.Type)
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
