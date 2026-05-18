package deployment

import (
	"context"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/filter"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/flex"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = (*deploymentsDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*deploymentsDataSource)(nil)
)

type deploymentsDataSource struct {
	client *client.Client
}

type deploymentsDataSourceModel struct {
	ApplicationUUID types.String          `tfsdk:"application_uuid"`
	Deployments     []deploymentItemModel `tfsdk:"deployments"`
	Filters         []filter.Config       `tfsdk:"filter"`
}

type deploymentItemModel struct {
	UUID       types.String `tfsdk:"uuid"`
	Status     types.String `tfsdk:"status"`
	ServerUUID types.String `tfsdk:"server_uuid"`
}

// NewListDataSource returns a new deployments list data source instance.
func NewListDataSource() datasource.DataSource {
	return &deploymentsDataSource{}
}

func (d *deploymentsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_deployments"
}

func (d *deploymentsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists deployments. Optionally filter by application UUID.",
		Attributes: map[string]schema.Attribute{
			"application_uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the application to filter deployments by. If not set, all deployments are returned.",
				Optional:            true,
				Validators:          []validator.String{validate.UUID()},
			},
			"deployments": schema.ListNestedAttribute{
				MarkdownDescription: "The list of deployments.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"uuid": schema.StringAttribute{
							MarkdownDescription: "The UUID of the deployment.",
							Computed:            true,
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
				},
			},
		},
		Blocks: map[string]schema.Block{
			"filter": filter.Block(),
		},
	}
}

func (d *deploymentsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = flex.ConfigureDataSourceClient(req, &resp.Diagnostics)
}

func (d *deploymentsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config deploymentsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "reading data source", map[string]interface{}{"data_source_type": "coolify_deployments"})

	var deployments []client.Deployment
	var err error

	if !config.ApplicationUUID.IsNull() && !config.ApplicationUUID.IsUnknown() {
		deployments, err = d.client.ListApplicationDeployments(ctx, config.ApplicationUUID.ValueString())
	} else {
		deployments, err = d.client.ListDeployments(ctx)
	}

	if err != nil {
		resp.Diagnostics.AddError("Error listing deployments", err.Error())
		return
	}

	deployments = filter.Apply(ctx, deployments, config.Filters, func(d client.Deployment, field string) (string, bool) {
		switch field {
		case "uuid":
			return d.UUID, true
		case "status":
			return d.Status, true
		case "server_uuid":
			return d.ServerUUID, true
		default:
			return "", false
		}
	})

	items := make([]deploymentItemModel, len(deployments))
	for i, dep := range deployments {
		items[i] = deploymentItemModel{
			UUID:       types.StringValue(dep.UUID),
			Status:     types.StringValue(dep.Status),
			ServerUUID: types.StringValue(dep.ServerUUID),
		}
	}
	config.Deployments = items

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
