package cloudtokenvalidate

import (
	"context"
	"fmt"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/flex"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/validate"
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
	_ resource.Resource              = (*cloudTokenValidateResource)(nil)
	_ resource.ResourceWithConfigure = (*cloudTokenValidateResource)(nil)
)

type cloudTokenValidateResource struct {
	client *client.Client
}

type cloudTokenValidateModel struct {
	CloudTokenUUID types.String `tfsdk:"cloud_token_uuid"`
	Triggers       types.Map    `tfsdk:"triggers"`
}

func NewResource() resource.Resource {
	return &cloudTokenValidateResource{}
}

func (r *cloudTokenValidateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cloud_token_validate"
}

func (r *cloudTokenValidateResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Validates a cloud provider token against the cloud provider's API. Use as a dependency gate to ensure tokens are valid before provisioning cloud servers.",
		Attributes: map[string]schema.Attribute{
			"cloud_token_uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the cloud token to validate.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          []validator.String{validate.UUID()},
			},
			"triggers": schema.MapAttribute{
				MarkdownDescription: "An arbitrary map of values that, when changed, forces re-validation.",
				Optional:            true,
				ElementType:         types.StringType,
				PlanModifiers:       []planmodifier.Map{mapplanmodifier.RequiresReplace()},
			},
		},
	}
}

func (r *cloudTokenValidateResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = flex.ConfigureClient(req, &resp.Diagnostics)
}

func (r *cloudTokenValidateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan cloudTokenValidateModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	uuid := plan.CloudTokenUUID.ValueString()
	tflog.Debug(ctx, "validating cloud token", map[string]interface{}{"resource_type": "coolify_cloud_token_validate", "uuid": uuid})

	if err := r.client.ValidateCloudToken(ctx, uuid); err != nil {
		resp.Diagnostics.AddError(
			"Cloud token validation failed",
			fmt.Sprintf("Could not validate cloud token %s: %s", uuid, err),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *cloudTokenValidateResource) Read(_ context.Context, _ resource.ReadRequest, _ *resource.ReadResponse) {
	// Validation is a point-in-time check. Read is a no-op.
}

func (r *cloudTokenValidateResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Unexpected update", "coolify_cloud_token_validate does not support in-place updates")
}

func (r *cloudTokenValidateResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	// Validation results cannot be undone; delete is a no-op.
}
