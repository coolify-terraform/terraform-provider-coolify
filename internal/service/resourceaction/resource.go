package resourceaction

import (
	"context"
	"fmt"
	"strings"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/flex"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource              = (*resourceActionResource)(nil)
	_ resource.ResourceWithConfigure = (*resourceActionResource)(nil)
)

type resourceActionResource struct {
	client *client.Client
}

type resourceActionModel struct {
	ResourceUUID types.String `tfsdk:"resource_uuid"`
	ResourceType types.String `tfsdk:"resource_type"`
	Action       types.String `tfsdk:"action"`
	Triggers     types.Map    `tfsdk:"triggers"`
}

func NewResource() resource.Resource {
	return &resourceActionResource{}
}

func (r *resourceActionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource_action"
}

func (r *resourceActionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Triggers a start, stop, or restart action on a Coolify application, database, or service. Changing the `triggers` map forces the action to run again.",
		Attributes: map[string]schema.Attribute{
			"resource_uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the target resource (application, database, or service).",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          []validator.String{validate.UUID()},
			},
			"resource_type": schema.StringAttribute{
				MarkdownDescription: "The type of the target resource. Must be one of: `application`, `database`, `service`.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators: []validator.String{
					stringvalidator.OneOf("application", "database", "service"),
				},
			},
			"action": schema.StringAttribute{
				MarkdownDescription: "The action to perform. Must be one of: `start`, `stop`, `restart`.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators: []validator.String{
					stringvalidator.OneOf("start", "stop", "restart"),
				},
			},
			"triggers": schema.MapAttribute{
				MarkdownDescription: "An arbitrary map of values that, when changed, forces the action to run again.",
				Optional:            true,
				ElementType:         types.StringType,
				PlanModifiers:       []planmodifier.Map{mapplanmodifier.RequiresReplace()},
			},
		},
	}
}

func (r *resourceActionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = flex.ConfigureClient(req, &resp.Diagnostics)
}

func (r *resourceActionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan resourceActionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	uuid := plan.ResourceUUID.ValueString()
	resType := plan.ResourceType.ValueString()
	action := plan.Action.ValueString()

	tflog.Debug(ctx, "executing resource action", map[string]interface{}{
		"resource_type": "coolify_resource_action",
		"target_type":   resType,
		"action":        action,
		"uuid":          uuid,
	})

	if err := r.executeAction(ctx, resType, action, uuid); err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error performing %s on %s", action, resType),
			fmt.Sprintf("Could not %s %s %s: %s", action, resType, uuid, err),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *resourceActionResource) Read(_ context.Context, _ resource.ReadRequest, _ *resource.ReadResponse) {
	// Resource actions are fire-and-forget. Read is a no-op; the state
	// persists what was last executed.
}

func (r *resourceActionResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	// All mutable attributes use RequiresReplace, so Update is never called.
	resp.Diagnostics.AddError("Unexpected update", "coolify_resource_action does not support in-place updates")
}

func (r *resourceActionResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	// Actions cannot be undone; delete is a no-op.
}

func (r *resourceActionResource) executeAction(ctx context.Context, resType, action, uuid string) error {
	var err error
	switch resType + ":" + action {
	case "application:start":
		err = r.client.StartApplication(ctx, uuid)
	case "application:stop":
		err = r.client.StopApplication(ctx, uuid)
	case "application:restart":
		_, err = r.client.RestartApplication(ctx, uuid)
	case "database:start":
		err = r.client.StartDatabase(ctx, uuid)
	case "database:stop":
		err = r.client.StopDatabase(ctx, uuid)
	case "database:restart":
		err = r.client.RestartDatabase(ctx, uuid)
	case "service:start":
		err = r.client.StartService(ctx, uuid)
	case "service:stop":
		err = r.client.StopService(ctx, uuid)
	case "service:restart":
		err = r.client.RestartService(ctx, uuid)
	default:
		return fmt.Errorf("unsupported combination: %s/%s", resType, action)
	}

	if err != nil && isAlreadyInDesiredState(err, action) {
		tflog.Debug(ctx, "resource already in desired state, treating as success", map[string]interface{}{
			"action": action,
			"uuid":   uuid,
		})
		return nil
	}
	return err
}

// isAlreadyInDesiredState returns true when Coolify reports a 400 because
// the resource is already in the target state (e.g. "Database is already
// stopped." when action is "stop"). These are idempotent successes.
func isAlreadyInDesiredState(err error, action string) bool {
	msg := err.Error()
	switch action {
	case "start":
		return strings.Contains(msg, "already running")
	case "stop":
		return strings.Contains(msg, "already stopped")
	default:
		return false
	}
}
