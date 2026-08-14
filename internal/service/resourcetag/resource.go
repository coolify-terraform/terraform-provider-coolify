package resourcetag

import (
	"context"
	"strings"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = (*resourceTagResource)(nil)
	_ resource.ResourceWithConfigure   = (*resourceTagResource)(nil)
	_ resource.ResourceWithImportState = (*resourceTagResource)(nil)
)

type resourceTagResource struct{ client *client.Client }

type resourceTagModel struct {
	ID           types.String `tfsdk:"id"`
	ResourceType types.String `tfsdk:"resource_type"`
	ResourceUUID types.String `tfsdk:"resource_uuid"`
	TagName      types.String `tfsdk:"tag_name"`
	TagUUID      types.String `tfsdk:"tag_uuid"`
}

func NewResource() resource.Resource { return &resourceTagResource{} }

func (r *resourceTagResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource_tag"
}

func (r *resourceTagResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Attaches a Coolify tag to an application, database, or service. Requires Coolify >= v4.2.0.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Composite id `resource_type:resource_uuid:tag_uuid`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"resource_type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "One of `application`, `database`, or `service`. Changing this forces a new resource.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          []validator.String{stringvalidator.OneOf("application", "database", "service")},
			},
			"resource_uuid": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "UUID of the tagged resource. Changing this forces a new resource.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"tag_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Tag name to attach (must already exist). Changing this forces a new resource.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"tag_uuid": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "UUID of the attached tag.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *resourceTagResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = flex.ConfigureClient(req, &resp.Diagnostics)
}

func (r *resourceTagResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan resourceTagModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	rt, ru, name := plan.ResourceType.ValueString(), plan.ResourceUUID.ValueString(), plan.TagName.ValueString()
	if err := r.client.AttachResourceTag(ctx, rt, ru, name); err != nil {
		resp.Diagnostics.AddError("Error attaching tag", err.Error())
		return
	}
	tags, err := r.client.ListResourceTags(ctx, rt, ru)
	if err != nil {
		resp.Diagnostics.AddError("Error listing resource tags after attach", err.Error())
		return
	}
	var uuid string
	for _, t := range tags {
		if strings.EqualFold(t.Name, name) {
			uuid = t.UUID
			break
		}
	}
	if uuid == "" {
		resp.Diagnostics.AddError("Error attaching tag", "tag was not present on the resource after attach")
		return
	}
	plan.TagUUID = types.StringValue(uuid)
	plan.ID = types.StringValue(rt + ":" + ru + ":" + uuid)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *resourceTagResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state resourceTagModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tags, err := r.client.ListResourceTags(ctx, state.ResourceType.ValueString(), state.ResourceUUID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading resource tags", err.Error())
		return
	}
	found := false
	for _, t := range tags {
		if t.UUID == state.TagUUID.ValueString() || t.Name == state.TagName.ValueString() {
			state.TagUUID = types.StringValue(t.UUID)
			state.TagName = types.StringValue(t.Name)
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

func (r *resourceTagResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan resourceTagModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *resourceTagResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state resourceTagModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DetachResourceTag(ctx, state.ResourceType.ValueString(), state.ResourceUUID.ValueString(), state.TagUUID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error detaching tag", err.Error())
	}
}

func (r *resourceTagResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ":")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		resp.Diagnostics.AddError("Invalid import ID", "Expected resource_type:resource_uuid:tag_uuid")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("resource_type"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("resource_uuid"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("tag_uuid"), parts[2])...)
}
