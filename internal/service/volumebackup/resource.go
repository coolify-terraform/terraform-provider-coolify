package volumebackup

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework-validators/float64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                   = (*storageBackupResource)(nil)
	_ resource.ResourceWithConfigure      = (*storageBackupResource)(nil)
	_ resource.ResourceWithImportState    = (*storageBackupResource)(nil)
	_ resource.ResourceWithValidateConfig = (*storageBackupResource)(nil)
)

type storageBackupResource struct {
	client *client.Client
}

type storageBackupResourceModel struct {
	UUID                     types.String  `tfsdk:"uuid"`
	ApplicationUUID          types.String  `tfsdk:"application_uuid"`
	ServiceUUID              types.String  `tfsdk:"service_uuid"`
	DatabaseUUID             types.String  `tfsdk:"database_uuid"`
	StorageUUID              types.String  `tfsdk:"storage_uuid"`
	StorageType              types.String  `tfsdk:"storage_type"`
	Frequency                types.String  `tfsdk:"frequency"`
	Enabled                  types.Bool    `tfsdk:"enabled"`
	SaveS3                   types.Bool    `tfsdk:"save_s3"`
	DisableLocalBackup       types.Bool    `tfsdk:"disable_local_backup"`
	StopDuringBackup         types.Bool    `tfsdk:"stop_during_backup"`
	S3StorageUUID            types.String  `tfsdk:"s3_storage_uuid"`
	RetentionAmountLocally   types.Int64   `tfsdk:"retention_amount_locally"`
	RetentionDaysLocally     types.Int64   `tfsdk:"retention_days_locally"`
	RetentionMaxStorageLocal types.Float64 `tfsdk:"retention_max_storage_locally"`
	RetentionAmountS3        types.Int64   `tfsdk:"retention_amount_s3"`
	RetentionDaysS3          types.Int64   `tfsdk:"retention_days_s3"`
	RetentionMaxStorageS3    types.Float64 `tfsdk:"retention_max_storage_s3"`
	Timeout                  types.Int64   `tfsdk:"timeout"`
}

func NewResource() resource.Resource { return &storageBackupResource{} }

func (r *storageBackupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_storage_backup"
}

