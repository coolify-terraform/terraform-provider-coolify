package scheduledtask

import (
	"context"
	"fmt"

	"github.com/SebTardif/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
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
					stringvalidator.ExactlyOneOf(
						path.MatchRoot("service_uuid"),
					),
				},
			},
			"service_uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the service. Exactly one of `application_uuid` or `service_uuid` must be provided.",
				Optional:            true,
			},
			"task_uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the scheduled task.",
				Required:            true,
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
	}
}

func (d *taskExecutionsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *taskExecutionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config taskExecutionsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

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
