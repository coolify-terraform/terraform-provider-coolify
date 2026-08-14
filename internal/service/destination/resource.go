package destination

import (
	"context"
	"fmt"
	"regexp"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = (*destinationResource)(nil)
	_ resource.ResourceWithConfigure   = (*destinationResource)(nil)
	_ resource.ResourceWithImportState = (*destinationResource)(nil)
)

type destinationResource struct {
	client *client.Client
}

type destinationResourceModel struct {
	UUID       types.String `tfsdk:"uuid"`
	ServerUUID types.String `tfsdk:"server_uuid"`
	Name       types.String `tfsdk:"name"`
	Network    types.String `tfsdk:"network"`
	Type       types.String `tfsdk:"type"`
}

func NewResource() resource.Resource { return &destinationResource{} }

func (r *destinationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_destination"
}

func (r *destinationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Coolify Docker network destination (standalone or swarm). Requires Coolify >= v4.2.0.\n\n" +
			"Destinations map to Coolify standalone/swarm Docker networks used when deploying applications and services.",
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the destination.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"server_uuid": schema.StringAttribute{
				MarkdownDescription: "UUID of the server that owns this destination. Changing this forces a new resource.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          []validator.String{validate.UUID()},
			},
			"network": schema.StringAttribute{
				MarkdownDescription: "Docker network name. Must match `^[a-zA-Z0-9][a-zA-Z0-9._-]*$`. Changing this forces a new resource.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`), "network must start with alphanumeric and contain only alphanumerics, dots, underscores, or hyphens"),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Display name. Defaults to `{server-name}-{network}` when omitted. Coolify accepts in-place rename via PATCH.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "Destination type: `standalone` or `swarm`. Omit to let Coolify match the server's Docker mode. Changing this forces a new resource.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace(), stringplanmodifier.UseStateForUnknown()},
				Validators:          []validator.String{stringvalidator.OneOf("standalone", "swarm")},
			},
		},
	}
}

func (r *destinationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = flex.ConfigureClient(req, &resp.Diagnostics)
}

func (r *destinationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan destinationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "creating resource", map[string]interface{}{"resource_type": "coolify_destination"})

	input := client.CreateDestinationInput{
		Network: plan.Network.ValueString(),
	}
	flex.SetIfKnown(&input.Name, plan.Name)
	// Omit type when unknown/null so Coolify defaults to the server's Docker mode.
	if !plan.Type.IsNull() && !plan.Type.IsUnknown() && plan.Type.ValueString() != "" {
		input.Type = plan.Type.ValueString()
	}

	created, err := r.client.CreateDestination(ctx, plan.ServerUUID.ValueString(), input)
	if err != nil {
		resp.Diagnostics.AddError("Error creating destination",
			fmt.Sprintf("destination network %q on server %s: %s", plan.Network.ValueString(), plan.ServerUUID.ValueString(), err))
		return
	}

	plan.UUID = types.StringValue(created.UUID)
	if plan.Name.IsUnknown() || plan.Name.IsNull() {
		plan.Name = flex.StringToFramework(created.Name)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	got, err := r.client.GetDestination(ctx, created.UUID)
	if err != nil {
		// Detail uses lower-case "destination" for historical wording; summary is title case.
		resp.Diagnostics.AddError(
			flex.CreateReadBackFailedSummary("Destination"),
			flex.CreateReadBackFailedDetail("destination", created.UUID, err),
		)
		return
	}
	flattenDestination(got, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *destinationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state destinationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	got, err := r.client.GetDestination(ctx, state.UUID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading destination", fmt.Sprintf("destination %s: %s", state.UUID.ValueString(), err))
		return
	}
	flattenDestination(got, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *destinationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan destinationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	name := plan.Name.ValueString()
	updated, err := r.client.UpdateDestination(ctx, plan.UUID.ValueString(), client.UpdateDestinationInput{Name: name})
	if err != nil {
		resp.Diagnostics.AddError("Error updating destination", fmt.Sprintf("destination %s: %s", plan.UUID.ValueString(), err))
		return
	}
	flattenDestination(updated, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *destinationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state destinationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteDestination(ctx, state.UUID.ValueString()); err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting destination", fmt.Sprintf("destination %s: %s", state.UUID.ValueString(), err))
	}
}

func (r *destinationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if err := validate.ImportUUID(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("uuid"), req, resp)
}

func flattenDestination(d *client.Destination, m *destinationResourceModel) {
	m.UUID = types.StringValue(d.UUID)
	m.Name = flex.StringToFramework(d.Name)
	m.Network = types.StringValue(d.Network)
	m.Type = types.StringValue(d.Type)
	if d.ServerUUID != "" {
		m.ServerUUID = types.StringValue(d.ServerUUID)
	}
}
