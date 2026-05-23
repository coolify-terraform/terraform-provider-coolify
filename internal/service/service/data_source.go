package service

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
	_ datasource.DataSource              = (*serviceDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*serviceDataSource)(nil)
)

type serviceDataSource struct {
	client *client.Client
}

type serviceDataSourceModel struct {
	UUID            types.String `tfsdk:"uuid"`
	Name            types.String `tfsdk:"name"`
	Description     types.String `tfsdk:"description"`
	Type            types.String `tfsdk:"type"`
	ServerUUID      types.String `tfsdk:"server_uuid"`
	ProjectUUID     types.String `tfsdk:"project_uuid"`
	EnvironmentName types.String `tfsdk:"environment_name"`
}

func NewDataSource() datasource.DataSource {
	return &serviceDataSource{}
}

func (d *serviceDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service"
}

func (d *serviceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Use this data source to retrieve information about a Coolify service by UUID.",
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the service.",
				Required:            true,
				Validators:          []validator.String{validate.UUID()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the service.",
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "A description of the service.",
				Computed:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "The type of the service.",
				Computed:            true,
			},
			"server_uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the server the service is deployed on.",
				Computed:            true,
			},
			"project_uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the project this service belongs to.",
				Computed:            true,
			},
			"environment_name": schema.StringAttribute{
				MarkdownDescription: "The environment name.",
				Computed:            true,
			},
		},
	}
}

func (d *serviceDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = flex.ConfigureDataSourceClient(req, &resp.Diagnostics)
}

func (d *serviceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config serviceDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "reading data source", map[string]interface{}{"data_source_type": "coolify_service"})

	svc, err := d.client.GetService(ctx, config.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading service", fmt.Sprintf("service %s: %s", config.UUID.ValueString(), err))
		return
	}

	config.UUID = types.StringValue(svc.UUID)
	config.Name = flex.StringToFramework(svc.Name)
	config.Description = flex.StringToFramework(svc.Description)
	config.Type = flex.StringToFramework(svc.Type)
	config.ServerUUID = flex.StringToFramework(svc.ServerUUID)
	config.ProjectUUID = flex.StringToFramework(svc.ProjectUUID)
	config.EnvironmentName = flex.StringToFramework(svc.EnvironmentName)

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
