package volumebackup

import (
	"context"
	"fmt"
	"strings"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework-validators/float64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
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
	UUID                          types.String  `tfsdk:"uuid"`
	ApplicationUUID               types.String  `tfsdk:"application_uuid"`
	ServiceUUID                   types.String  `tfsdk:"service_uuid"`
	DatabaseUUID                  types.String  `tfsdk:"database_uuid"`
	StorageUUID                   types.String  `tfsdk:"storage_uuid"`
	StorageType                   types.String  `tfsdk:"storage_type"`
	Frequency                     types.String  `tfsdk:"frequency"`
	Enabled                       types.Bool    `tfsdk:"enabled"`
	SaveS3                        types.Bool    `tfsdk:"save_s3"`
	DisableLocalBackup            types.Bool    `tfsdk:"disable_local_backup"`
	StopDuringBackup              types.Bool    `tfsdk:"stop_during_backup"`
	S3StorageUUID                 types.String  `tfsdk:"s3_storage_uuid"`
	RetentionAmountLocally        types.Int64   `tfsdk:"retention_amount_locally"`
	RetentionDaysLocally          types.Int64   `tfsdk:"retention_days_locally"`
	RetentionMaxStorageLocal      types.Float64 `tfsdk:"retention_max_storage_locally"`
	RetentionAmountS3             types.Int64   `tfsdk:"retention_amount_s3"`
	RetentionDaysS3               types.Int64   `tfsdk:"retention_days_s3"`
	RetentionMaxStorageS3         types.Float64 `tfsdk:"retention_max_storage_s3"`
	Timeout                       types.Int64   `tfsdk:"timeout"`
	MissingBackupNotificationDays types.Int64   `tfsdk:"missing_backup_notification_days"`
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
			"(VolumeBackupsController). That API landed in " +
			"[coollabsio/coolify#10946](https://github.com/coollabsio/coolify/pull/10946) and ships in " +
			"**Coolify >= v4.3.0** (stable CDN). It is **not** present in git tag `v4.2.0` or older stable lines.\n\n" +
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
				MarkdownDescription: "Cron or Coolify human expression for the schedule (e.g. `0 2 * * *`, `daily`, `@daily`, `hourly`). Coolify also accepts `every_minute`, `weekly`, `monthly`, and `yearly` without `@`.",
				Required:            true,
				Validators:          []validator.String{validate.CoolifyFrequency()},
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the schedule is enabled. Create uses true when this is omitted. After import, set it before the next apply. Coolify replaces an omitted value.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"save_s3": schema.BoolAttribute{
				MarkdownDescription: "Upload backups to S3. When true, `s3_storage_uuid` is required. Create uses false when this is omitted. After import, set it before the next apply. Coolify replaces an omitted value.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"disable_local_backup": schema.BoolAttribute{
				MarkdownDescription: "Skip local archives. Only valid when `save_s3` is true. Create uses false when this is omitted. After import, set it before the next apply. Coolify replaces an omitted value.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"stop_during_backup": schema.BoolAttribute{
				MarkdownDescription: "Stop the resource while the backup runs. Create uses false when this is omitted. After import, set it before the next apply. Coolify replaces an omitted value.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"s3_storage_uuid": schema.StringAttribute{
				MarkdownDescription: "UUID of a usable team S3 storage when `save_s3` is true.",
				Optional:            true,
				Validators:          []validator.String{validate.UUID()},
			},
			"retention_amount_locally": schema.Int64Attribute{
				MarkdownDescription: "Number of local backups to retain. Create uses 7 when this is omitted. After import, set it before the next apply. Coolify replaces an omitted value.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
				Validators:          []validator.Int64{int64validator.Between(0, 10000)},
			},
			"retention_days_locally": schema.Int64Attribute{
				MarkdownDescription: "Days to retain local backups. Create uses 0 (unlimited by age) when this is omitted. After import, set it before the next apply. Coolify replaces an omitted value.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
				Validators:          []validator.Int64{int64validator.AtLeast(0)},
			},
			"retention_max_storage_locally": schema.Float64Attribute{
				MarkdownDescription: "Max local backup storage (Coolify units). Create uses 0 (unlimited) when this is omitted. After import, set it before the next apply. Coolify replaces an omitted value.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Float64{float64planmodifier.UseStateForUnknown()},
				Validators:          []validator.Float64{float64validator.AtLeast(0)},
			},
			"retention_amount_s3": schema.Int64Attribute{
				MarkdownDescription: "Number of S3 backups to retain. Create uses 7 when this is omitted. After import, set it before the next apply. Coolify replaces an omitted value.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
				Validators:          []validator.Int64{int64validator.Between(0, 10000)},
			},
			"retention_days_s3": schema.Int64Attribute{
				MarkdownDescription: "Days to retain S3 backups. Create uses 0 when this is omitted. After import, set it before the next apply. Coolify replaces an omitted value.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
				Validators:          []validator.Int64{int64validator.AtLeast(0)},
			},
			"retention_max_storage_s3": schema.Float64Attribute{
				MarkdownDescription: "Max S3 backup storage. Create uses 0 when this is omitted. After import, set it before the next apply. Coolify replaces an omitted value.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Float64{float64planmodifier.UseStateForUnknown()},
				Validators:          []validator.Float64{float64validator.AtLeast(0)},
			},
			"timeout": schema.Int64Attribute{
				MarkdownDescription: "Backup timeout in seconds (60-36000). When this is omitted, Coolify stores its column default (3600 on v4.4.2, 36000 after the 2026-08-15 migration) and the provider keeps the value from the create response. Import does not require it, because Coolify keeps the stored timeout when the key is absent.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
				Validators:          []validator.Int64{int64validator.Between(60, 36000)},
			},
			"missing_backup_notification_days": schema.Int64Attribute{
				MarkdownDescription: "Days without a backup execution before Coolify sends a missing-backup notification. " +
					"`0` disables alerts. Valid range is 0-365. " +
					"Requires Coolify >= v4.4.1. Tag v4.4.0 and `v4.4-rc.1` reject the key. " +
					"On older instances the provider keeps the value in state and does not send it.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
				Validators:    []validator.Int64{int64validator.Between(0, 365)},
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
	// Unknown attributes (vars, references) must not be treated as false/empty.
	// ValueBool() on unknown is false, which falsely triggers these checks.
	if cfg.DisableLocalBackup.IsUnknown() || cfg.SaveS3.IsUnknown() || cfg.S3StorageUUID.IsUnknown() {
		return
	}
	if !cfg.DisableLocalBackup.IsNull() && cfg.DisableLocalBackup.ValueBool() &&
		(cfg.SaveS3.IsNull() || !cfg.SaveS3.ValueBool()) {
		resp.Diagnostics.AddAttributeError(
			path.Root("disable_local_backup"),
			"Invalid backup destinations",
			"disable_local_backup requires save_s3 = true (Coolify rejects local-only disable).",
		)
	}
	if !cfg.SaveS3.IsNull() && cfg.SaveS3.ValueBool() &&
		(cfg.S3StorageUUID.IsNull() || cfg.S3StorageUUID.ValueString() == "") {
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

func scheduleWhere(storageUUID, parentType, parentUUID string) string {
	switch {
	case storageUUID != "" && parentUUID != "":
		return fmt.Sprintf("storage %s on %s %s", storageUUID, parentType, parentUUID)
	case storageUUID != "":
		return "storage " + storageUUID
	default:
		return "storage backup"
	}
}

func buildInput(c *client.Client, plan storageBackupResourceModel) client.UpsertVolumeBackupInput {
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
	if c != nil && c.SupportsVolumeBackupMissingNotificationDays() &&
		!plan.MissingBackupNotificationDays.IsNull() && !plan.MissingBackupNotificationDays.IsUnknown() {
		v := plan.MissingBackupNotificationDays.ValueInt64()
		input.MissingBackupNotificationDays = &v
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
	if got.MissingBackupNotificationDays != nil {
		m.MissingBackupNotificationDays = types.Int64Value(*got.MissingBackupNotificationDays)
	} else if m.MissingBackupNotificationDays.IsUnknown() {
		m.MissingBackupNotificationDays = types.Int64Null()
	}
}

func warnUnsupportedVolumeBackupMissingDays(c *client.Client, plan storageBackupResourceModel, diags *diag.Diagnostics) {
	if diags == nil || c == nil || c.SupportsVolumeBackupMissingNotificationDays() {
		return
	}
	if plan.MissingBackupNotificationDays.IsNull() || plan.MissingBackupNotificationDays.IsUnknown() {
		return
	}
	ver := c.CoolifyVersion
	if ver == "" {
		ver = "unknown"
	}
	diags.AddWarning(
		"Coolify version cannot write missing_backup_notification_days",
		fmt.Sprintf(
			"This Coolify instance (%s) does not accept missing_backup_notification_days on volume backup schedules (added in Coolify v4.4.1). "+
				"The provider will keep the value in Terraform state but will not send it. "+
				"Upgrade to Coolify v4.4.1 or later, or remove missing_backup_notification_days from configuration.",
			ver,
		),
	)
}

func (r *storageBackupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan storageBackupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	parentType, parentUUID, ok := resolveParent(&plan)
	if !ok {
		resp.Diagnostics.AddError("Configuration Error",
			fmt.Sprintf("%s: one of application_uuid, service_uuid, or database_uuid must be set",
				scheduleWhere(plan.StorageUUID.ValueString(), "", "")))
		return
	}
	tflog.Debug(ctx, "creating resource", map[string]interface{}{"resource_type": "coolify_storage_backup"})
	warnUnsupportedVolumeBackupMissingDays(r.client, plan, &resp.Diagnostics)
	fillCreateDefaults(&plan)

	got, err := r.client.UpsertVolumeBackup(ctx, parentType, parentUUID, plan.StorageUUID.ValueString(), buildInput(r.client, plan))
	if err != nil {
		resp.Diagnostics.AddError("Error creating storage backup schedule",
			fmt.Sprintf("%s: %s", scheduleWhere(plan.StorageUUID.ValueString(), parentType, parentUUID), err))
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
		resp.Diagnostics.AddError("Configuration Error",
			fmt.Sprintf("%s: one of application_uuid, service_uuid, or database_uuid must be set",
				scheduleWhere(state.StorageUUID.ValueString(), "", "")))
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
		resp.Diagnostics.AddError("Configuration Error",
			fmt.Sprintf("%s: one of application_uuid, service_uuid, or database_uuid must be set",
				scheduleWhere(plan.StorageUUID.ValueString(), "", "")))
		return
	}
	tflog.Debug(ctx, "updating resource", map[string]interface{}{
		"resource_type": "coolify_storage_backup", "uuid": plan.UUID.ValueString(),
	})

	warnUnsupportedVolumeBackupMissingDays(r.client, plan, &resp.Diagnostics)
	if missing := unsetDestructiveFields(plan); len(missing) > 0 {
		resp.Diagnostics.AddError(
			"Storage backup schedule is incomplete",
			fmt.Sprintf("%s. Coolify replaces omitted enabled, save_s3, disable_local_backup, stop_during_backup, and retention fields. "+
				"Set %s before apply. Import does not know the live values. "+
				"timeout and missing_backup_notification_days can stay omitted.",
				scheduleWhere(plan.StorageUUID.ValueString(), parentType, parentUUID),
				strings.Join(missing, ", ")),
		)
		return
	}
	got, err := r.client.UpsertVolumeBackup(ctx, parentType, parentUUID, plan.StorageUUID.ValueString(), buildInput(r.client, plan))
	if err != nil {
		resp.Diagnostics.AddError("Error updating storage backup schedule",
			fmt.Sprintf("%s: %s", scheduleWhere(plan.StorageUUID.ValueString(), parentType, parentUUID), err))
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
		resp.Diagnostics.AddError("Error deleting storage backup schedule",
			fmt.Sprintf("%s: one of application_uuid, service_uuid, or database_uuid must be set",
				scheduleWhere(state.StorageUUID.ValueString(), "", "")))
		return
	}
	tflog.Debug(ctx, "deleting resource", map[string]interface{}{
		"resource_type": "coolify_storage_backup", "uuid": state.UUID.ValueString(),
	})
	if err := r.client.DeleteVolumeBackup(ctx, parentType, parentUUID, state.StorageUUID.ValueString()); err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting storage backup schedule",
			fmt.Sprintf("%s: %s", scheduleWhere(state.StorageUUID.ValueString(), parentType, parentUUID), err))
	}
}

func (r *storageBackupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Format: application|service|database : parent_uuid : storage_uuid
	parts := strings.SplitN(req.ID, ":", 3)
	if len(parts) != 3 {
		resp.Diagnostics.AddError("Invalid import ID format",
			fmt.Sprintf("Import ID %q. Expected \"application|service|database:<parent_uuid>:<storage_uuid>\"", req.ID))
		return
	}
	parentKey := parts[0]
	switch parentKey {
	case "application", "service", "database":
	default:
		resp.Diagnostics.AddError("Invalid import ID",
			fmt.Sprintf("parent type %q must be application, service, or database (import ID %q)", parentKey, req.ID))
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
	resp.Diagnostics.AddWarning(
		"Set schedule fields before the next apply",
		fmt.Sprintf("Imported %s. Coolify has no GET for this schedule and replaces omitted enabled and retention fields. "+
			"Set enabled, save_s3, disable_local_backup, stop_during_backup, and every retention attribute before the next apply. "+
			"timeout and missing_backup_notification_days can stay omitted.",
			scheduleWhere(parts[2], parentKey, parts[1])),
	)
}

// fillCreateDefaults supplies the values Coolify uses when a create omits
// them. Schema defaults cannot do this: Default runs before plan modifiers,
// so an import would plan those defaults and overwrite the live schedule.
func fillCreateDefaults(plan *storageBackupResourceModel) {
	setBoolDefault(&plan.Enabled, true)
	setBoolDefault(&plan.SaveS3, false)
	setBoolDefault(&plan.DisableLocalBackup, false)
	setBoolDefault(&plan.StopDuringBackup, false)
	setIntDefault(&plan.RetentionAmountLocally, 7)
	setIntDefault(&plan.RetentionDaysLocally, 0)
	setFloatDefault(&plan.RetentionMaxStorageLocal, 0)
	setIntDefault(&plan.RetentionAmountS3, 7)
	setIntDefault(&plan.RetentionDaysS3, 0)
	setFloatDefault(&plan.RetentionMaxStorageS3, 0)
}

func setBoolDefault(v *types.Bool, def bool) {
	if v.IsNull() || v.IsUnknown() {
		*v = types.BoolValue(def)
	}
}

func setIntDefault(v *types.Int64, def int64) {
	if v.IsNull() || v.IsUnknown() {
		*v = types.Int64Value(def)
	}
}

func setFloatDefault(v *types.Float64, def float64) {
	if v.IsNull() || v.IsUnknown() {
		*v = types.Float64Value(def)
	}
}

func unsetDestructiveFields(plan storageBackupResourceModel) []string {
	var missing []string
	checkBool := func(name string, v types.Bool) {
		if v.IsNull() || v.IsUnknown() {
			missing = append(missing, name)
		}
	}
	checkInt := func(name string, v types.Int64) {
		if v.IsNull() || v.IsUnknown() {
			missing = append(missing, name)
		}
	}
	checkFloat := func(name string, v types.Float64) {
		if v.IsNull() || v.IsUnknown() {
			missing = append(missing, name)
		}
	}
	checkBool("enabled", plan.Enabled)
	checkBool("save_s3", plan.SaveS3)
	checkBool("disable_local_backup", plan.DisableLocalBackup)
	checkBool("stop_during_backup", plan.StopDuringBackup)
	checkInt("retention_amount_locally", plan.RetentionAmountLocally)
	checkInt("retention_days_locally", plan.RetentionDaysLocally)
	checkFloat("retention_max_storage_locally", plan.RetentionMaxStorageLocal)
	checkInt("retention_amount_s3", plan.RetentionAmountS3)
	checkInt("retention_days_s3", plan.RetentionDaysS3)
	checkFloat("retention_max_storage_s3", plan.RetentionMaxStorageS3)
	return missing
}
