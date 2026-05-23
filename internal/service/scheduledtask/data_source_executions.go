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
	_ datasource.DataSource              = (*taskExecutionsDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*taskExecutionsDataSource)(nil)
)

type taskExecutionsDataSource struct {
	client *client.Client
}

type taskExecutionsDataSourceModel struct {
	ApplicationUUID types.String         `tfsdk:"application_uuid"`
	ServiceUUID     types.String         `tfsdk:"service_uuid"`
	TaskUUID        types.String         `tfsdk:"task_uuid"`
	Executions      []taskExecutionModel `tfsdk:"executions"`
	Filters         []filter.Config      `tfsdk:"filter"`
}

type taskExecutionModel struct {
	UUID      types.String `tfsdk:"uuid"`
	Status    types.String `tfsdk:"status"`
	Message   types.String `tfsdk:"message"`
	CreatedAt types.String `tfsdk:"created_at"`
}

// NewExecutionsDataSource returns a new task executions data source instance.
func NewExecutionsDataSource() datasource.DataSource { return &taskExecutionsDataSource{} }

func (d *taskExecutionsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_task_executions"
}

func (d *taskExecutionsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists all executions for a scheduled task on an application or service.",
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
			"task_uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the scheduled task.",
				Required:            true,
				Validators:          []validator.String{validate.UUID()},
			},
			"executions": schema.ListNestedAttribute{
				MarkdownDescription: "The list of task executions.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"uuid":       schema.StringAttribute{MarkdownDescription: "The UUID of the execution.", Computed: true},
						"status":     schema.StringAttribute{MarkdownDescription: "The status of the execution.", Computed: true},
						"message":    schema.StringAttribute{MarkdownDescription: "The output message of the execution.", Computed: true},
						"created_at": schema.StringAttribute{MarkdownDescription: "The creation timestamp.", Computed: true},
					},
				},
			},
		},
		Blocks: map[string]schema.Block{
			"filter": filter.Block(),
		},
	}
}

func (d *taskExecutionsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = flex.ConfigureDataSourceClient(req, &resp.Diagnostics)
}

func (d *taskExecutionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config taskExecutionsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "reading data source", map[string]interface{}{"data_source_type": "coolify_task_executions"})

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

	execs, err := d.client.ListTaskExecutions(ctx, parentType, parentUUID, config.TaskUUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error listing task executions", err.Error())
		return
	}

	execs = filter.Apply(ctx, execs, config.Filters, func(e client.TaskExecution, field string) (string, bool) {
		switch field {
		case "uuid":
			return e.UUID, true
		case "status":
			return e.Status, true
		case "message":
			return e.Message, true
		case "created_at":
			return e.CreatedAt, true
		default:
			return "", false
		}
	})

	items := make([]taskExecutionModel, len(execs))
	for i, e := range execs {
		items[i] = taskExecutionModel{
			UUID:      types.StringValue(e.UUID),
			Status:    types.StringValue(e.Status),
			Message:   types.StringValue(e.Message),
			CreatedAt: types.StringValue(e.CreatedAt),
		}
	}
	config.Executions = items

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
