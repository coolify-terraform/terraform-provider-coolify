package cloudtoken

import (
	"context"
	"fmt"

	"github.com/SebTardif/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
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
				MarkdownDescription: "The cloud provider type. Changing this forces a new resource.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"token": schema.StringAttribute{
				MarkdownDescription: "The cloud provider token.",
				Required:            true,
				Sensitive:           true,
			},
		},
	}
}

func (r *cloudTokenResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Provider Data",
			"Expected *client.Client, got an unexpected type. Please report this issue to the provider developers.",
		)
		return
	}
	r.client = c
}

func (r *cloudTokenResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan cloudTokenResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := client.CreateCloudTokenInput{
		Name:     plan.Name.ValueString(),
		Provider: plan.CloudProvider.ValueString(),
		Token:    plan.Token.ValueString(),
	}

	ct, err := r.client.CreateCloudToken(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Cloud Token", fmt.Sprintf("Could not create cloud token: %s", err))
		return
	}

	plan.UUID = types.StringValue(ct.UUID)

	// Read back the full cloud token to populate all fields.
	diags := r.readCloudToken(ctx, ct.UUID, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *cloudTokenResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state cloudTokenResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ct, err := r.client.GetCloudToken(ctx, state.UUID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			// The cloud token was deleted outside of Terraform; remove from state.
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Cloud Token", fmt.Sprintf("Could not read cloud token: %s", err))
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

	name := plan.Name.ValueString()
	token := plan.Token.ValueString()
	input := client.UpdateCloudTokenInput{
		Name:  &name,
		Token: &token,
	}

	_, err := r.client.UpdateCloudToken(ctx, state.UUID.ValueString(), input)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Cloud Token", fmt.Sprintf("Could not update cloud token: %s", err))
		return
	}

	plan.UUID = state.UUID

	// Read back the full cloud token to populate all fields.
	diags := r.readCloudToken(ctx, state.UUID.ValueString(), &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *cloudTokenResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state cloudTokenResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteCloudToken(ctx, state.UUID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			// Already deleted; nothing to do.
			return
		}
		resp.Diagnostics.AddError("Error Deleting Cloud Token", fmt.Sprintf("Could not delete cloud token: %s", err))
	}
}

func (r *cloudTokenResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("uuid"), req, resp)
}

// readCloudToken fetches the cloud token from the API and updates the model in place.
func (r *cloudTokenResource) readCloudToken(ctx context.Context, uuid string, model *cloudTokenResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	ct, err := r.client.GetCloudToken(ctx, uuid)
	if err != nil {
		diags.AddError("Error Reading Cloud Token", fmt.Sprintf("Could not read cloud token after create/update: %s", err))
		return diags
	}

	model.UUID = types.StringValue(ct.UUID)
	model.Name = types.StringValue(ct.Name)
	model.CloudProvider = types.StringValue(ct.Provider)
	// Preserve token from the current model value since the API may not return
	// sensitive fields. Only overwrite if the API actually returned a token.
	if ct.Token != "" {
		model.Token = types.StringValue(ct.Token)
	}

	return diags
}
