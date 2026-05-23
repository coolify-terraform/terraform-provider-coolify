package servervalidate

import (
	"context"
	"fmt"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/validate"
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
	_ resource.Resource              = (*serverValidateResource)(nil)
	_ resource.ResourceWithConfigure = (*serverValidateResource)(nil)
)

type serverValidateResource struct {
	client *client.Client
}

type serverValidateModel struct {
	ServerUUID types.String `tfsdk:"server_uuid"`
	Valid      types.Bool   `tfsdk:"valid"`
	Message    types.String `tfsdk:"message"`
	Triggers   types.Map    `tfsdk:"triggers"`
}

func NewResource() resource.Resource {
	return &serverValidateResource{}
}

func (r *serverValidateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server_validate"
}

func (r *serverValidateResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Triggers SSH connectivity validation on a Coolify server. Use as a dependency gate to ensure servers are reachable before deploying resources. Changing the `triggers` map forces re-validation.",
		Attributes: map[string]schema.Attribute{
			"server_uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the server to validate.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          []validator.String{validate.UUID()},
			},
			"valid": schema.BoolAttribute{
				MarkdownDescription: "Whether the server passed validation.",
				Computed:            true,
			},
			"message": schema.StringAttribute{
				MarkdownDescription: "Validation result message from Coolify.",
				Computed:            true,
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

func (r *serverValidateResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = flex.ConfigureClient(req, &resp.Diagnostics)
}

func (r *serverValidateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan serverValidateModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	uuid := plan.ServerUUID.ValueString()
	tflog.Debug(ctx, "validating server", map[string]interface{}{"resource_type": "coolify_server_validate", "uuid": uuid})

	result, err := r.client.ValidateServer(ctx, uuid)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error validating server",
			fmt.Sprintf("Could not validate server %s: %s", uuid, err),
		)
		return
	}

	plan.Valid = types.BoolValue(result.Valid)
	plan.Message = types.StringValue(result.Message)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *serverValidateResource) Read(_ context.Context, _ resource.ReadRequest, _ *resource.ReadResponse) {
	// Validation is a point-in-time check. Read is a no-op.
}

func (r *serverValidateResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Unexpected update", "coolify_server_validate does not support in-place updates")
}

func (r *serverValidateResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	// Validation results cannot be undone; delete is a no-op.
}
