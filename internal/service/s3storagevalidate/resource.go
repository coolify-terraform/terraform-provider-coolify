package s3storagevalidate

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
	_ resource.Resource              = (*s3StorageValidateResource)(nil)
	_ resource.ResourceWithConfigure = (*s3StorageValidateResource)(nil)
)

type s3StorageValidateResource struct {
	client *client.Client
}

type s3StorageValidateModel struct {
	S3StorageUUID types.String `tfsdk:"s3_storage_uuid"`
	Triggers      types.Map    `tfsdk:"triggers"`
}

func NewResource() resource.Resource {
	return &s3StorageValidateResource{}
}

func (r *s3StorageValidateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_s3_storage_validate"
}

func (r *s3StorageValidateResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Validates an S3-compatible storage configuration against the remote endpoint " +
			"(POST `/api/v1/s3-storages/{uuid}/validate`). Use after credential rotation or as a dependency " +
			"gate before configuring backups. Requires Coolify >= v4.3.0. Changing the `triggers` map forces re-validation.",
		Attributes: map[string]schema.Attribute{
			"s3_storage_uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the S3 storage configuration to validate.",
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

func (r *s3StorageValidateResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = flex.ConfigureClient(req, &resp.Diagnostics)
}

func (r *s3StorageValidateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan s3StorageValidateModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	uuid := plan.S3StorageUUID.ValueString()
	tflog.Debug(ctx, "validating s3 storage", map[string]interface{}{"resource_type": "coolify_s3_storage_validate", "uuid": uuid})

	if err := r.client.ValidateS3Storage(ctx, uuid); err != nil {
		resp.Diagnostics.AddError(
			"S3 storage validation failed",
			fmt.Sprintf("Could not validate s3 storage %s: %s", uuid, err),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *s3StorageValidateResource) Read(_ context.Context, _ resource.ReadRequest, _ *resource.ReadResponse) {
	// Validation is a point-in-time check. Read is a no-op.
}

func (r *s3StorageValidateResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Unexpected update", "coolify_s3_storage_validate does not support in-place updates")
}

func (r *s3StorageValidateResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	// Validation results cannot be undone; delete is a no-op.
}
