package scheduledtask

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/SebTardif/terraform-provider-coolify/internal/client"
	"github.com/SebTardif/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
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
		MarkdownDescription: "Manages a scheduled task on a Coolify application or service.",
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
				MarkdownDescription: "The cron expression for the schedule (e.g. `*/5 * * * *`).",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^(\S+\s+){4,5}\S+$|^@(annually|yearly|monthly|weekly|daily|hourly)$`),
						"must be a valid cron expression (e.g. \"*/5 * * * *\" or \"@daily\")",
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
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData),
		)
		return
	}
	r.client = c
}

func (r *scheduledTaskResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan scheduledTaskResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	parentType, parentUUID := plan.parentInfo()

	input := client.CreateScheduledTaskInput{
		Name:      plan.Name.ValueString(),
		Command:   plan.Command.ValueString(),
		Frequency: plan.Frequency.ValueString(),
	}

	taskUUID, err := r.client.CreateScheduledTask(ctx, parentType, parentUUID, input)
	if err != nil {
		resp.Diagnostics.AddError("Error creating scheduled task", err.Error())
		return
	}

	plan.UUID = types.StringValue(taskUUID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *scheduledTaskResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state scheduledTaskResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	parentType, parentUUID := state.parentInfo()

	tasks, err := r.client.ListScheduledTasks(ctx, parentType, parentUUID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading scheduled tasks", err.Error())
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

	parentType, parentUUID := plan.parentInfo()

	name := plan.Name.ValueString()
	command := plan.Command.ValueString()
	frequency := plan.Frequency.ValueString()
	enabled := plan.Enabled.ValueBool()

	input := client.UpdateScheduledTaskInput{
		Name:      &name,
		Command:   &command,
		Frequency: &frequency,
		Enabled:   &enabled,
	}

	if err := r.client.UpdateScheduledTask(ctx, parentType, parentUUID, plan.UUID.ValueString(), input); err != nil {
		resp.Diagnostics.AddError("Error updating scheduled task", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *scheduledTaskResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state scheduledTaskResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	parentType, parentUUID := state.parentInfo()

	if err := r.client.DeleteScheduledTask(ctx, parentType, parentUUID, state.UUID.ValueString()); err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting scheduled task", err.Error())
		return
	}
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
