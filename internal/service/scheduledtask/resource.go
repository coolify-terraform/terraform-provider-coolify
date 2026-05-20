package scheduledtask

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/flex"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &scheduledTaskResource{}
	_ resource.ResourceWithConfigure   = &scheduledTaskResource{}
	_ resource.ResourceWithImportState = &scheduledTaskResource{}
)

// scheduledTaskResource manages a single scheduled task on an application or service.
type scheduledTaskResource struct {
	client *client.Client
}

// scheduledTaskResourceModel maps the resource schema to Go types.
type scheduledTaskResourceModel struct {
	UUID            types.String `tfsdk:"uuid"`
	ApplicationUUID types.String `tfsdk:"application_uuid"`
	ServiceUUID     types.String `tfsdk:"service_uuid"`
	Name            types.String `tfsdk:"name"`
	Command         types.String `tfsdk:"command"`
	Frequency       types.String `tfsdk:"frequency"`
	Enabled         types.Bool   `tfsdk:"enabled"`
}

// NewResource returns a new scheduledTaskResource instance.
func NewResource() resource.Resource {
	return &scheduledTaskResource{}
}

func (r *scheduledTaskResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_scheduled_task"
}

func (r *scheduledTaskResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a scheduled task on a Coolify application or service.\n\n" +
			"~> **Note:** Each instance requires a List API call to read because the Coolify API does not " +
			"provide a singular GET endpoint for scheduled tasks. Large numbers of these resources " +
			"on a single parent resource may cause slower plan/apply times due to this API limitation.",
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the scheduled task.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"application_uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the application to create the scheduled task on. Exactly one of `application_uuid` or `service_uuid` must be provided. Changing this forces a new resource.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(
						path.MatchRoot("service_uuid"),
					),
					validate.UUID(),
				},
			},
			"service_uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the service to create the scheduled task on. Exactly one of `application_uuid` or `service_uuid` must be provided. Changing this forces a new resource.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{validate.UUID()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the scheduled task.",
				Required:            true,
			},
			"command": schema.StringAttribute{
				MarkdownDescription: "The command to execute.",
				Required:            true,
			},
			"frequency": schema.StringAttribute{
				MarkdownDescription: "The cron expression for the schedule (e.g., `*/5 * * * *`).",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^(\S+\s+){4,5}\S+$|^@(annually|yearly|monthly|weekly|daily|hourly)$`),
						"must be a valid cron expression (e.g., \"*/5 * * * *\" or \"@daily\")",
					),
				},
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the scheduled task is enabled (defaults to `true`).",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
		},
	}
}

// parentInfo returns the parentType ("applications" or "services") and the parent UUID.
func (m *scheduledTaskResourceModel) parentInfo() (string, string) {
	if !m.ApplicationUUID.IsNull() && !m.ApplicationUUID.IsUnknown() {
		return "applications", m.ApplicationUUID.ValueString()
	}
	return "services", m.ServiceUUID.ValueString()
}

func (r *scheduledTaskResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = flex.ConfigureClient(req, &resp.Diagnostics)
}

func (r *scheduledTaskResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan scheduledTaskResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "creating resource", map[string]interface{}{"resource_type": "coolify_scheduled_task"})

	parentType, parentUUID := plan.parentInfo()

	input := client.CreateScheduledTaskInput{
		Name:      plan.Name.ValueString(),
		Command:   plan.Command.ValueString(),
		Frequency: plan.Frequency.ValueString(),
		Enabled:   plan.Enabled.ValueBool(),
	}

	taskUUID, err := r.client.CreateScheduledTask(ctx, parentType, parentUUID, input)
	if err != nil {
		resp.Diagnostics.AddError("Error creating scheduled task", fmt.Sprintf("scheduled task on %s: %s", parentUUID, err))
		return
	}

	plan.UUID = types.StringValue(taskUUID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	tflog.Debug(ctx, "created resource", map[string]interface{}{"resource_type": "coolify_scheduled_task", "uuid": taskUUID})
}

func (r *scheduledTaskResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state scheduledTaskResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "reading resource", map[string]interface{}{"resource_type": "coolify_scheduled_task", "uuid": state.UUID.ValueString()})

	parentType, parentUUID := state.parentInfo()

	tasks, err := r.client.ListScheduledTasks(ctx, parentType, parentUUID)
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Debug(ctx, "resource not found, removing from state", map[string]interface{}{"resource_type": "coolify_scheduled_task", "uuid": state.UUID.ValueString()})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading scheduled tasks", fmt.Sprintf("scheduled task %s: %s", state.UUID.ValueString(), err))
		return
	}

	found := false
	for _, t := range tasks {
		if t.UUID == state.UUID.ValueString() {
			state.Name = types.StringValue(t.Name)
			state.Command = types.StringValue(t.Command)
			state.Frequency = types.StringValue(t.Frequency)
			state.Enabled = types.BoolValue(t.Enabled)
			found = true
			break
		}
	}

	if !found {
		tflog.Debug(ctx, "resource not found, removing from state", map[string]interface{}{"resource_type": "coolify_scheduled_task", "uuid": state.UUID.ValueString()})
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *scheduledTaskResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan scheduledTaskResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state scheduledTaskResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "updating resource", map[string]interface{}{"resource_type": "coolify_scheduled_task", "uuid": plan.UUID.ValueString()})

	parentType, parentUUID := plan.parentInfo()

	input := client.UpdateScheduledTaskInput{
		Name:      flex.StringIfChanged(plan.Name, state.Name),
		Command:   flex.StringIfChanged(plan.Command, state.Command),
		Frequency: flex.StringIfChanged(plan.Frequency, state.Frequency),
		Enabled:   flex.BoolIfChanged(plan.Enabled, state.Enabled),
	}

	if err := r.client.UpdateScheduledTask(ctx, parentType, parentUUID, plan.UUID.ValueString(), input); err != nil {
		resp.Diagnostics.AddError("Error updating scheduled task", fmt.Sprintf("scheduled task %s: %s", plan.UUID.ValueString(), err))
		return
	}

	// Write plan values directly to state without a read-back GET.
	// Scheduled tasks use a list endpoint (no singular GET), so read-back
	// would require listing all tasks and filtering. The plan values are
	// authoritative since the PATCH was accepted.
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *scheduledTaskResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state scheduledTaskResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "deleting resource", map[string]interface{}{"resource_type": "coolify_scheduled_task", "uuid": state.UUID.ValueString()})

	parentType, parentUUID := state.parentInfo()

	if err := r.client.DeleteScheduledTask(ctx, parentType, parentUUID, state.UUID.ValueString()); err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting scheduled task", fmt.Sprintf("scheduled task %s: %s", state.UUID.ValueString(), err))
		return
	}
	tflog.Debug(ctx, "deleted resource", map[string]interface{}{"resource_type": "coolify_scheduled_task", "uuid": state.UUID.ValueString()})
}

func (r *scheduledTaskResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, ":", 3)
	if len(parts) != 3 {
		resp.Diagnostics.AddError(
			"Invalid import ID format",
			`Expected "application:{app_uuid}:{task_uuid}" or "service:{svc_uuid}:{task_uuid}".`,
		)
		return
	}

	resourceType := parts[0]
	parentUUID := parts[1]
	taskUUID := parts[2]

	if err := validate.ImportUUID(parentUUID); err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", fmt.Sprintf("parent UUID segment: %s", err))
		return
	}
	if err := validate.ImportUUID(taskUUID); err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", fmt.Sprintf("task UUID segment: %s", err))
		return
	}

	switch resourceType {
	case "application":
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("application_uuid"), parentUUID)...)
	case "service":
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("service_uuid"), parentUUID)...)
	default:
		resp.Diagnostics.AddError(
			"Invalid import ID type",
			fmt.Sprintf("Expected \"application\" or \"service\", got %q.", resourceType),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("uuid"), taskUUID)...)
}
