package scheduledtask

import (
	"context"
	"fmt"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = (*scheduledTaskDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*scheduledTaskDataSource)(nil)
)

type scheduledTaskDataSource struct {
	client *client.Client
}

type scheduledTaskDataSourceModel struct {
	UUID            types.String `tfsdk:"uuid"`
	ApplicationUUID types.String `tfsdk:"application_uuid"`
	ServiceUUID     types.String `tfsdk:"service_uuid"`
	Name            types.String `tfsdk:"name"`
	Command         types.String `tfsdk:"command"`
	Frequency       types.String `tfsdk:"frequency"`
	Enabled         types.Bool   `tfsdk:"enabled"`
}

// NewDataSource returns a new singular scheduled task data source.
func NewDataSource() datasource.DataSource { return &scheduledTaskDataSource{} }

func (d *scheduledTaskDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_scheduled_task"
}

func (d *scheduledTaskDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves a single scheduled task by UUID from a Coolify application or service.",
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the scheduled task.",
				Required:            true,
			},
			"application_uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the application. Exactly one of `application_uuid` or `service_uuid` must be provided.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(
						path.MatchRoot("service_uuid"),
					),
				},
			},
			"service_uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the service. Exactly one of `application_uuid` or `service_uuid` must be provided.",
				Optional:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the scheduled task.",
				Computed:            true,
			},
			"command": schema.StringAttribute{
				MarkdownDescription: "The command to execute.",
				Computed:            true,
			},
			"frequency": schema.StringAttribute{
				MarkdownDescription: "The cron expression.",
				Computed:            true,
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the task is enabled.",
				Computed:            true,
			},
		},
	}
}

func (d *scheduledTaskDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *scheduledTaskDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config scheduledTaskDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var parentType, parentUUID string
	//nolint:gocritic // if-else chain dispatches to different parent types; switch not applicable
	if !config.ApplicationUUID.IsNull() {
		parentType = "applications"
		parentUUID = config.ApplicationUUID.ValueString()
	} else if !config.ServiceUUID.IsNull() {
		parentType = "services"
		parentUUID = config.ServiceUUID.ValueString()
	} else {
		resp.Diagnostics.AddError("Configuration error", "One of application_uuid or service_uuid must be set")
		return
	}

	tasks, err := d.client.ListScheduledTasks(ctx, parentType, parentUUID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading scheduled task", err.Error())
		return
	}

	uuid := config.UUID.ValueString()
	for _, t := range tasks {
		if t.UUID == uuid {
			config.Name = types.StringValue(t.Name)
			config.Command = types.StringValue(t.Command)
			config.Frequency = types.StringValue(t.Frequency)
			config.Enabled = types.BoolValue(t.Enabled)
			resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
			return
		}
	}

	resp.Diagnostics.AddError("Error reading scheduled task",
		fmt.Sprintf("Scheduled task with UUID %q not found", uuid))
}
