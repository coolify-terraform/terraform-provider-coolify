package scheduledtask

import (
	"context"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/filter"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/validate"
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
	UUID                types.String `tfsdk:"uuid"`
	Name                types.String `tfsdk:"name"`
	Command             types.String `tfsdk:"command"`
	Frequency           types.String `tfsdk:"frequency"`
	Enabled             types.Bool   `tfsdk:"enabled"`
	Container           types.String `tfsdk:"container"`
	Timeout             types.Int64  `tfsdk:"timeout"`
	UseShell            types.Bool   `tfsdk:"use_shell"`
	ConsecutiveFailures types.Int64  `tfsdk:"consecutive_failures"`
	DisabledReason      types.String `tfsdk:"disabled_reason"`
	FailingSince        types.String `tfsdk:"failing_since"`
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
						"uuid":                 schema.StringAttribute{MarkdownDescription: "The UUID of the scheduled task.", Computed: true},
						"name":                 schema.StringAttribute{MarkdownDescription: "The name of the scheduled task.", Computed: true},
						"command":              schema.StringAttribute{MarkdownDescription: "The command to execute.", Computed: true},
						"frequency":            schema.StringAttribute{MarkdownDescription: "The cron expression.", Computed: true},
						"enabled":              schema.BoolAttribute{MarkdownDescription: "Whether the task is enabled.", Computed: true},
						"container":            schema.StringAttribute{MarkdownDescription: "Container name where the command runs.", Computed: true},
						"timeout":              schema.Int64Attribute{MarkdownDescription: "Maximum run time in seconds.", Computed: true},
						"use_shell":            schema.BoolAttribute{MarkdownDescription: "Whether the command runs with `sh -c`. Null when the instance is older than v4.7.0.", Computed: true},
						"consecutive_failures": schema.Int64Attribute{MarkdownDescription: "How many runs in a row have failed. Null when Coolify has not reported one.", Computed: true},
						"disabled_reason":      schema.StringAttribute{MarkdownDescription: "Why Coolify disabled the task. Null when the task is not disabled for failures.", Computed: true},
						"failing_since":        schema.StringAttribute{MarkdownDescription: "When the current failure streak started. Null when the task is not in a failure streak.", Computed: true},
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
	d.client = flex.ConfigureDataSourceClient(req, &resp.Diagnostics)
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

	tasks = filter.Apply(ctx, tasks, config.Filters, func(t client.ScheduledTask, field string) (string, bool) {
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
		case "container":
			return t.Container, true
		case "timeout":
			if t.Timeout == nil {
				return "300", true
			}
			return filter.Int64ToString(*t.Timeout), true
		default:
			return "", false
		}
	})

	items := make([]scheduledTaskItemModel, len(tasks))
	for i, t := range tasks {
		timeout := int64(300)
		if t.Timeout != nil {
			timeout = *t.Timeout
		}
		item := scheduledTaskItemModel{
			UUID:      types.StringValue(t.UUID),
			Name:      types.StringValue(t.Name),
			Command:   types.StringValue(t.Command),
			Frequency: types.StringValue(t.Frequency),
			Enabled:   types.BoolValue(t.Enabled),
			Container: types.StringValue(t.Container),
			Timeout:   types.Int64Value(timeout),
		}
		if t.UseShell != nil {
			item.UseShell = types.BoolValue(*t.UseShell)
		} else {
			item.UseShell = types.BoolNull()
		}
		if t.ConsecutiveFailures != nil {
			item.ConsecutiveFailures = types.Int64Value(*t.ConsecutiveFailures)
		} else {
			item.ConsecutiveFailures = types.Int64Null()
		}
		item.DisabledReason = stringOrNull(t.DisabledReason)
		item.FailingSince = stringOrNull(t.FailingSince)
		items[i] = item
	}
	config.Tasks = items

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
