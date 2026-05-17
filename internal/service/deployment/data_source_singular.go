package deployment

import (
	"context"
	"fmt"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/flex"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = (*deploymentDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*deploymentDataSource)(nil)
)

type deploymentDataSource struct {
	client *client.Client
}

type deploymentDataSourceModel struct {
	UUID       types.String `tfsdk:"uuid"`
	Status     types.String `tfsdk:"status"`
	ServerUUID types.String `tfsdk:"server_uuid"`
}

// NewDataSource returns a new singular deployment data source.
func NewDataSource() datasource.DataSource { return &deploymentDataSource{} }

func (d *deploymentDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_deployment"
}

func (d *deploymentDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves information about a single Coolify deployment by UUID.",
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the deployment.",
				Required:            true,
				Validators:          []validator.String{validate.UUID()},
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "The current status of the deployment.",
				Computed:            true,
			},
			"server_uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the server the deployment ran on.",
				Computed:            true,
			},
		},
	}
}

func (d *deploymentDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Configure Type", fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData))
		return
	}
	d.client = c
}

func (d *deploymentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config deploymentDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "reading data source", map[string]interface{}{"data_source_type": "coolify_deployment"})

	dep, err := d.client.GetDeployment(ctx, config.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading deployment", err.Error())
		return
	}

	config.UUID = types.StringValue(dep.UUID)
	config.Status = flex.StringToFramework(dep.Status)
	config.ServerUUID = flex.StringToFramework(dep.ServerUUID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
