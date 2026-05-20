package backupexecution

import (
	"context"
	"fmt"
	"strings"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/flex"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = (*backupExecutionResource)(nil)
	_ resource.ResourceWithConfigure   = (*backupExecutionResource)(nil)
	_ resource.ResourceWithImportState = (*backupExecutionResource)(nil)
)

type backupExecutionResource struct {
	client *client.Client
}

type backupExecutionModel struct {
	DatabaseUUID  types.String `tfsdk:"database_uuid"`
	BackupUUID    types.String `tfsdk:"backup_uuid"`
	ExecutionUUID types.String `tfsdk:"execution_uuid"`
}

func NewResource() resource.Resource {
	return &backupExecutionResource{}
}

func (r *backupExecutionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_backup_execution"
}

func (r *backupExecutionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the lifecycle of a database backup execution. Create and read are state-only; on destroy, the execution record is deleted from Coolify. Use for automated cleanup of old backup execution logs.",
		Attributes: map[string]schema.Attribute{
			"database_uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the database.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          []validator.String{validate.UUID()},
			},
			"backup_uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the scheduled backup.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          []validator.String{validate.UUID()},
			},
			"execution_uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the backup execution to manage.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          []validator.String{validate.UUID()},
			},
		},
	}
}

func (r *backupExecutionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = flex.ConfigureClient(req, &resp.Diagnostics)
}

func (r *backupExecutionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan backupExecutionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "creating resource", map[string]interface{}{
		"resource_type":  "coolify_backup_execution",
		"execution_uuid": plan.ExecutionUUID.ValueString(),
	})

	// State-only on create. The execution is produced by Coolify's backup
	// scheduler. This resource tracks it so terraform destroy can purge it.
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *backupExecutionResource) Read(_ context.Context, _ resource.ReadRequest, _ *resource.ReadResponse) {
	// No individual GET endpoint for executions. Preserve state.
}

func (r *backupExecutionResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Unexpected update", "coolify_backup_execution does not support in-place updates")
}

func (r *backupExecutionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state backupExecutionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	dbUUID := state.DatabaseUUID.ValueString()
	backupUUID := state.BackupUUID.ValueString()
	execUUID := state.ExecutionUUID.ValueString()

	tflog.Debug(ctx, "deleting backup execution", map[string]interface{}{
		"resource_type":  "coolify_backup_execution",
		"execution_uuid": execUUID,
	})

	if err := r.client.DeleteBackupExecution(ctx, dbUUID, backupUUID, execUUID); err != nil {
		if !client.IsNotFound(err) {
			resp.Diagnostics.AddError(
				"Error deleting backup execution",
				fmt.Sprintf("Could not delete execution %s: %s", execUUID, err),
			)
		}
	}
}

func (r *backupExecutionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: database_uuid:backup_uuid:execution_uuid
	parts := strings.SplitN(req.ID, ":", 3)
	if len(parts) != 3 {
		resp.Diagnostics.AddError("Invalid import ID", "Expected format: database_uuid:backup_uuid:execution_uuid")
		return
	}

	state := backupExecutionModel{
		DatabaseUUID:  types.StringValue(parts[0]),
		BackupUUID:    types.StringValue(parts[1]),
		ExecutionUUID: types.StringValue(parts[2]),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