func (r *storageBackupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Coolify scheduled backup for a persistent volume or directory storage " +
			"attached to an application, database, or service.\n\n" +
			"**Coolify version requirement:** needs `PUT/DELETE .../storages/{storage_uuid}/backups` " +
			"(VolumeBackupsController). That API landed on Coolify branch `v4.x` in " +
			"[coollabsio/coolify#10946](https://github.com/coollabsio/coolify/pull/10946) (merged 2026-07-20). " +
			"It is **not** present in git tag `v4.2.0` or stable CDN `4.1.2`. " +
			"There is no Coolify release tag yet that is known to include it; use a self-built or " +
			"nightly image from `v4.x` after that merge. CDN nightly may still report version `4.2.0` " +
			"even when tip commits are present, so do not treat the version string alone as proof.\n\n" +
			"~> **API note:** Coolify only exposes create/replace (PUT) and delete. There is no GET for the schedule. " +
			"Read verifies the parent storage still exists via list and keeps schedule attributes from state. " +
			"Out-of-band schedule edits may not appear until the next apply.",
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				MarkdownDescription: "UUID of the scheduled volume backup.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"application_uuid": schema.StringAttribute{
				MarkdownDescription: "UUID of the application that owns the storage. Exactly one of `application_uuid`, `service_uuid`, or `database_uuid`. Changing this forces a new resource.",
				Optional:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(path.MatchRoot("service_uuid"), path.MatchRoot("database_uuid")),
					validate.UUID(),
				},
			},
			"service_uuid": schema.StringAttribute{
				MarkdownDescription: "UUID of the service that owns the storage. Exactly one of `application_uuid`, `service_uuid`, or `database_uuid`. Changing this forces a new resource.",
				Optional:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          []validator.String{validate.UUID()},
			},
			"database_uuid": schema.StringAttribute{
				MarkdownDescription: "UUID of the database that owns the storage. Exactly one of `application_uuid`, `service_uuid`, or `database_uuid`. Changing this forces a new resource.",
				Optional:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          []validator.String{validate.UUID()},
			},
			"storage_uuid": schema.StringAttribute{
				MarkdownDescription: "UUID of the persistent volume or directory storage to back up. Changing this forces a new resource.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          []validator.String{validate.UUID()},
			},
			"storage_type": schema.StringAttribute{
				MarkdownDescription: "Storage kind returned by Coolify: `persistent` or `directory`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"frequency": schema.StringAttribute{
				MarkdownDescription: "Cron or Coolify human expression for the schedule (e.g. `0 2 * * *`, `@daily`).",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^(\S+\s+){4}\S+$|^@(annually|yearly|monthly|weekly|daily|hourly)$`),
						"must be a valid cron expression (e.g. \"0 2 * * *\") or @daily/@hourly/@weekly/etc.",
					),
				},
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the schedule is enabled. Defaults to true.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"save_s3": schema.BoolAttribute{
				MarkdownDescription: "Upload backups to S3. When true, `s3_storage_uuid` is required. Defaults to false.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"disable_local_backup": schema.BoolAttribute{
				MarkdownDescription: "Skip local archives. Only valid when `save_s3` is true. Defaults to false.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"stop_during_backup": schema.BoolAttribute{
				MarkdownDescription: "Stop the resource while the backup runs. Defaults to false.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"s3_storage_uuid": schema.StringAttribute{
				MarkdownDescription: "UUID of a usable team S3 storage when `save_s3` is true.",
				Optional:            true,
				Validators:          []validator.String{validate.UUID()},
			},
			"retention_amount_locally": schema.Int64Attribute{
				MarkdownDescription: "Number of local backups to retain. Defaults to 7.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(7),
				Validators:          []validator.Int64{int64validator.Between(0, 10000)},
			},
			"retention_days_locally": schema.Int64Attribute{
				MarkdownDescription: "Days to retain local backups. Defaults to 0 (unlimited by age).",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(0),
				Validators:          []validator.Int64{int64validator.AtLeast(0)},
			},
			"retention_max_storage_locally": schema.Float64Attribute{
				MarkdownDescription: "Max local backup storage (Coolify units). Defaults to 0 (unlimited).",
				Optional:            true,
				Computed:            true,
				Default:             float64default.StaticFloat64(0),
				Validators:          []validator.Float64{float64validator.AtLeast(0)},
			},
			"retention_amount_s3": schema.Int64Attribute{
				MarkdownDescription: "Number of S3 backups to retain. Defaults to 7.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(7),
				Validators:          []validator.Int64{int64validator.Between(0, 10000)},
			},
			"retention_days_s3": schema.Int64Attribute{
				MarkdownDescription: "Days to retain S3 backups. Defaults to 0.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(0),
				Validators:          []validator.Int64{int64validator.AtLeast(0)},
			},
			"retention_max_storage_s3": schema.Float64Attribute{
				MarkdownDescription: "Max S3 backup storage. Defaults to 0.",
				Optional:            true,
				Computed:            true,
				Default:             float64default.StaticFloat64(0),
				Validators:          []validator.Float64{float64validator.AtLeast(0)},
			},
			"timeout": schema.Int64Attribute{
				MarkdownDescription: "Backup timeout in seconds (60-36000). Defaults to 3600.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(3600),
				Validators:          []validator.Int64{int64validator.Between(60, 36000)},
			},
		},
	}
}

func (r *storageBackupResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = flex.ConfigureClient(req, &resp.Diagnostics)
}

func (r *storageBackupResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var cfg storageBackupResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if cfg.DisableLocalBackup.ValueBool() && !cfg.SaveS3.ValueBool() {
		resp.Diagnostics.AddAttributeError(
			path.Root("disable_local_backup"),
			"Invalid backup destinations",
			"disable_local_backup requires save_s3 = true (Coolify rejects local-only disable).",
		)
	}
	if cfg.SaveS3.ValueBool() && (cfg.S3StorageUUID.IsNull() || cfg.S3StorageUUID.ValueString() == "") {
		resp.Diagnostics.AddAttributeError(
			path.Root("s3_storage_uuid"),
			"Missing S3 storage",
			"s3_storage_uuid is required when save_s3 is true.",
		)
	}
}

func resolveParent(m *storageBackupResourceModel) (parentType, parentUUID string, ok bool) {
	if !m.ApplicationUUID.IsNull() && !m.ApplicationUUID.IsUnknown() && m.ApplicationUUID.ValueString() != "" {
		return "applications", m.ApplicationUUID.ValueString(), true
	}
	if !m.ServiceUUID.IsNull() && !m.ServiceUUID.IsUnknown() && m.ServiceUUID.ValueString() != "" {
		return "services", m.ServiceUUID.ValueString(), true
	}
	if !m.DatabaseUUID.IsNull() && !m.DatabaseUUID.IsUnknown() && m.DatabaseUUID.ValueString() != "" {
		return "databases", m.DatabaseUUID.ValueString(), true
	}
	return "", "", false
}

func buildInput(plan storageBackupResourceModel) client.UpsertVolumeBackupInput {
	input := client.UpsertVolumeBackupInput{
		Frequency: plan.Frequency.ValueString(),
	}
	input.Enabled = flex.BoolValueOrNull(plan.Enabled)
	input.SaveS3 = flex.BoolValueOrNull(plan.SaveS3)
	input.DisableLocalBackup = flex.BoolValueOrNull(plan.DisableLocalBackup)
	input.StopDuringBackup = flex.BoolValueOrNull(plan.StopDuringBackup)
	if !plan.S3StorageUUID.IsNull() && !plan.S3StorageUUID.IsUnknown() {
		input.S3StorageUUID = plan.S3StorageUUID.ValueString()
	}
	if !plan.RetentionAmountLocally.IsNull() && !plan.RetentionAmountLocally.IsUnknown() {
		v := plan.RetentionAmountLocally.ValueInt64()
		input.RetentionAmountLocally = &v
	}
	if !plan.RetentionDaysLocally.IsNull() && !plan.RetentionDaysLocally.IsUnknown() {
		v := plan.RetentionDaysLocally.ValueInt64()
		input.RetentionDaysLocally = &v
	}
	if !plan.RetentionMaxStorageLocal.IsNull() && !plan.RetentionMaxStorageLocal.IsUnknown() {
		v := plan.RetentionMaxStorageLocal.ValueFloat64()
		input.RetentionMaxStorageLocal = &v
	}
	if !plan.RetentionAmountS3.IsNull() && !plan.RetentionAmountS3.IsUnknown() {
		v := plan.RetentionAmountS3.ValueInt64()
		input.RetentionAmountS3 = &v
	}
	if !plan.RetentionDaysS3.IsNull() && !plan.RetentionDaysS3.IsUnknown() {
		v := plan.RetentionDaysS3.ValueInt64()
		input.RetentionDaysS3 = &v
	}
	if !plan.RetentionMaxStorageS3.IsNull() && !plan.RetentionMaxStorageS3.IsUnknown() {
		v := plan.RetentionMaxStorageS3.ValueFloat64()
		input.RetentionMaxStorageS3 = &v
	}
	if !plan.Timeout.IsNull() && !plan.Timeout.IsUnknown() {
		v := plan.Timeout.ValueInt64()
		input.Timeout = &v
	}
	return input
}

func flatten(got *client.VolumeBackupSchedule, m *storageBackupResourceModel) {
	m.UUID = types.StringValue(got.UUID)
	m.StorageUUID = types.StringValue(got.StorageUUID)
	m.StorageType = types.StringValue(got.StorageType)
	m.Frequency = types.StringValue(got.Frequency)
	m.Enabled = types.BoolValue(got.Enabled)
	m.SaveS3 = types.BoolValue(got.SaveS3)
	m.DisableLocalBackup = types.BoolValue(got.DisableLocalBackup)
	m.StopDuringBackup = types.BoolValue(got.StopDuringBackup)
	if got.S3StorageUUID != "" {
		m.S3StorageUUID = types.StringValue(got.S3StorageUUID)
	} else {
		m.S3StorageUUID = types.StringNull()
	}
	m.RetentionAmountLocally = types.Int64Value(got.RetentionAmountLocally)
	m.RetentionDaysLocally = types.Int64Value(got.RetentionDaysLocally)
	m.RetentionMaxStorageLocal = types.Float64Value(got.RetentionMaxStorageLocal)
	m.RetentionAmountS3 = types.Int64Value(got.RetentionAmountS3)
	m.RetentionDaysS3 = types.Int64Value(got.RetentionDaysS3)
	m.RetentionMaxStorageS3 = types.Float64Value(got.RetentionMaxStorageS3)
	m.Timeout = types.Int64Value(got.Timeout)
}

func (r *storageBackupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan storageBackupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	parentType, parentUUID, ok := resolveParent(&plan)
	if !ok {
		resp.Diagnostics.AddError("Configuration Error", "One of application_uuid, service_uuid, or database_uuid must be set")
		return
	}
	tflog.Debug(ctx, "creating resource", map[string]interface{}{"resource_type": "coolify_storage_backup"})

	got, err := r.client.UpsertVolumeBackup(ctx, parentType, parentUUID, plan.StorageUUID.ValueString(), buildInput(plan))
	if err != nil {
		resp.Diagnostics.AddError("Error creating storage backup schedule",
			fmt.Sprintf("storage %s on %s %s: %s", plan.StorageUUID.ValueString(), parentType, parentUUID, err))
		return
	}
	flatten(got, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *storageBackupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state storageBackupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	parentType, parentUUID, ok := resolveParent(&state)
	if !ok {
		resp.Diagnostics.AddError("Configuration Error", "One of application_uuid, service_uuid, or database_uuid must be set")
		return
	}
	tflog.Debug(ctx, "reading resource", map[string]interface{}{
		"resource_type": "coolify_storage_backup", "uuid": state.UUID.ValueString(),
	})

	// No GET for schedules: verify the storage still exists on the parent.
	storages, err := r.client.ListStorages(ctx, parentType, parentUUID)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading storage backup schedule",
			fmt.Sprintf("listing storages for %s %s: %s", parentType, parentUUID, err))
		return
	}
	found := false
	for _, s := range storages {
		if s.UUID == state.StorageUUID.ValueString() {
			found = true
			break
		}
	}
	if !found {
		tflog.Debug(ctx, "parent storage missing, removing storage backup from state", map[string]interface{}{
			"storage_uuid": state.StorageUUID.ValueString(),
		})
		resp.State.RemoveResource(ctx)
		return
	}
	// Keep schedule attributes from state (API has no GET).
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *storageBackupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan storageBackupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	parentType, parentUUID, ok := resolveParent(&plan)
	if !ok {
		resp.Diagnostics.AddError("Configuration Error", "One of application_uuid, service_uuid, or database_uuid must be set")
		return
	}
	tflog.Debug(ctx, "updating resource", map[string]interface{}{
		"resource_type": "coolify_storage_backup", "uuid": plan.UUID.ValueString(),
	})

	got, err := r.client.UpsertVolumeBackup(ctx, parentType, parentUUID, plan.StorageUUID.ValueString(), buildInput(plan))
	if err != nil {
		resp.Diagnostics.AddError("Error updating storage backup schedule",
			fmt.Sprintf("storage %s: %s", plan.StorageUUID.ValueString(), err))
		return
	}
	flatten(got, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *storageBackupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state storageBackupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	parentType, parentUUID, ok := resolveParent(&state)
	if !ok {
		return
	}
	tflog.Debug(ctx, "deleting resource", map[string]interface{}{
		"resource_type": "coolify_storage_backup", "uuid": state.UUID.ValueString(),
	})
	if err := r.client.DeleteVolumeBackup(ctx, parentType, parentUUID, state.StorageUUID.ValueString()); err != nil {
		if client.IsNotFound(err) {
			return
		}
		// Coolify returns 404 with message when schedule missing.
		if strings.Contains(err.Error(), "not found") {
			return
		}
		resp.Diagnostics.AddError("Error deleting storage backup schedule",
			fmt.Sprintf("storage %s: %s", state.StorageUUID.ValueString(), err))
	}
}

func (r *storageBackupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Format: application|service|database : parent_uuid : storage_uuid
	parts := strings.SplitN(req.ID, ":", 3)
	if len(parts) != 3 {
		resp.Diagnostics.AddError("Invalid import ID format",
			`Expected "application|service|database:<parent_uuid>:<storage_uuid>"`)
		return
	}
	parentKey := parts[0]
	switch parentKey {
	case "application", "service", "database":
	default:
		resp.Diagnostics.AddError("Invalid import ID",
			fmt.Sprintf("parent type %q must be application, service, or database", parentKey))
		return
	}
	if err := validate.ImportUUID(parts[1]); err != nil {
		resp.Diagnostics.AddError("Invalid import ID", "parent uuid: "+err.Error())
		return
	}
	if err := validate.ImportUUID(parts[2]); err != nil {
		resp.Diagnostics.AddError("Invalid import ID", "storage uuid: "+err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(parentKey+"_uuid"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("storage_uuid"), parts[2])...)
}
