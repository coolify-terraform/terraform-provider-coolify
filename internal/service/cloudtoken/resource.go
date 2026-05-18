package cloudtoken

import (
	"context"
	"fmt"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/flex"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = (*cloudTokenResource)(nil)
	_ resource.ResourceWithImportState = (*cloudTokenResource)(nil)
	_ resource.ResourceWithConfigure   = (*cloudTokenResource)(nil)
)

// cloudTokenResource is the resource implementation for a Coolify cloud token.
type cloudTokenResource struct {
	client *client.Client
}

// cloudTokenResourceModel maps the resource schema data.
type cloudTokenResourceModel struct {
	UUID          types.String `tfsdk:"uuid"`
	Name          types.String `tfsdk:"name"`
	CloudProvider types.String `tfsdk:"cloud_provider"`
	Token         types.String `tfsdk:"token"`
}

// NewResource returns a new cloud token resource instance.
func NewResource() resource.Resource {
	return &cloudTokenResource{}
}

func (r *cloudTokenResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cloud_token"
}

func (r *cloudTokenResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Coolify cloud token.",
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the cloud token.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the cloud token.",
				Required:            true,
			},
			"cloud_provider": schema.StringAttribute{
				MarkdownDescription: "The cloud provider type (e.g., `hetzner`, `aws`). Changing this forces a new resource.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"token": schema.StringAttribute{
				MarkdownDescription: "The cloud provider token. Coolify validates this against the cloud provider's API on creation; placeholder values will be rejected.",
				Required:            true,
				Sensitive:           true,
			},
		},
	}
}

func (r *cloudTokenResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = flex.ConfigureClient(req, &resp.Diagnostics)
}

func (r *cloudTokenResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan cloudTokenResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "creating resource", map[string]interface{}{"resource_type": "coolify_cloud_token"})

	input := client.CreateCloudTokenInput{
		Name:     plan.Name.ValueString(),
		Provider: plan.CloudProvider.ValueString(),
		Token:    plan.Token.ValueString(),
	}

	created, err := r.client.CreateCloudToken(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Error creating cloud token", fmt.Sprintf("cloud token %q: %s", plan.Name.ValueString(), err))
		return
	}

	createdUUID := created.UUID
	plan.UUID = types.StringValue(createdUUID)

	// Save partial state so the resource is tracked even if the read-back fails.
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read back the full cloud token to populate all fields.
	ct, err := r.client.GetCloudToken(ctx, createdUUID)
	if err != nil {
		addCreateReadBackError(resp, createdUUID, err)
		return
	}

	plan.UUID = types.StringValue(ct.UUID)
	plan.Name = types.StringValue(ct.Name)
	plan.CloudProvider = types.StringValue(ct.Provider)
	if ct.Token != "" {
		plan.Token = types.StringValue(ct.Token)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	tflog.Debug(ctx, "created resource", map[string]interface{}{"resource_type": "coolify_cloud_token", "uuid": plan.UUID.ValueString()})
}

func (r *cloudTokenResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state cloudTokenResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "reading resource", map[string]interface{}{"resource_type": "coolify_cloud_token", "uuid": state.UUID.ValueString()})

	ct, err := r.client.GetCloudToken(ctx, state.UUID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			// The cloud token was deleted outside of Terraform; remove from state.
			tflog.Debug(ctx, "resource not found, removing from state", map[string]interface{}{"resource_type": "coolify_cloud_token", "uuid": state.UUID.ValueString()})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading cloud token", fmt.Sprintf("Could not read cloud token %s: %s", state.UUID.ValueString(), err))
		return
	}

	state.UUID = types.StringValue(ct.UUID)
	state.Name = types.StringValue(ct.Name)
	state.CloudProvider = types.StringValue(ct.Provider)
	// Preserve token from state if API does not return it (sensitive field).
	if ct.Token != "" {
		state.Token = types.StringValue(ct.Token)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *cloudTokenResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan cloudTokenResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state cloudTokenResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "updating resource", map[string]interface{}{"resource_type": "coolify_cloud_token", "uuid": state.UUID.ValueString()})

	input := client.UpdateCloudTokenInput{
		Name:  flex.StringIfChanged(plan.Name, state.Name),
		Token: flex.StringIfChanged(plan.Token, state.Token),
	}

	_, err := r.client.UpdateCloudToken(ctx, state.UUID.ValueString(), input)
	if err != nil {
		resp.Diagnostics.AddError("Error updating cloud token", fmt.Sprintf("Could not update cloud token %s: %s", state.UUID.ValueString(), err))
		return
	}

	plan.UUID = state.UUID

	// Read back the full cloud token to populate all fields.
	ct, err := r.client.GetCloudToken(ctx, state.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading cloud token after update", fmt.Sprintf("Could not read cloud token %s after update: %s", state.UUID.ValueString(), err))
		return
	}

	plan.UUID = types.StringValue(ct.UUID)
	plan.Name = types.StringValue(ct.Name)
	plan.CloudProvider = types.StringValue(ct.Provider)
	if ct.Token != "" {
		plan.Token = types.StringValue(ct.Token)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *cloudTokenResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state cloudTokenResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "deleting resource", map[string]interface{}{"resource_type": "coolify_cloud_token", "uuid": state.UUID.ValueString()})

	err := r.client.DeleteCloudToken(ctx, state.UUID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			// Already deleted; nothing to do.
			return
		}
		resp.Diagnostics.AddError("Error deleting cloud token", fmt.Sprintf("Could not delete cloud token %s: %s", state.UUID.ValueString(), err))
		return
	}
	tflog.Debug(ctx, "deleted resource", map[string]interface{}{"resource_type": "coolify_cloud_token", "uuid": state.UUID.ValueString()})
}

func addCreateReadBackError(resp *resource.CreateResponse, uuid string, err error) {
	resp.Diagnostics.AddError(
		"Cloud token created but refresh failed",
		fmt.Sprintf(
			"Coolify created cloud token %s, but the provider could not read it back: Could not read cloud token %s after create: %s. The partial Terraform state was saved, so rerun terraform apply or terraform refresh after the API becomes reachable again.",
			uuid,
			uuid,
			err,
		),
	)
}

func (r *cloudTokenResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if err := validate.ImportUUID(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("uuid"), req, resp)
}
