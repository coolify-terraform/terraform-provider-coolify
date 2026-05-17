package scheduledtask

import (
	"context"
	"fmt"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/filter"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = (*scheduledTaskListDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*scheduledTaskListDataSource)(nil)
)

type scheduledTaskListDataSource struct {
	client *client.Client
}

type scheduledTaskListModel struct {
	ApplicationUUID types.String             `tfsdk:"application_uuid"`
	ServiceUUID     types.String             `tfsdk:"service_uuid"`
	Tasks           []scheduledTaskItemModel `tfsdk:"tasks"`
	Filters         []filter.Config          `tfsdk:"filter"`
}

type scheduledTaskItemModel struct {
	UUID      types.String `tfsdk:"uuid"`
	Name      types.String `tfsdk:"name"`
	Command   types.String `tfsdk:"command"`
	Frequency types.String `tfsdk:"frequency"`
	Enabled   types.Bool   `tfsdk:"enabled"`
}

// NewListDataSource returns a new scheduledTaskListDataSource instance.
func NewListDataSource() datasource.DataSource { return &scheduledTaskListDataSource{} }

func (d *scheduledTaskListDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_scheduled_tasks"
}

func (d *scheduledTaskListDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists all scheduled tasks for a Coolify application or service.",
		Attributes: map[string]schema.Attribute{
			"application_uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the application. Exactly one of `application_uuid` or `service_uuid` must be provided.",
				Optional:            true,
				Validators: []validator.String{
					validate.UUID(),
					stringvalidator.ExactlyOneOf(
						path.MatchRoot("service_uuid"),
					),
				},
			},
			"service_uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the service. Exactly one of `application_uuid` or `service_uuid` must be provided.",
				Optional:            true,
				Validators:          []validator.String{validate.UUID()},
			},
			"tasks": schema.ListNestedAttribute{
				MarkdownDescription: "The list of scheduled tasks.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"uuid":      schema.StringAttribute{MarkdownDescription: "The UUID of the scheduled task.", Computed: true},
						"name":      schema.StringAttribute{MarkdownDescription: "The name of the scheduled task.", Computed: true},
						"command":   schema.StringAttribute{MarkdownDescription: "The command to execute.", Computed: true},
						"frequency": schema.StringAttribute{MarkdownDescription: "The cron expression.", Computed: true},
						"enabled":   schema.BoolAttribute{MarkdownDescription: "Whether the task is enabled.", Computed: true},
					},
				},
			},
		},
		Blocks: map[string]schema.Block{
			"filter": filter.Block(),
		},
	}
}

func (d *scheduledTaskListDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *scheduledTaskListDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config scheduledTaskListModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "reading data source", map[string]interface{}{"data_source_type": "coolify_scheduled_tasks"})

	var parentType, parentUUID string
	//nolint:gocritic // if-else chain with different client calls and early return; switch not clearer
	if !config.ApplicationUUID.IsNull() {
		parentType = "applications"
		parentUUID = config.ApplicationUUID.ValueString()
	} else if !config.ServiceUUID.IsNull() {
		parentType = "services"
		parentUUID = config.ServiceUUID.ValueString()
	} else {
		resp.Diagnostics.AddError("Configuration Error", "One of application_uuid or service_uuid must be set")
		return
	}

	tasks, err := d.client.ListScheduledTasks(ctx, parentType, parentUUID)
	if err != nil {
		resp.Diagnostics.AddError("Error listing scheduled tasks", err.Error())
		return
	}

	tasks = filter.Apply(tasks, config.Filters, func(t client.ScheduledTask, field string) (string, bool) {
		switch field {
		case "uuid":
			return t.UUID, true
		case "name":
			return t.Name, true
		case "command":
			return t.Command, true
		case "frequency":
			return t.Frequency, true
		case "enabled":
			return filter.BoolToString(t.Enabled), true
		default:
			return "", false
		}
	})

	items := make([]scheduledTaskItemModel, len(tasks))
	for i, t := range tasks {
		items[i] = scheduledTaskItemModel{
			UUID:      types.StringValue(t.UUID),
			Name:      types.StringValue(t.Name),
			Command:   types.StringValue(t.Command),
			Frequency: types.StringValue(t.Frequency),
			Enabled:   types.BoolValue(t.Enabled),
		}
	}
	config.Tasks = items

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
